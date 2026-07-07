# ADR-005 — Workflow Builder Completion + UX Fleksibel

Status: Proposed
Tanggal: 2026-07-07
Penulis: System Architect (Zosmed)
Scope: (1) Menaikkan jumlah node **runnable** dari katalog §7 dengan menambah node netral yang infrastrukturnya sudah ada, dan **membuka jalur runtime** agar node comment-triggered generik benar-benar tereksekusi. (2) Membuat UX builder fleksibel: drag posisi manual, buat/hapus edge di canvas, inspector generik berbasis config-schema, dan validasi inline. **Tidak** menulis ulang engine.
Referensi: CLAUDE.md §4 (batasan IG), §7 (katalog node feasible), §8 (engine netral + Kit), §10 (safety), §12a (prinsip coding), §14 (DoD). Membangun di atas ADR-004 (workflow builder + persistence + engine wiring).

---

## 0. Ringkasan Keputusan

ADR-004 sudah menghadirkan: persistence (`workflow`/`workflow_node`/`workflow_edge`/`workflow_run`), compiler netral (`libs/workflow/compile.go`), node library netral (`libs/workflow/nodes`), factory Kit (`libs/kits/seller/factories.go`), loader + runstore di worker, dan builder FE yang tersambung ke backend. **5 node runnable**: `comment-received`, `comment-to-order`, `keyword-match`, `send-whatsapp-link`, `reserve-stock`. Engine core (`engine.go`, `node.go`, `context.go`, `gate.go`, `event.go`) tidak akan disentuh (guardrail §9 ADR-004, dipertahankan di sini).

Dua keluhan pengguna yang ADR ini selesaikan:

1. **16 node non-runnable.** ADR ini mengaudit ke-16 node, mengklasifikasikan **(A) buildable sekarang / (B) butuh subsystem / (C) tidak feasible**, lalu membangun subset (A) yang bernilai tinggi tanpa menyentuh engine.
2. **UX builder kaku.** ADR ini merancang: drag posisi manual, edge create/delete di canvas, inspector **config-schema-driven** (bukan `switch` hardcoded 2 tipe), dan validasi inline yang menampilkan alasan 422 di canvas.

### Temuan arsitektur kunci (menentukan scope backend)

**Jalur ingest saat ini terkunci ke pre-screen seller.** Sebuah workflow generik `[comment-received → reply-comment]` **tidak akan pernah jalan** pada komentar biasa, karena dua gerbang seller memblokirnya sebelum engine:

- `apps/api/internal/webhook/handler.go` (`processComment`, step 6): komentar **hanya di-enqueue** bila `media_id` ada di **catalog_post aktif**. Komentar di post non-katalog tidak pernah sampai ke worker.
- `apps/worker/internal/tasks/comment_ingest.go` (step 2): **early-return** bila `DetectKeepCode` gagal. Komentar tanpa kode keep/C berhenti di sini.

Artinya: node seperti `reply-comment` **feasible di level node** (gate `KindCommentReply` + `Sender.ReplyToComment` sudah ada — dikonfirmasi), tetapi **tidak reachable di runtime**. Karena itu tugas backend paling berleverage di ADR ini adalah **decoupling pipeline ingest** dari pre-screen seller, bukan sekadar menambah node.

**Yang SUDAH beres (tidak perlu kerja backend):** persistensi `position_x/y` sudah ditulis `store.go` (`InsertNode`), dan edge diremap via `ClientID` saat Save (R4 ADR-004). Jadi **drag-persist dan edge-persist tidak butuh perubahan backend** — FE hanya belum memunculkan interaksinya. Ini menyempitkan scope backend secara signifikan.

### Acceptance Criteria (DoD §14)

