# ADR-004 — Workflow (Visual Builder + Persistence + Engine Wiring)

Status: Proposed
Tanggal: 2026-07-06
Penulis: System Architect (Zosmed)
Scope: Menjadikan **workflow** (trigger→filter→action) bisa dibuat/disimpan pengguna dan **dijalankan oleh engine netral yang SUDAH ADA** (`libs/workflow`) saat event webhook masuk. Bukan menulis ulang engine.
Referensi: CLAUDE.md §4 (batasan IG), §5/§5a (arsitektur/monorepo), §6 (domain), §7 (katalog node feasible), §8 (engine netral + Kit), §10 (safety), §12a (prinsip coding). Bangun di atas ADR-001 (comment-to-order), ADR-002 (Instagram Login token), ADR-003 (auth/onboarding).

---

## 0. Ringkasan Keputusan

Engine netral `libs/workflow` sudah lengkap dan teruji (`engine.go`, `node.go`, `gate.go`, `event.go`, `context.go` + 741 baris test). Modelnya: `Engine.Run(event, sender, gate)` mengevaluasi `[]WorkflowDef{ID, TriggerKeys, FilterKeys, ActionKeys}` terhadap `Registry` berisi node yang di-`Register*`. Trigger = OR, Filter = AND, Action = urut. Yang **belum ada**: persistence workflow, API CRUD, jembatan dari graf tersimpan → `WorkflowDef`, dan penulisan run log.

Keputusan inti iterasi ini:

1. **Node type ≠ node instance.** `Registry` engine sekarang menyimpan implementasi node sebagai singleton (seller kit). Untuk mendukung banyak workflow/banyak instance per tipe dengan config berbeda, kita perkenalkan **factory per node-type** dan **compiler**: graf tersimpan → registry berisi instance yang di-bind config → `WorkflowDef`. **Kunci registry per-run = UUID node** (bukan tipe), jadi tidak ada tabrakan key dan config tertangkap di dalam instance. **Engine tidak berubah sama sekali.**
2. **Compiler & node library netral segmen.** Node feasible §7 yang netral (keyword-match, comment-received, whatsapp-link, dst.) hidup di paket baru **`libs/workflow/nodes`** (masih netral — tidak tahu segmen). Node spesifik segmen (seller.reserve, trust-kit) tetap di `libs/kits/*`. Compiler (`libs/workflow/compile.go`) netral: ia hanya memetakan `node_type` → factory yang diberikan wiring startup. Peta factory (yang tahu seller kit) dirakit di `apps/worker` — boleh, karena `apps/*` memang mengimpor Kit.
3. **Loader menggantikan slice hardcoded.** `runner.CommentToOrderWorkflow` (hardcoded) diganti: worker memuat workflow `live` milik akun dari DB, compile, jalankan. **Fallback transisional**: bila akun belum punya workflow `live`, jalankan built-in comment-to-order lama supaya slice ADR-001 tidak putus selama rollout.
4. **Run log ditulis worker.** `RunResult.Steps` diserialisasi ke tabel `workflow_run` setelah `Engine.Run`, untuk layar Runs.
5. **Guardrail feasibility ditegakkan di validasi.** `node_type` hanya boleh dari **katalog feasible §7**. Tipe apa pun yang menyentuh DO-NOT list §4b (follower trigger, blast, auto-follow, IG Live) **tidak** ada di katalog → otomatis tertolak saat save/activate. Semua outbound tetap lewat `rc.Gate` (§10). Instagram Login only (`graph.instagram.com`, §4.0) — tidak ada permukaan API baru.

### Acceptance Criteria (Definition of Done — §14)

1. Pengguna terautentikasi (ADR-003) bisa **buat, baca, update (simpan graf), hapus** workflow milik akunnya via REST `/api/v1/workflows`.
2. Workflow tersimpan sebagai `workflow` + `workflow_node` + `workflow_edge`; save mengganti node/edge secara **transaksional**.
3. **Activate** (`live`) hanya lolos bila graf valid: ≥1 trigger, ≥1 action, semua `node_type` ada di katalog feasible §7, edge membentuk DAG. **Pause** mengembalikan ke `paused`.
4. Saat event `comments` masuk, worker memuat workflow `live` akun, **compile** ke `WorkflowDef`, dan menjalankan **engine yang sudah ada** — tanpa perubahan pada `libs/workflow/engine.go`.
5. Setiap run yang ter-trigger menulis satu baris `workflow_run` (status, steps JSONB, durasi); layar Runs membacanya via `/api/v1/workflows/{id}/runs`.
6. Tidak ada `node_type` di katalog yang melanggar §4b. Comment trigger berbasis post/Reel, BUKAN IG Live (§4b.4–5).
7. Semua aksi outbound IG melewati `rc.Gate.Allow` (§10) — tidak ada pengiriman yang mem-bypass safety layer.
8. Engine (`libs/workflow` core), compiler, dan node library `libs/workflow/nodes` **netral segmen**; istilah keep/C/produk/trust hanya di `libs/kits/*`.
9. FE builder (`apps/web/app/(app)/workflows`) tersambung ke backend: list, load graf ke canvas, save draft, publish/pause, dan Runs — menggantikan `@/lib/mock`.
10. Kontrak API selaras `packages/types/src/domain.ts` (`Workflow`, `WorkflowNode`, `WorkflowEdge`, `WorkflowStatus`, `RunLog`) — diperbarui dalam commit yang sama bila sisi Go berubah.

### Non-Scope (sengaja ditunda)

- Editing graf real-time/kolaboratif, undo/redo, versioning riwayat penuh (hanya kolom `version` untuk cache-busting).
- Caching engine ter-compile per (account, version) — iterasi 1 compile per event (volume MVP aman). Ditandai sebagai optimasi lanjutan (§Risiko R3).
- Node "Test run" / simulator di UI (tombol boleh ada, non-fungsional).
- Cakupan node §7 **penuh**. Iterasi 1 hanya subset yang bisa dieksekusi (§7 di bawah). Sisanya = roadmap node, bukan blocker.
- Trigger DM/story sebagai jalur ingest baru (belum ada task ingest DM). Iterasi 1 tetap dari webhook `comments` yang sudah ada. Node trigger dm/story boleh tampil di palette tapi belum bisa `live` (validasi menolak workflow yang trigger-nya belum didukung runtime).
- Schedule/cron trigger, notify opt-in, AI reply node (butuh AI service terpisah).

---

## 1. Model Eksekusi — Jembatan Graf ↔ Engine (keputusan arsitektur utama)

```
                         STARTUP (apps/worker)
  ┌───────────────────────────────────────────────────────────────┐
  │  nodes.RegisterFactories(fmap)   // libs/workflow/nodes (netral)│
  │  seller.RegisterFactories(fmap, svc, waPhone)  // libs/kits/... │
  │  compiler := workflow.NewCompiler(fmap)                         │
  └───────────────────────────────────────────────────────────────┘
                                  │  fmap: map[node_type] -> Factory
                                  ▼
       PER EVENT (comment:ingest handler, apps/worker/internal/tasks)
  ┌───────────────────────────────────────────────────────────────┐
  │ 1. loader.LoadLive(accountID) -> []PersistedWorkflow (dari DB) │
  │ 2. for each pwf: compiler.Compile(pwf)                          │
  │        -> (registry berisi instance ber-config,                │
  │            WorkflowDef{ID:wf.uuid, TriggerKeys/Filter/Action    │
  │                        = UUID node per kategori, terurut})      │
  │ 3. eng := workflow.NewEngine(registry, defs)   // ENGINE LAMA   │
  │ 4. res := eng.Run(ctx, event, sender, rc.Gate) // TAK BERUBAH   │
  │ 5. runStore.Insert(res)  -> workflow_run                        │
  └───────────────────────────────────────────────────────────────┘
```

**Factory** = fungsi yang menerima config JSON satu node dan mengembalikan `workflow.NodeEntry` siap-pakai:

```go
// libs/workflow/compile.go  (NETRAL — tidak tahu segmen)
type Factory struct {
    Category workflow.NodeKind // KindTrigger/KindFilter/KindAction
    Build    func(cfg json.RawMessage) (workflow.NodeEntry, error)
}
type FactoryMap map[string]Factory // key = node_type (katalog §7)

type Compiler struct{ f FactoryMap }
func NewCompiler(f FactoryMap) *Compiler { ... }

// Compile memetakan graf tersimpan -> registry + def.
// - Registry.Register* dipanggil dengan key = node.ID (UUID) agar unik per instance.
// - TriggerKeys/FilterKeys/ActionKeys dikelompokkan per Category,
//   ActionKeys diurutkan topologis dari edges (fallback: position_x lalu created_at).
func (c *Compiler) Compile(pwf PersistedWorkflow) (*workflow.Registry, workflow.WorkflowDef, error)
```