1. Node **(A)** berikut `Runnable=true` di `libs/workflow/nodes/catalog.go` dan tereksekusi engine: `post-selection`, `time-window` (filter); `reply-comment`, `outbound-webhook` (action). Mirror `packages/types` NODE_CATALOG disinkronkan dalam commit yang sama.
2. Workflow generik `[comment-received → reply-comment]` **fire pada komentar apa pun** di akun (post non-katalog, tanpa kode keep), lewat engine yang sama — hasil decoupling ingest (B1).
3. Perilaku seller **tidak berubah**: `comment-to-order` hanya reserve saat kode keep/C terdeteksi **dan** post ada di catalog aktif; komentar biasa tidak memicu reservasi.
4. `reply-comment` memanggil `rc.Gate.Allow` dengan `Kind=KindCommentReply` **sebelum** `rc.Sender.ReplyToComment` (§10 one-door). `outbound-webhook` bukan outbound IG → tanpa gate IG, tetapi diberi guard sendiri (timeout + tolak IP privat/loopback).
5. Builder FE: node bisa **di-drag** (posisi tersimpan), edge bisa **dibuat/dihapus** di canvas, koneksi divalidasi urutan trigger→filter→action + anti-cycle di sisi klien (server tetap otoritatif via activate 422).
6. Inspector merender field dari **config-schema per node_type** (satu `SchemaForm` generik) — menambah node baru **tidak** menulis komponen inspector baru (DRY §12a-1/§12a-3). Node `keyword-match` & `send-whatsapp-link` yang lama pindah ke schema tanpa regresi.
7. Validasi activate 422 (`reason`) ditampilkan inline + node bermasalah di-highlight di canvas.
8. Katalog tetap tidak memuat item §4b (follower/blast/auto-follow/IG-Live). Semua node (A) berdiri di §4a. Copy default Bahasa Indonesia gaya olshop, token desain §11.
9. Engine core `libs/workflow/{engine,node,context,gate,event}.go` **tidak diubah** (grep diff kosong). `libs/workflow/nodes/*` & `compile.go` tidak mengimpor `libs/kits/*`.

### Non-Scope (ditunda — lihat klasifikasi (B) §1)

- Semua trigger non-comment: `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad` → butuh **Messaging/Story Ingest pipeline** (ADR terpisah).
- `conversation-state` (butuh window store), `intent` (AI/atau redundan dgn keyword-match), `send-dm` (butuh window 24h terbuka), `ai-reply` (AI service), `send-trust-kit` (asset store), `notify-optin` (opt-in + scheduler), `handoff-human` (inbox/assignment), `tag-contact` (contacts store).
- Stable node-id lintas save (R4 ADR-004 dipertahankan — replace + remap; posisi & edge sudah persist dengan benar).

---

## 1. Audit 16 Node Terdisable — Klasifikasi A/B/C

Verifikasi feasibility §4b dijalankan lebih dulu: **tidak ada** node yang menyentuh DO-NOT list (new-follower trigger, auto-follow/unfollow, follow-status, IG-Live viewer/komentar-live, blast DM massal, scraping). Katalog memang sudah disaring — **(C) = kosong**, dikonfirmasi. Klasifikasi karena itu hanya soal **infra tersedia (A)** vs **butuh subsystem baru (B)**.

```
  KATEGORI                         KELAS   ALASAN SINGKAT
  ───────────────────────────────────────────────────────────────────────────
  TRIGGERS
   dm-received                      B      Messaging ingest (webhook `messages` belum diparse)
   story-reply                      B      Messaging ingest (story-context di `messages`)
   story-mention                    B      Mentions ingest (field `mentions` belum dilanggan/parse)
   click-to-dm-ad                   B      Ad-referral di `messages` + setup iklan
  FILTERS
   post-selection                  ✅A     Baca Event.MediaID — logika server murni
   time-window                     ✅A     Waktu server — logika server murni
   conversation-state               B      Butuh window/last-interaction store (ikut DM ingest)
   intent (ragu/trust)              B*     AI classifier; versi murah = redundan dgn keyword-match
  ACTIONS
   reply-comment                   ✅A     Gate KindCommentReply + Sender.ReplyToComment ADA
   outbound-webhook                ✅A     HTTP POST — bukan outbound IG, tanpa gate IG
   send-dm                          B      Butuh window 24h TERBUKA (gate reject di flow komentar)
   ai-reply                         B      AI service Go belum ada
   send-trust-kit                   B      TrustAsset store belum ada (seller §8.1.3)
   notify-optin                     B      OptIn table + scheduler + one-time-notification
   handoff-human                    B      Inbox/assignment (Fase 2)
   tag-contact                      B      Contacts store belum ada (kecil, tetap subsystem)
  ───────────────────────────────────────────────────────────────────────────
  (C) tidak feasible / §4b: TIDAK ADA.
```

### (A) Buildable phase ini — 4 node + 1 enabler ingest

| node_type | kategori | Paket | Guardrail kunci |
| --- | --- | --- | --- |
| `post-selection` | filter | `libs/workflow/nodes` | logika server; baca `Event.MediaID`; tanpa outbound |
| `time-window` | filter | `libs/workflow/nodes` | logika server; waktu evaluasi; tanpa outbound |
| `reply-comment` | action | `libs/workflow/nodes` | **wajib** `rc.Gate.Allow(Kind=KindCommentReply)` sebelum `rc.Sender.ReplyToComment`; set `PostID` (cap per-post/5min §4c); window ≤7 hari; Instagram Login only (§4.0) |
| `outbound-webhook` | action | `libs/workflow/nodes` | bukan IG → tanpa gate IG; guard SSRF (tolak loopback/IP privat), timeout, hanya http/https |

Enabler wajib (tanpa ini `reply-comment` tidak reachable): **decoupling ingest** (B1 §5).

Catatan feasibility per node (A):
- `reply-comment` = balas komentar publik `POST /{comment-id}/replies` (§4a ALLOW, §7 "Reply comment"). Dedupe per (account, user, comment) via `TriggerKey=commentID`. **Bukan** blast (§4b.6) — satu balasan per komentar.
- `outbound-webhook` = §7 "Webhook (keluar)" ke backend pengguna — bebas, bukan permukaan IG. Tidak melewati `graph.instagram.com`. Guard SSRF karena URL dikontrol pengguna.
- `post-selection` & `time-window` = §7 filter "Post selection" & "Time window", eksplisit "logika server".

### (B) Ditunda — subsystem yang dibutuhkan (untuk ADR berikutnya)

- **Messaging/Story Ingest pipeline** → membuka `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad`, `conversation-state`, `send-dm`. Butuh: parse field `messages`/`mentions` di `webhook/payload.go`, task ingest baru (`dm_ingest`), Event tanpa ketergantungan catalog_post, dan **window/last-interaction store** (Redis atau tabel `conversation`). `send-dm` khusus: pada flow **komentar** tidak ada window 24h terbuka → gate akan selalu reject; node baru bermakna saat dipicu event DM/story. Menghadirkan `send-dm` sekarang = node yang selalu reject (menyesatkan) → **ditunda bersama ingest**.
- **AI service (Go)** → `ai-reply` dan versi AI dari `intent`. Belum ada (folder `apps/web/.../ai` hanya UI). `intent` versi murah (frasa "real kak?", "ga tipu2", "amanah", "COD") secara teknis = `keyword-match` dengan preset frasa; menambah node terpisah yang near-duplicate melanggar §12a-4 → **rekomendasi: pakai `keyword-match` sebagai interim, tunda `intent` sampai AI classifier ada** (open decision R4).
- **TrustAsset store** → `send-trust-kit` (seller §8.1.3, node segmen di `libs/kits/seller`). Butuh tabel `trust_asset` + upload + trigger intent.
- **OptIn + scheduler** → `notify-optin` (commerce calendar §8.1.5). Butuh tabel `opt_in`, scheduler (asynq cron), one-time-notification sender.
- **Contacts store** → `tag-contact` (kecil, self-contained, tanpa outbound IG — bisa jadi kandidat "quick win" fase berikut).
- **Inbox/assignment** → `handoff-human` (Fase 2 §13).

### (C) Tidak feasible — kosong. Dikonfirmasi terhadap §4b.

---

## 2. Desain node (A) — kontrak & guardrail

Semua node netral hidup di `libs/workflow/nodes` (tidak tahu segmen; tidak impor `libs/kits/*`). Registrasi factory lewat `nodes.RegisterFactories(fmap)` yang sudah ada — cukup tambah entri.