**Kenapa key = UUID instance, bukan node_type?** Karena `Registry.mustRegister` panic saat key duplikat, dan satu workflow bisa punya dua "keyword-match" berbeda config. UUID instance menjamin unik dan config tertangkap di closure factory — nol perubahan pada engine/`WorkflowDef` (yang memang cuma butuh key string).

**Config mengalir lewat instance, konteks runtime lewat `Event.Raw`.** Bedakan dua hal:
- **Config node** (mis. daftar keyword, template teks) → di-bind saat `Compile` (closure factory). Statis per workflow.
- **Konteks runtime** (mis. `catalog_post_id`, `comment_at`, `ig_user_id`) → tetap ditulis handler ingest ke `Event.Raw`, seperti sekarang (lihat `comment_ingest.go` + `seller.RawKey*`). Tidak berubah.

Engine core (`libs/workflow/engine.go`, `node.go`, `context.go`, `gate.go`, `event.go`) **TIDAK diubah**. Yang baru semata `compile.go` (netral) + `libs/workflow/nodes/*` (netral) + factory registrasi di Kit + loader/runstore di worker.

---

## 2. Skema DB (desain migrasi goose — bukan file final)

Ikuti gaya migrasi existing (`00003_reservations.sql`): enum eksplisit, index hot-path, catatan sinkronisasi ke `packages/types`. Nomor migrasi berikutnya: **`00011`–`00014`** (atau digabung; pisah agar `down` rapi).

### 2.1 `workflow_status` enum + tabel `workflow`

```sql
-- 00011_workflow.sql
CREATE TYPE workflow_status AS ENUM ('draft', 'live', 'paused', 'error');
-- Sinkron dengan packages/types/src/domain.ts WorkflowStatus.

CREATE TABLE workflow (
    id          uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid            NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    name        text            NOT NULL,
    status      workflow_status NOT NULL DEFAULT 'draft',
    segment     text            NOT NULL CHECK (segment IN ('seller','creator','booking')),
    -- segment hanya untuk pilihan preset/palette & AI persona; engine ABAIKAN ini (netral).
    version     int             NOT NULL DEFAULT 1,   -- bump saat publish; untuk cache-bust & run_log snapshot
    created_at  timestamptz     NOT NULL DEFAULT now(),
    updated_at  timestamptz     NOT NULL DEFAULT now()
);
CREATE INDEX workflow_account_id_idx ON workflow(account_id);
-- Hot path loader: hanya workflow live per akun.
CREATE INDEX workflow_live_idx ON workflow(account_id) WHERE status = 'live';
```

### 2.2 `workflow_node`

```sql
-- 00012_workflow_node.sql
CREATE TABLE workflow_node (
    id           uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id  uuid  NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    category     text  NOT NULL CHECK (category IN ('trigger','filter','action')),
    node_type    text  NOT NULL,     -- katalog feasible §7 (divalidasi di app-layer terhadap katalog)
    config       jsonb NOT NULL DEFAULT '{}'::jsonb,
    position_x   int   NOT NULL DEFAULT 0,
    position_y   int   NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workflow_node_workflow_id_idx ON workflow_node(workflow_id);
```

Catatan: `node_type` tidak di-CHECK di DB (katalog berevolusi di kode) — validasi terhadap katalog feasible dilakukan di handler save/activate (single source: `libs/workflow/nodes` catalog + mirror `packages/types`).

### 2.3 `workflow_edge`

```sql
-- 00013_workflow_edge.sql
CREATE TABLE workflow_edge (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id  uuid NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    from_node_id uuid NOT NULL REFERENCES workflow_node(id) ON DELETE CASCADE,
    to_node_id   uuid NOT NULL REFERENCES workflow_node(id) ON DELETE CASCADE,
    UNIQUE (workflow_id, from_node_id, to_node_id)
);
CREATE INDEX workflow_edge_workflow_id_idx ON workflow_edge(workflow_id);
```

### 2.4 `workflow_run` (run log)

```sql
-- 00014_workflow_run.sql
CREATE TABLE workflow_run (
    id             uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id    uuid  REFERENCES workflow(id) ON DELETE SET NULL, -- riwayat tetap ada bila wf dihapus
    workflow_name  text  NOT NULL DEFAULT '',   -- snapshot untuk tampilan riwayat
    account_id     uuid  NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    trigger_source text  NOT NULL DEFAULT '',   -- 'comment' | 'dm' | 'story'
    trigger_summary text NOT NULL DEFAULT '',   -- mis. "comment by @rina_susanti"
    object_id      text  NOT NULL DEFAULT '',   -- ig comment/message id (dedupe/trace)
    status         text  NOT NULL CHECK (status IN ('success','failed','skipped')),
    triggered      bool  NOT NULL DEFAULT false,
    filter_passed  bool  NOT NULL DEFAULT false,
    steps          jsonb NOT NULL DEFAULT '[]'::jsonb, -- serialisasi []workflow.StepLog
    error          text  NOT NULL DEFAULT '',
    duration_ms    int   NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workflow_run_account_created_idx ON workflow_run(account_id, created_at DESC);
CREATE INDEX workflow_run_workflow_created_idx ON workflow_run(workflow_id, created_at DESC);
```

Mapping status: `RunResult.Err != nil` → `failed`; `Triggered && Err == nil` → `success`; `!Triggered` → `skipped` (opsional: skip write untuk mengurangi noise — lihat R2).

---

## 3. Kontrak API

Semua di bawah `RequireUser` (ADR-003). **Scoping akun**: workflow milik akun IG pengguna; handler resolve akun via `GetAccountByUserID(userID)` (bukan `accountId` query param) — lebih aman dari pola lama commentorder. Semua respons pakai `httpx.Envelope` `{data, error}`.

Tipe DTO diselaraskan dengan `packages/types/src/domain.ts` (sudah ada `Workflow`, `WorkflowNode`, `WorkflowEdge`, `NodeKind`, `WorkflowStatus`, `RunLog`, `RunStatus`). Tambahkan DTO request/summary baru di `packages/types/src/workflow.ts` (file baru) + mirror struct Go di `apps/api/internal/workflow/dto.go`.

| Method | Path | Body / Query | Response `data` |
| --- | --- | --- | --- |
| GET | `/api/v1/workflows` | — | `WorkflowSummary[]` |
| POST | `/api/v1/workflows` | `CreateWorkflowRequest{name, segment}` | `Workflow` (graf kosong / preset) |
| GET | `/api/v1/workflows/{id}` | — | `Workflow` (nodes+edges+config) |
| PUT | `/api/v1/workflows/{id}` | `SaveWorkflowRequest{name, nodes[], edges[]}` | `Workflow` |
| DELETE | `/api/v1/workflows/{id}` | — | `{deleted: true}` |
| POST | `/api/v1/workflows/{id}/activate` | — | `Workflow` (status=`live`) atau 422 `validation_failed` |
| POST | `/api/v1/workflows/{id}/pause` | — | `Workflow` (status=`paused`) |
| GET | `/api/v1/workflows/{id}/runs` | `?limit=50` | `RunSummary[]` |
| GET | `/api/v1/runs` | `?limit=50` | `RunSummary[]` (semua workflow akun; layar Runs global) |
| GET | `/api/v1/node-catalog` | — | `NodeCatalogEntry[]` (opsional; boleh statik FE) |

Shape DTO baru (TS; struct Go mengikuti, field JSON snake→camel via tag):

```ts
// packages/types/src/workflow.ts
export interface WorkflowSummary {
  id: Id; name: string; status: WorkflowStatus; segment: Segment;
  nodeCount: number; updatedAt: ISODateTime;
}
export interface CreateWorkflowRequest { name: string; segment: Segment; }
export interface SaveWorkflowRequest {
  name: string;
  nodes: WorkflowNode[];   // reuse domain.ts (id boleh baru/kosong -> server assign)
  edges: WorkflowEdge[];
}
export interface RunSummary {
  id: Id; workflowId: Id | null; workflowName: string;
  triggerSummary: string; status: RunStatus; // 'success'|'failed'|'skipped'
  durationMs: number; steps: RunStepDTO[]; at: ISODateTime;
}
export interface RunStepDTO { nodeKey: string; kind: 'trigger'|'filter'|'action'; status: string; detail: string; }
// NodeCatalogEntry: { category, nodeType, label, iconKey, runnable, configSchema? }
```

Validasi `activate` (422 `validation_failed` + `reason`):
- `no_trigger` — 0 node kategori trigger.
- `no_action` — 0 node kategori action.
- `unknown_node_type` — ada `node_type` di luar katalog feasible (menegakkan §4b).
- `trigger_not_runnable` — trigger tipe belum didukung runtime iterasi ini (mis. `dm-received`).
- `cycle` — edge tidak membentuk DAG.