### 2.1 `post-selection` (filter, netral)

```
config: { mediaIds: string[] }         // kosong = permissive (lolos)
Allow(rc): true jika Event.MediaID ∈ mediaIds (atau mediaIds kosong)
```
Catatan: `comment-received` sudah punya opsi inline `mediaId` tunggal. `post-selection` melengkapi kasus multi-post & komposabilitas (§12a-4: bukan duplikasi — trigger memfilter di titik masuk, filter node di rantai). Tidak ada outbound.

### 2.2 `time-window` (filter, netral)

```
config: { days?: number[0-6], startMinute?: int, endMinute?: int, timezone?: string }
Allow(rc): true jika waktu evaluasi (time.Now di tz) masuk hari & rentang menit
```
Default permissive bila config kosong. Gunakan waktu evaluasi (bukan `comment_at`) agar konsisten untuk "hanya jalan saat jam kerja". Tidak ada outbound.

### 2.3 `reply-comment` (action, netral) — GATED

```
config: { template: string }           // placeholder {nama}; default BI olshop
Execute(rc):
  1. bangun OutboundReq{ AccountID, Kind=KindCommentReply, TargetUserID=Event.FromID,
                          TriggerKey=Event.ObjectID, CommentID=Event.ObjectID,
                          CommentAt=Raw[comment_at], PostID=Event.MediaID }
  2. d := rc.Gate.Allow(ctx, req)      // ← ONE-DOOR, sebelum Sender apa pun
  3. Allow  → rc.Sender.ReplyToComment(ctx, Event.ObjectID, text)
     Queue  → dilaporkan ditunda (belum ada retry generik — sama seperti send-whatsapp-link)
     Reject → skipped, dilaporkan
```
Guardrail: gate memakai `KindCommentReply` (cap 750/jam + 30/post/5min §4c) — bukan cap DM. Reviewer menolak PR yang memanggil `rc.Sender.ReplyToComment` tanpa `rc.Gate.Allow`. Instagram Login only (§4.0); tidak ada permukaan API baru — `Sender.ReplyToComment` sudah dipetakan ke `graph.instagram.com` di `libs/igapi`.

### 2.4 `outbound-webhook` (action, netral) — non-IG, guard sendiri

```
config: { url: string, includeSignature?: bool }
Execute(rc):
  1. validasi url: skema http/https; host BUKAN loopback/link-local/IP privat (anti-SSRF)
  2. POST JSON { event, account_id, from, text, media_id, vars } dengan timeout (mis. 5s)
  3. opsional X-Zosmed-Signature (HMAC) bila includeSignature
  4. status non-2xx → ActionResult{Detail:"webhook non-2xx"} (tidak menggagalkan run)
```
Bukan outbound IG → **tidak** lewat `rc.Gate` (gate khusus kuota IG). Tapi wajib guard: timeout + tolak alamat internal (SSRF). Validasi URL final juga saat `Factory.Build`.

---

## 3. Decoupling Ingest (enabler B1) — agar comment-triggered generik reachable

Tujuan: komentar biasa mencapai engine, **tanpa** mengubah perilaku seller.

```
SEBELUM (terkunci seller):
  webhook.processComment ── enqueue HANYA jika media ∈ catalog aktif ──► worker
  comment_ingest ── early-return jika DetectKeepCode gagal ──► (stop)

SESUDAH (dua jalur, satu pipeline):
  webhook.processComment
     enqueue jika (media ∈ catalog aktif)  OR  (akun punya ≥1 workflow live)
                                                └─ query cheap HasLiveWorkflow(accountID)
        │
        ▼
  comment_ingest (TIDAK early-return):
     1. bangun Event dasar (Source=comment) — SELALU
     2. SELALU set Raw[ig_user_id], Raw[comment_at]     (dibutuhkan node netral outbound)
     3. best-effort seller enrichment:
          jika DetectKeepCode(text) ok DAN catalog_post ditemukan:
             set Raw[catalog_post_id], Raw[kode], Raw[hold_seconds]
        (jika tidak: key seller absen — workflow seller tidak fire, lihat 3.1)
     4. LoadLive → compile → Engine.Run per workflow (first triggered wins)  ← TAK BERUBAH
```

### 3.1 Menjaga seller tetap benar

`seller.commentTrigger.Match` saat ini mengembalikan `true` untuk **semua** komentar (mengandalkan pre-screen handler). Setelah decoupling, komentar biasa juga menjalankan workflow — maka trigger seller yang dibangun via **factory** (`comment-to-order`) harus **fire hanya bila `Event.Raw[kode]` terisi**. Perubahan minimal & lokal di `libs/kits/seller` (Kit boleh berubah; engine tidak):

- Factory `comment-to-order` (di `factories.go`) membangun trigger yang mengecek `e.Raw[RawKeyKode] != ""` (kode terdeteksi + catalog match oleh handler). Komentar tanpa kode → trigger seller tidak fire → tidak ada reservasi.
- Jalur **fallback** legacy (`runner.CommentToOrderWorkflow` + `RegisterNodes`) **tidak diubah** — ia tetap dipakai hanya saat `LoadLive` kosong, dan handler-nya (di jalur itu) tetap pre-screen. Untuk jalur `live`, trigger factory yang meng-guard `Raw[kode]`.

Hasil: satu pipeline melayani (a) workflow seller (fire saat kode+katalog) dan (b) workflow generik (fire pada komentar apa pun) tanpa saling mengganggu.

### 3.2 Catatan volume (risiko R2)

Enqueue semua komentar untuk akun yang punya workflow live menaikkan beban worker & menghitung terhadap cap comment-reply §4c bila `reply-comment` dipakai luas. Mitigasi: dedupe `processed_comment` sudah ada; **safety gate §10 tetap satu-satunya pintu outbound** (cap 750/jam, 30/post/5min, auto-pause). Untuk volume MVP: aman. Rekomendasi diterima (R2).

---

## 4. Redesign UX Builder

### 4.1 Keputusan: adopsi **React Flow (@xyflow/react)** — bukan pertahankan canvas custom

`FlowCanvas.tsx` saat ini murni presentational (SVG bezier + kartu) **tanpa** interaksi: tidak ada drag, tidak ada pembuatan edge, hit-testing manual. Membangun sendiri drag/zoom/pan, connection-by-handle-drag, penghapusan edge, selection multi-node, dan validasi koneksi = pekerjaan besar, rawan bug (matematika drag, z-index, hit-test).

**Rekomendasi: React Flow.** Alasan:

- Drag reposisi, pan/zoom, **edge via drag handle**, hapus edge, selection, minimap, controls — semua bawaan & teruji. `isValidConnection` untuk membatasi koneksi valid (trigger→filter→action).
- MIT, headless-stylable → node custom mudah dipakaikan token desain §11 (dark + lime), reuse styling kartu node yang ada.
- Migrasi murah: canvas lama presentational-only, jadi tidak ada logika interaksi yang hilang. Kita pertahankan **model domain** (`WorkflowNode`/`WorkflowEdge`) dan pasang **adapter** ke tipe `Node`/`Edge` React Flow di tepi — **kontrak backend tak berubah**.

Trade-off jujur: menambah 1 dependency (bundle). Tapi biaya membangun paritas dengan tangan jauh lebih besar dan hasilnya lebih buruk. Net: React Flow.

Batasan yang ditegakkan di FE (server tetap otoritatif):
- `isValidConnection`: sumber & target harus mengikuti urutan kategori (trigger→filter, filter→action, trigger→action bila tak ada filter); tolak koneksi ke kategori yang lebih awal.
- Anti-cycle di klien sebelum publish (server tetap cek `cycle` di activate 422).

### 4.2 Inspector config-schema-driven (DRY §12a)

Ganti `switch(node.node.kind)` di `NodeInspector.tsx` (hanya 2 tipe) dengan **satu `SchemaForm` generik** yang merender field dari **skema per node_type**.