---

## 4. Wiring Engine (dari DB → engine yang sudah ada)

### 4.1 Paket baru & yang disentuh

```
libs/workflow/
├── compile.go            # BARU (NETRAL): Factory, FactoryMap, Compiler, PersistedWorkflow, topo-sort
├── nodes/                # BARU (NETRAL): implementasi node feasible §7 yang tak tahu segmen
│   ├── catalog.go        #   daftar node_type + kategori + "runnable" (single source; mirror packages/types)
│   ├── trigger_comment.go#   comment-received (+ filter post opsional dari config)
│   ├── filter_keyword.go #   keyword-match (config: {keywords[], caseInsensitive})
│   └── action_wa_link.go #   send-whatsapp-link: 1 private-reply berisi wa.me (GATED via rc.Gate)
│
libs/kits/seller/
└── factories.go          # BARU: RegisterFactories(fmap, svc, waPhone) untuk node_type seller.* (reserve, private-reply)
│
apps/worker/internal/
├── runner/runner.go      # UBAH: rakit FactoryMap (nodes + seller), simpan *Compiler + WorkflowLoader + RunStore
├── wfload/loader.go      # BARU: LoadLive(accountID) -> []workflow.PersistedWorkflow (via dbgen)
├── wfload/runstore.go    # BARU: Insert(RunResult, meta) -> workflow_run
└── tasks/comment_ingest.go # UBAH: load->compile->NewEngine->Run->runstore.Insert (fallback built-in bila kosong)
```

### 4.2 Alur di `comment_ingest.go` (ubah minimal)

Handler ingest saat ini sudah: deteksi keep-code, cek catalog_post, load account/token, isi `Event.Raw`, panggil `h.r.Engine.Run`. Perubahan:

1. Ganti `h.r.Engine` (statis) → `defs, reg := loadAndCompile(accountID)`; bila `len(defs)==0` pakai built-in `CommentToOrderWorkflow` + registry seller lama (fallback transisional).
2. `eng := workflow.NewEngine(reg, defs)`; `res, err := eng.Run(ctx, event, sender, h.r.Gate)`.
3. `h.r.RunStore.Insert(ctx, res, runMeta{accountID, triggerSummary:"comment by @"+p.FromUsername, source:"comment", objectID:p.CommentID, dur})`.

Catatan penting yang menjaga slice ADR-001 tetap hidup:
- Deteksi keep-code + lookup catalog + isi `Event.Raw` **tetap** dilakukan handler (pre-screen), karena node seller mengandalkan `Event.Raw` (`RawKeyCatalogPostID`, dst.). Untuk workflow generik (non-seller), pre-screen ini tidak merusak apa pun (node generik tidak baca key seller).
- Trigger `comment-received` generik hanya cek `Source == comment`; filter/action generik jalan tanpa `Event.Raw` seller.

### 4.3 Compile: pemetaan graf → `WorkflowDef`

- Kelompokkan node per `category`. `TriggerKeys = [node.id trigger…]`, `FilterKeys = [node.id filter…]`, `ActionKeys = [node.id action…]`.
- Urutan `ActionKeys`: topological sort atas `workflow_edge` yang dibatasi pada node action; jika tak ada edge antar action, fallback `position_x` lalu `created_at`. (Engine menjalankan action berurutan — urutan penting; trigger/filter tidak sensitif urutan karena OR/AND.)
- Tiap node → `fmap[node.node_type].Build(node.config)` → `NodeEntry`; register ke `Registry` dengan key `node.id`.
- Guard: `Build` error / `node_type` tak dikenal → Compile gagal, workflow di-skip + log; tidak menjatuhkan event lain.

### 4.4 Guardrail yang wajib tetap terpasang

- **§10 one-door**: action outbound (mis. `action_wa_link`) **wajib** `rc.Gate.Allow(...)` sebelum `rc.Sender.*`, meniru persis `seller.privateReplyAction` (Allow→send, Queue→defer, Reject→skip). Reviewer menolak PR node action yang memanggil `rc.Sender` tanpa gate.
- **§4b**: katalog `nodes.Catalog` **tidak memuat** follower-trigger, blast, auto-follow, atau IG-Live. `comment-received` didokumentasikan sebagai post/Reel (webhook `comments`), bukan Live.
- **§4.0**: tidak ada permukaan API baru; hanya `graph.instagram.com` yang sudah dipakai `libs/igapi`.

---

## 5. Katalog Node — cakupan iterasi ini

Palette FE boleh menampilkan **seluruh** §7 (sudah ada di `mock/workflows.ts`), tetapi hanya subset yang **runnable** (bisa `live`). `catalog.go` menandai `Runnable bool`; validasi `activate` menolak workflow yang memakai node non-runnable sebagai satu-satunya jalur.

| node_type | category | Paket | Iterasi 1 |
| --- | --- | --- | --- |
| `comment-received` | trigger | `libs/workflow/nodes` | ✅ runnable |
| `comment-to-order` | trigger | `libs/kits/seller` | ✅ runnable (existing) |
| `keyword-match` | filter | `libs/workflow/nodes` | ✅ runnable |
| `send-whatsapp-link` | action | `libs/workflow/nodes` | ✅ runnable (GATED) |
| `reserve-stock` | action | `libs/kits/seller` | ✅ runnable (existing) |
| `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad` | trigger | — | ⛔ palette-only (belum ada ingest) |
| `conversation-state`, `intent`, `post-selection`, `time-window` | filter | — | ⛔ roadmap |
| `reply-comment`, `send-dm`, `ai-reply`, `send-trust-kit`, `notify-optin`, `handoff-human`, `tag-contact`, `outbound-webhook` | action | — | ⛔ roadmap |

Alasan subset: `reply-comment`/`send-dm` publik butuh perluasan gate (Kind `comment-reply` + counter per-post) — di luar scope agar tidak menyentuh `libs/safety`. `send-whatsapp-link` memakai jalur private-reply yang **sudah** didukung gate (identik seller), jadi aman. Node lain (AI, opt-in, trust-kit) butuh service yang belum ada.

> Menambah node = tambah entri katalog + factory, tanpa menyentuh engine (§8). Node netral → `libs/workflow/nodes`; node segmen → `libs/kits/<segmen>`.

---

## 6. Pembagian Kerja (2 agen paralel)

### 6.1 go-backend-engineer

Urut sesuai dependency (§7). Path absolut dari root repo.

- **B1. Migrasi DB** — `db/migrations/00011_workflow.sql` … `00014_workflow_run.sql` (skema §2). `goose up`.
- **B2. Query sqlc** — `db/query/workflow.sql`: `CreateWorkflow`, `GetWorkflowByID`, `ListWorkflowsByAccount`, `ListLiveWorkflowsByAccount`, `UpdateWorkflowMeta`, `SetWorkflowStatus` (guard `WHERE status=@expected` bila perlu), `DeleteWorkflow`, `ListNodesByWorkflow`, `ListEdgesByWorkflow`, `InsertNode`, `InsertEdge`, `DeleteNodesByWorkflow`, `DeleteEdgesByWorkflow`, `InsertRun`, `ListRunsByWorkflow`, `ListRunsByAccount`. `sqlc generate`.
- **B3. Compiler netral** — `libs/workflow/compile.go`: `Factory`, `FactoryMap`, `PersistedWorkflow` (+ `PersistedNode`/`PersistedEdge`), `Compiler.Compile`, topo-sort action. Tambah `libs/workflow` sudah di `go.work`. Unit test: dua keyword-match beda config → dua instance, urutan action dari edge, error node_type tak dikenal.
- **B4. Node library netral** — `libs/workflow/nodes/{catalog.go,trigger_comment.go,filter_keyword.go,action_wa_link.go}` + `RegisterFactories(fmap FactoryMap)`. `action_wa_link` **wajib** pola gate (Allow/Queue/Reject) meniru `seller.privateReplyAction`. Tambah `./libs/workflow/nodes` ke `go.work` bila jadi module terpisah (atau sub-package `libs/workflow` — pilih sub-package agar tak perlu module baru).
- **B5. Seller factories** — `libs/kits/seller/factories.go`: `RegisterFactories(fmap, svc *ReservationService, waPhone string)` untuk `comment-to-order`/`reserve-stock`/`seller.private-reply`, membungkus node existing (`RegisterNodes` di-refactor agar berbagi konstruksi instance — DRY §12a-1).
- **B6. Loader + RunStore** — `apps/worker/internal/wfload/loader.go` (`LoadLive` → `[]workflow.PersistedWorkflow` via dbgen) + `wfload/runstore.go` (`Insert(res, meta)` → `workflow_run`, serialisasi `[]StepLog`).
- **B7. Wire runner + ingest** — `runner.go`: rakit `FactoryMap` (nodes+seller), simpan `*Compiler`, `*wfload.Loader`, `*wfload.RunStore`. `comment_ingest.go`: load→compile→`NewEngine`→`Run`→`RunStore.Insert`, fallback built-in bila kosong (§4.2).
- **B8. API handler** — `apps/api/internal/workflow/{handler.go,dto.go,mapping.go}`: semua endpoint §3, scoping via `GetAccountByUserID`, validasi activate (§3). Reuse `httpx.JSON/Err/ErrWithReason`, `uuidx`, pola `parseUUIDParam`. Katalog feasible untuk validasi = import dari `libs/workflow/nodes` (single source).
- **B9. Router + main** — `apps/api/internal/httpx/router.go`: tambah field `http.HandlerFunc` untuk tiap endpoint di grup `RequireUser`. `apps/api/cmd/api/main.go`: `workflow.NewHandler(queries)` + isi `Routes`.
- **B10. Seed preset seller** — saat `PutSegment`/`CompleteOnboarding` segment=`seller` (atau via `cmd/seed`), buat satu `workflow` seller comment-to-order `live` sehingga loader punya isi & fallback bisa dipensiunkan. (Koordinasi dengan auth handler — opsional, boleh menyusul.)
- **B11. Tests** — compiler unit (B3), node gate (B4), handler validasi activate (B8, fakedb pola `auth/fakedb_test.go`).