```
packages/types/src/workflow.ts
  NODE_CATALOG[i].configSchema: FieldSchema[]

  type FieldSchema =
    | { kind:'text';      key; label; placeholder?; help? }
    | { kind:'textarea';  key; label; placeholder?; help?; vars?: string[] }
    | { kind:'keywords';  key; label; help? }            // chip / csv → string[]
    | { kind:'boolean';   key; label }
    | { kind:'number';    key; label; min?; max? }
    | { kind:'select';    key; label; options:{value,label}[] }
    | { kind:'multiselect';key; label; options:{value,label}[] }
    | { kind:'phone';     key; label; help? }
    | { kind:'url';       key; label; help? }

apps/web/.../inspector/SchemaForm.tsx    // render field by kind; onChange → state graf
```

Field type dibangun **hanya yang dipakai node phase ini** (rule of three §12a-4): `text`, `textarea`, `keywords`, `boolean`, `number`, `multiselect`, `phone`, `url`. Menambah node baru = tambah entri katalog + `configSchema` → **nol komponen inspector baru**. `defaultConfigFor` diturunkan dari default schema (hapus duplikasi konstanta default di `useWorkflowGraph`).

Skema di TS = untuk **rendering form** (label/hint copy BI). Validasi **kebenaran** tetap independen di `Factory.Build` Go (mis. `phone` wajib di `send-whatsapp-link`, `url` valid di `outbound-webhook`). Ini dua concern berbeda (form-hint vs validasi otoritatif) — bukan duplikasi logika; nama field wajib selaras (disiplin kontrak §5a). Dicatat sebagai R5.

### 4.3 Validasi inline

- Pra-publish di klien: cek `no_trigger`/`no_action`/`trigger_not_runnable`/`cycle` tanpa round-trip; tampilkan banner + **highlight node** bermasalah (ring pink token §11).
- Saat activate 422: map `reason` → node/kondisi terkait, highlight di canvas (mis. `trigger_not_runnable` → beri ring pada node trigger non-runnable; `unknown_node_type` → node tak dikenal). Reuse `VALIDATION_FAILURE_MESSAGES` yang sudah ada.
- Node non-runnable tetap tampil di palette dengan badge "segera" (R6 ADR-004 dipertahankan) — hanya menolak saat publish.

---

## 5. Pembagian Kerja (2 agen paralel)

Path absolut dari root `/Users/fahminurcahya/Documents/Project/zosmed/zosmed_v2`.

### 5.1 go-backend-engineer

- **B1. Decoupling ingest (enabler, prioritas 1).**
  - `apps/api/internal/webhook/handler.go` — `processComment`: enqueue bila `(media ∈ catalog aktif) OR HasLiveWorkflow(accountID)`.
  - `db/query/workflow.sql` — tambah `HasLiveWorkflow` / `CountLiveWorkflowsByAccount` (indexed `workflow_live_idx`). `sqlc generate`.
  - `apps/worker/internal/tasks/comment_ingest.go` — hapus early-return no-keep-code untuk jalur `live`; **selalu** set `Raw[ig_user_id]`, `Raw[comment_at]`; best-effort enrichment seller (kode + catalog) untuk `Raw[catalog_post_id/kode/hold_seconds]`; sisanya (LoadLive→compile→Run) tak berubah. Fallback tetap.
  - `libs/kits/seller/factories.go` (+ trigger di `keep.go`/`kit.go`) — trigger factory `comment-to-order` fire hanya bila `Event.Raw[RawKeyKode] != ""`. Legacy `RegisterNodes` **tidak diubah**.
  - AC: workflow `[comment-received → reply-comment]` fire pada komentar non-katalog tanpa kode; seller tetap reserve hanya saat kode+katalog. Tes worker (fakedb pola existing).
- **B2. Filter netral `post-selection` + `time-window`.**
  - `libs/workflow/nodes/filter_post_selection.go`, `libs/workflow/nodes/filter_time_window.go` (+ `Build*`).
  - `libs/workflow/nodes/catalog.go` — `Runnable=true` untuk keduanya.
  - `nodes.RegisterFactories` — daftarkan factory.
  - Unit test: post match/no-match/empty; time in/out window; empty permissive.