### 6.2 frontend-ui-engineer

- **F1. Tipe kontrak** — `packages/types/src/workflow.ts` (DTO §3) + export di `packages/types/src/index.ts`. Pastikan selaras `domain.ts` (jangan duplikasi `Workflow`/`WorkflowNode`).
- **F2. Data layer** — `apps/web/lib/api/workflows.ts`: fetch typed (list/get/create/save/activate/pause/delete/runs) memakai envelope `{data,error}`. Ganti `getWorkflowBuilder`/`getWorkflowRuns` dari `@/lib/mock/api` (pola sama seperti data-fetching existing).
- **F3. List workflows** — layar/daftar workflow (index) memakai `WorkflowSummary[]` + tombol "New workflow" (POST create → redirect ke builder). (Sidebar key `workflows` §9.)
- **F4. Builder tersambung** — `apps/web/app/(app)/workflows/page.tsx` + `_components/{Palette,FlowCanvas,InspectorCanvas}.tsx`: muat `Workflow` (nodes+edges+config) ke canvas; drag dari `Palette` (katalog `NodeCatalogEntry`, tandai non-runnable disabled/badge); **Save draft** → PUT; **Publish** → POST activate (tampilkan error validasi 422 dengan `reason`); **Pause** → POST pause. State canvas → hook (`useWorkflowGraph`) memisah presentational vs data (SoC §12a-3).
- **F5. Inspector config** — panel kanan (`page.tsx` bagian inspector + `inspector/`): edit `config` per node terpilih sesuai `node.node.kind` (mis. keyword-match → daftar keyword; send-whatsapp-link → template + nomor WA). Simpan ke state graf (bukan fetch di JSX).
- **F6. Runs** — `apps/web/app/(app)/workflows/runs/page.tsx`: ganti `mockRuns` dengan `RunSummary[]` dari `/api/v1/workflows/{id}/runs` (atau `/runs`). Render `steps` ke Step Timeline. Pill status `success/failed/skipped` (bukan `review`). Pill "● LIVE" = workflow live, BUKAN IG Live (§9).
- **F7. Palette dari katalog** — jadikan `Palette`/`NODE_COLORS` mengonsumsi katalog feasible (bisa statik dari `packages/types` mirror `catalog.go`), hapus item yang melanggar §4b bila ada. Node non-runnable diberi badge "segera".
- **F8. Guard copy** — semua label default Bahasa Indonesia gaya olshop, token desain §11 (lime/dark), mono untuk label teknis.

---

## 7. Urutan & Dependency

```
B1 migrasi ─► B2 sqlc ─┬─► B6 loader/runstore ─┐
                       │                        ├─► B7 wire runner+ingest ─► (jalur runtime hidup)
B3 compiler ───────────┼─► B4 nodes ────────────┤
                       │   B5 seller factories ─┘
                       └─► B8 API handler ─► B9 router+main ─► (jalur CRUD hidup)
B10 seed (setelah B8/B9)      B11 tests (menempel tiap paket)

FE: F1 (butuh kesepakatan DTO §3, tidak blocking backend) ─► F2 ─┬─► F3 list
                                                                 ├─► F4 builder ─► F5 inspector
                                                                 └─► F6 runs ; F7/F8 paralel
```

Blocking utama:
- **B1→B2** memblokir semua kerja DB-backed.
- **B3 (compiler) & B4 (nodes)** memblokir B7 (runtime). B3 & B4 bisa paralel setelah kesepakatan `Factory`/`PersistedWorkflow`.
- **B8→B9** memblokir jalur CRUD; **F2** butuh B8/B9 up (FE bisa mulai F1 + UI dengan mock sementara).
- **DTO §3 (F1)** sebaiknya disepakati backend↔frontend di awal (kontrak selaras, §12a) — lakukan sebelum B8 & F2 agar tak revisi ganda.

Titik integrasi aman: backend bisa selesaikan B1–B7 (runtime) tanpa FE; FE bisa bangun F1/F3/F4 di atas mock lalu tukar ke `lib/api` saat B8/B9 siap.

---

## 8. Risiko / Keputusan Terbuka

- **R1 — Compile per event vs cache.** Iterasi 1 compile registry+engine tiap event (sederhana, aman untuk volume MVP). Bila traffic komentar tinggi, ini overhead. **Keputusan diminta:** setujui compile-per-event sekarang + tiket optimasi cache per `(account_id, version)` nanti? (Rekomendasi: ya.)
- **R2 — Log run "skipped".** Menulis `workflow_run` untuk event yang tak nge-trigger membanjiri tabel. **Rekomendasi:** hanya tulis saat `Triggered==true`; event non-match cukup log slog. Setujui?
- **R3 — Fallback built-in comment-to-order.** Fallback menjaga ADR-001 saat rollout, tapi menambah satu cabang. **Keputusan:** hapus fallback begitu B10 (seed preset seller) aktif, atau pertahankan sebagai jaring pengaman? (Rekomendasi: hapus setelah seed terpasang & terverifikasi.)
- **R4 — Save graf = replace vs diff.** PUT mengganti seluruh node/edge transaksional (delete+insert). Sederhana, tapi mengubah `node.id` tiap save → run_log lama kehilangan referensi node (kita snapshot `workflow_name`, bukan node). **Keputusan:** terima replace (rekomendasi, karena builder simpan seluruh canvas), atau perlu id node stabil lintas save? Jika stabil diperlukan → FE kirim id node existing dan server upsert.
- **R5 — Scoping via `GetAccountByUserID` (MVP 1 user⇄1 akun).** Bila nanti multi-akun per user (ADR-003 §2.2 note), endpoint workflow butuh `accountId` eksplisit. **Keputusan:** kunci ke satu akun untuk iterasi ini? (Rekomendasi: ya, konsisten dengan `/auth/me`.)
- **R6 — Trigger non-runnable di palette.** Menampilkan `dm-received`/story di palette tapi menolak `activate` bisa membingungkan. **Keputusan:** tampilkan dengan badge "segera" (rekomendasi) atau sembunyikan sampai ingest DM ada?
- **R7 — `node-catalog` endpoint vs konstanta.** Katalog bisa endpoint (dinamis) atau konstanta TS mirror `catalog.go` (statis, DRY lintas bahasa seperti `KIT_KEYWORDS`). **Rekomendasi:** konstanta mirror (tanpa endpoint) untuk iterasi 1.

---

## 9. Guardrail Ringkas (verifikasi PR)

- Engine `libs/workflow/{engine,node,context,gate,event}.go` **tidak diubah** (grep diff harus kosong untuk file ini).
- `libs/workflow/compile.go` + `libs/workflow/nodes/*` **tidak mengimpor** `libs/kits/*` (netral segmen §8).
- Tiap action outbound memanggil `rc.Gate.Allow` sebelum `rc.Sender.*` (§10 one-door).
- `nodes.Catalog` tidak memuat node yang melanggar §4b (follower/blast/auto-follow/IG-Live).
- Hanya `graph.instagram.com` (§4.0) — tidak ada referensi `graph.facebook.com`.
- `packages/types` diperbarui seiring struct Go (kontrak selaras, §12a).
```