- **B3. Action netral `reply-comment` (GATED).**
  - `libs/workflow/nodes/action_reply_comment.go` — pola one-door meniru `action_wa_link.go`/`seller.privateReplyAction`, `Kind=KindCommentReply`, set `PostID`.
  - catalog `Runnable=true`; `RegisterFactories` daftarkan.
  - Test gate Allow/Queue/Reject (fake Gater + fake Sender).
- **B4. Action netral `outbound-webhook` (non-IG, SSRF guard).**
  - `libs/workflow/nodes/action_outbound_webhook.go` — validasi URL (Build + runtime), tolak loopback/IP privat/link-local, timeout, opsi HMAC.
  - catalog `Runnable=true`; `RegisterFactories` daftarkan.
  - Test: URL valid/invalid, host privat ditolak, non-2xx tidak menggagalkan run.
- **B5. Mirror & validasi.**
  - Pastikan `store.go validateForActivate` otomatis mengizinkan node (A) (ia baca `nodes.Lookup` — otomatis begitu `Runnable` diflip). Konfirmasi `trigger_not_runnable` tetap menolak dm/story.
  - `Factory.Build` tiap node (A) memvalidasi config (mis. `outbound-webhook` url wajib & aman).
- **B6. Tests** menempel tiap paket (B1–B4). Grep-guard: engine core diff kosong; `libs/workflow/nodes/*` tak impor `libs/kits/*`; tak ada `graph.facebook.com`.

### 5.2 frontend-ui-engineer

- **F1. Adopsi React Flow.** Ganti `apps/web/app/(app)/workflows/_components/FlowCanvas.tsx` dengan wrapper `@xyflow/react`; node custom (reuse styling kartu + token §11 dark/lime); edge custom; controlled dari `useWorkflowGraph`. Adapter `WorkflowNode/Edge ↔ RF Node/Edge` di `lib/workflow-catalog.ts` (ganti `nodeToFlowNode`/`edgesToFlowLinks`).
- **F2. Drag + persist posisi.** `onNodeDragStop` → `moveNode(id,x,y)` (baru di `useWorkflowGraph.ts`) → `dirty`. Save sudah mengirim `position` (backend sudah simpan — tak ada kerja backend). Hapus `autoLayoutPosition`-only assumption; auto-layout tetap dipakai untuk posisi awal node baru.
- **F3. Edge create/delete di canvas.** `useWorkflowGraph`: `addEdge(from,to)`, `removeEdge(id)`, `moveNode`. `onConnect` + `isValidConnection` (urutan kategori + anti-cycle). Hapus edge via select+Delete. Buang `autoWireEdges` sebagai satu-satunya cara (boleh dipertahankan sebagai bantuan saat add node dari palette).
- **F4. Inspector schema-driven.** `packages/types/src/workflow.ts` — tambah `FieldSchema` + `configSchema` di tiap `NODE_CATALOG` entry (runnable). Buat `apps/web/app/(app)/workflows/_components/inspector/SchemaForm.tsx` + renderer per `kind`. `NodeInspector.tsx` render `SchemaForm` dari schema node terpilih — **hapus** `switch` hardcoded 2 tipe. `defaultConfigFor` diturunkan dari schema (DRY).
- **F5. Validasi inline.** Pra-publish client checks + highlight node bermasalah; map 422 `reason` → highlight. Reuse `VALIDATION_FAILURE_MESSAGES`.
- **F6. Katalog & tipe config (kontrak — koordinasi awal dgn B).** `packages/types` NODE_CATALOG: `runnable=true` untuk `post-selection`/`time-window`/`reply-comment`/`outbound-webhook` (mirror `catalog.go`); tambah interface config: `PostSelectionConfig`, `TimeWindowConfig`, `ReplyCommentConfig`, `OutboundWebhookConfig` + `configSchema` masing-masing.
- **F7. Guard copy & desain.** Label default BI gaya olshop; token §11 (lime/dark, mono untuk label teknis); pill "● LIVE" = workflow aktif, bukan IG Live (§9).

---

## 6. Urutan & Dependency

```
BACKEND
  B1 ingest decoupling ───────────────────────────────► (reply-comment reachable)
  B2 filters ─┐
  B3 reply-comment (GATED) ─┼─► B5 mirror/validate ─► B6 tests
  B4 outbound-webhook ──────┘
  (B2/B3/B4 paralel; hanya butuh kesepakatan katalog F6)

FRONTEND
  F6 katalog/tipe (KONTRAK — sepakati dulu) ─► F1 React Flow ─┬─► F2 drag
                                                              ├─► F3 edges
                                                              ├─► F4 inspector schema
                                                              └─► F5 validasi inline ; F7 paralel
```

Blocking utama:
- **B1** memblokir makna runtime `reply-comment` (tanpa itu node ada tapi tak pernah fire). Prioritaskan.
- **F6 (katalog+schema)** = kontrak lintas agen; sepakati **sebelum** B2–B4 & F1 agar tak revisi ganda.
- **F1 (React Flow)** fondasi untuk F2/F3/F5.
- Node (A) backend tidak butuh FE untuk lolos tes; FE bisa mulai F1/F4 di atas katalog yang disepakati.

---

## 7. Risiko / Keputusan Terbuka — DIPUTUSKAN (user, 2026-07-07)

> Semua diputuskan sesuai rekomendasi arsitek. Scope phase ini: **4 node (A) + decoupling ingest + UX overhaul React Flow**. 12 node (B) ditunda ke ADR berikutnya.

- **R1 — Adopsi React Flow.** ✅ **DISETUJUI.** Tambah `@xyflow/react` ke `apps/web`.
- **R2 — Volume ingest.** ✅ **DITERIMA untuk MVP.** Enqueue semua komentar untuk akun ber-workflow-live; mitigasi dedupe + gate §10. Tiket optimasi menyusul.
- **R3 — Seller trigger guard.** ✅ **BOLEH.** Trigger factory `comment-to-order` fire hanya saat `Raw[kode]` terisi (perubahan di `libs/kits/seller`, bukan engine).
- **R4 — `intent` ditunda.** ✅ **DISETUJUI.** Pakai `keyword-match` untuk deteksi frasa ragu/trust interim; node `intent` menyusul dengan AI service.
- **R5 — Sumber kebenaran config-schema.** ✅ **DITERIMA.** Render di `packages/types` (FE) + validasi otoritatif di `Factory.Build` (Go); nama field wajib selaras.
- **R6 — SSRF `outbound-webhook`.** ✅ **Guard minimal DITERIMA.** Tolak loopback/IP privat/link-local + timeout, tanpa allowlist domain phase ini.
- **R7 — `send-dm` ditunda.** ✅ **DISETUJUI.** Ditunda bersama Messaging Ingest (di flow komentar gate selalu reject).
- **R8 — `tag-contact`/`handoff-human` ditunda.** ✅ **DISETUJUI.** Butuh contacts/inbox store; menyusul fase berikut.

---

## 8. Guardrail Ringkas (verifikasi PR)

- Engine core `libs/workflow/{engine,node,context,gate,event}.go` **tidak diubah** (grep diff kosong).
- Node (A) netral di `libs/workflow/nodes/*`; **tidak** mengimpor `libs/kits/*` (§8 boundary). Perubahan seller (R3) hanya di `libs/kits/seller/*`.
- Tiap action **outbound IG** (`reply-comment`) memanggil `rc.Gate.Allow` (Kind tepat) sebelum `rc.Sender.*` (§10 one-door). `outbound-webhook` bukan IG → tanpa gate IG tapi ber-guard SSRF/timeout.
- `nodes.Catalog` tidak memuat item §4b (follower/blast/auto-follow/IG-Live). Node (A) semuanya §4a. `comment-received` = post/Reel, bukan IG Live (§4b.5).
- Hanya `graph.instagram.com` (§4.0) — tidak ada referensi `graph.facebook.com`.
- `packages/types` NODE_CATALOG + `configSchema` selaras `libs/workflow/nodes/catalog.go` (kontrak lintas bahasa, §12a-1) — diperbarui dalam commit yang sama.
- Copy default Bahasa Indonesia gaya olshop; token desain §11; pill "● LIVE" = workflow aktif, bukan IG Live (§9).
```
