# ADR-001 — Comment-to-Order (keep/C engine) — Vertical Slice

Status: Proposed
Tanggal: 2026-06-29
Penulis: System Architect (Zosmed)
Scope: SATU vertical slice end-to-end dari Seller Kit §8.1.4. Bukan seluruh MVP.
Referensi: CLAUDE.md §4 (batasan IG), §5/§5a (arsitektur/monorepo), §6 (domain), §7 (node), §8.1 (Seller Kit), §10 (safety), §12a (prinsip coding).

---

## 0. Ringkasan Keputusan

Membangun fondasi backend Go (yang saat ini belum ada) sekaligus dengan satu alur fungsional penuh: pelanggan komen kode `keep`/`C1` di **post/Reel** → webhook `comments` Meta → deteksi kode → reserve stok + countdown → auto private-reply (Graph API) → handoff closing via WhatsApp deep link (`wa.me`) → auto-release jika tidak ditebus.

Slice ini sengaja dipilih karena memaksa kita membangun semua tulang punggung yang dipakai ulang seluruh MVP: webhook ingest + verifikasi signature, queue (asynq), workflow engine netral, safety layer, igapi client, dan satu Kit (seller) di atas engine. Dengan menyelesaikan satu kolom vertikal, semua paket horizontal lahir dengan kontrak yang sudah terbukti.

### Acceptance Criteria (Definition of Done — §14)

1. Webhook `GET /webhooks/meta` (verify challenge) dan `POST /webhooks/meta` (event) berjalan; signature `X-Hub-Signature-256` diverifikasi; payload non-valid ditolak 403.
2. Komentar berisi kode keep/C pada post yang terdaftar membuat satu `Reservation` berstatus `reserved` dengan `expires_at = now + hold` (default 5 menit) dan men-decrement stok produk.
3. Tepat **satu** private reply per komentar dikirim via Graph API, melewati safety layer (§10), dalam window 7 hari. Tidak ada DM langsung yang mem-bypass safety layer.
4. Private reply berisi `wa.me` prefilled dengan `{nama}`, `{kode}`, `{produk}` (handoff closing — nol API pembayaran).
5. State machine reservasi berjalan: `reserved → waiting-pay → closed-wa` (sukses) atau `→ expired-released` (timeout) dengan auto-release stok.
6. Dedupe: komentar duplikat dari user yang sama untuk kode yang sama tidak membuat reservasi/DM kedua.
7. REST API menyediakan data untuk layar Comment-to-Order selaras `CommentOrderData` (apps/web/lib/mock/workflows.ts).
8. Tidak menyentuh DO-NOT list §4b: berbasis komentar post/Reel, BUKAN IG Live. Tidak ada follower trigger, blast, scraping, atau auto-follow.
9. Guardrail §4c ditegakkan: window 24h, 1 private reply/komentar (≤7 hari), rate-limit cap, dedupe, auto-pause.
10. Engine (`libs/workflow`, `libs/safety`, `libs/igapi`) tetap netral segmen; semua istilah keep/C/produk ada di `libs/kits/seller`.

---

## 1. Scaffold Plan (struktur §5a — file/paket yang harus dibuat)

Backend belum ada sama sekali. Slice ini membuat kerangka berikut. Path absolut dari root repo `/Users/fahminurcahya/Documents/Project/zosmed/zosmed_v2`.

### 1.1 Go workspace

```
go.work                          # go 1.23; use ./apps/api ./apps/worker ./libs/*
```

`go.work` isi (skeleton):

```
go 1.23

use (
    ./apps/api
    ./apps/worker
    ./libs/igapi
    ./libs/safety
    ./libs/workflow
    ./libs/kits/seller
    ./libs/platform        // shared infra: config, db pool, logger, queue client
)
```

> Catatan: `libs/platform` ditambahkan sebagai rumah hal infrastruktur lintas-app (pgx pool, asynq client/server bootstrap, config env, logger) supaya `apps/api` dan `apps/worker` tidak menduplikasi setup (DRY §12a-1). Ini tetap netral segmen.

### 1.2 apps/api — REST + webhook server

```
apps/api/
├── go.mod
├── cmd/api/main.go                    # bootstrap chi router, db pool, asynq client, graceful shutdown
└── internal/
    ├── httpx/
    │   ├── router.go                  # chi mount: /webhooks/meta, /api/v1/...
    │   ├── middleware.go              # request id, recover, logging, auth (stub MVP)
    │   └── respond.go                 # JSON envelope helper {data,error}
    ├── webhook/
    │   ├── handler.go                 # GET verify challenge, POST receive
    │   ├── verify.go                  # X-Hub-Signature-256 HMAC-SHA256 check
    │   └── payload.go                 # struct Meta webhook (comments field) + parse
    ├── commentorder/                  # transport untuk layar Comment-to-Order
    │   ├── handler.go                 # GET reservations/comments/products/stats; POST mark-closed
    │   └── dto.go                     # request/response shape (selaras packages/types)
    └── enqueue/
        └── enqueue.go                 # bungkus asynq.Client → task payload (SoC: handler tak tahu redis)
```

`apps/api` mengimpor: `libs/platform`, `libs/safety` (untuk pembacaan gauge), dan `db` (sqlc-generated). Tidak mengimpor `libs/kits/seller` secara langsung untuk path REST membaca data (data sudah ternormalisasi di DB); deteksi kode terjadi di worker.

### 1.3 apps/worker — asynq worker

```
apps/worker/
├── go.mod
├── cmd/worker/main.go                 # asynq.Server + mux register handlers + scheduler (asynq Scheduler/PeriodicTask)
└── internal/
    ├── tasks/
    │   ├── types.go                   # konstanta nama task + payload struct (satu sumber, dipakai api & worker)
    │   ├── comment_ingest.go          # handler task "comment:ingest" → jalankan workflow engine
    │   └── reservation_expire.go      # handler task "reservation:expire" → transisi expired-released
    └── runner/
        └── runner.go                  # wiring: workflow.Engine + safety.Gate + igapi.Client + seller Kit
```

> `tasks/types.go` adalah satu-satunya definisi nama task & payload, diimpor `apps/api/internal/enqueue` dan worker handler (DRY). Karena kedua app modul terpisah, paket ini sebaiknya tinggal di `libs/platform/tasks` agar dapat diimpor dua arah tanpa app→app. Keputusan: **letakkan di `libs/platform/tasks`**.

Revisi struktur task payload:

```
libs/platform/tasks/types.go           # TaskCommentIngest, TaskReservationExpire + payload structs
```

### 1.4 libs/ — shared Go

```
libs/
├── platform/
│   ├── go.mod
│   ├── config/config.go               # env: DB_URL, REDIS_URL, META_APP_SECRET, META_VERIFY_TOKEN, WA_PHONE
│   ├── db/pool.go                     # pgxpool bootstrap
│   ├── queue/queue.go                 # asynq client+server factory
│   ├── tasks/types.go                 # nama task + payload (lihat 1.3)
│   └── log/log.go                     # slog wrapper
│
├── igapi/                             # NETRAL — client Instagram Graph API
│   ├── go.mod
│   ├── client.go                      # type Client struct{httpClient, token}
│   ├── comments.go                    # ReplyToComment(ctx, commentID, text) ; HideComment(...)
│   ├── messages.go                    # SendPrivateReply(ctx, commentID, msg) ; SendDM(ctx, igUserID, msg)
│   └── types.go                       # request/response Graph API
│
├── safety/                            # NETRAL — §10 (skeleton; detail ke safety engineer)
│   ├── go.mod
│   ├── gate.go                        # type Gate interface: Allow(ctx, OutboundReq) (Decision, error)
│   ├── quota.go                       # counter per akun (redis); cap dari konstanta §4c
│   ├── window.go                      # cek 24h window & 7-day private-reply window
│   ├── dedupe.go                      # key (account, user, trigger) → sudah pernah?
│   └── decision.go                    # Decision: Allow | Queue | Reject + reason
│
├── workflow/                          # NETRAL — engine §7 (trigger→filter→action)
│   ├── go.mod
│   ├── engine.go                      # type Engine; Run(ctx, Event) (RunResult, error)
│   ├── node.go                        # interface Trigger/Filter/Action + Registry
│   ├── event.go                       # type Event (sumber-agnostik: comment/dm/story)
│   └── context.go                     # RunContext: variabel {{nama}} dll, services (igapi, safety)
│
└── kits/
    └── seller/                        # SELLER KIT §8.1 — di atas engine
        ├── go.mod
        ├── keep.go                    # DetectKeepCode(text) → (code, ok) ; KIT keywords
        ├── reservation.go             # ReservationService: Reserve/Close/Expire + state machine
        ├── privatereply.go            # template builder private reply + wa.me link
        └── kit.go                     # RegisterNodes(reg *workflow.Registry) — daftar node seller ke engine
```

Boundary impor (ditegakkan, §5a): `libs/kits/seller` boleh impor `workflow`, `safety`, `igapi`, `platform`. Engine (`workflow`/`safety`/`igapi`) TIDAK boleh impor `kits/*`. Engine menerima node seller via registry (dependency inversion), bukan import langsung.

### 1.5 db/ — migrations + sqlc

```
db/
├── sqlc.yaml                          # engine=postgresql, sqlc-gen-go, package "dbgen", out libs/platform/dbgen
├── migrations/                        # goose
│   ├── 00001_accounts.sql            # account (sudah ada konsep di FE) — minimal untuk slice
│   ├── 00002_catalog_post.sql        # catalog_post (post/Reel terdaftar) + products
│   ├── 00003_reservations.sql        # reservations + state
│   ├── 00004_processed_comments.sql  # dedupe ledger (comment_id unik)
│   └── 00005_outbound_log.sql        # jejak outbound (private reply) untuk audit/safety
└── query/
    ├── reservations.sql              # -- name: CreateReservation :one  dst.
    ├── products.sql                  # decrement/increment stok
    ├── comments.sql                  # insert processed comment (ON CONFLICT DO NOTHING → dedupe)
    └── commentorder_read.sql         # query agregat untuk layar (stats, queue, catalog)
```

Skema inti (ringkas):

```
catalog_post(id, account_id, ig_media_id, caption, comments_count, active, created_at)
product(id, catalog_post_id, code, name, price_idr, stock_total, stock_left)
reservation(id, account_id, catalog_post_id, product_id, code, ig_comment_id,
            contact_ig_user_id, contact_handle, status, hold_seconds,
            reserved_at, expires_at, closed_at, wa_link)
processed_comment(ig_comment_id PRIMARY KEY, account_id, received_at)   -- dedupe
outbound_log(id, account_id, kind, target_user_id, trigger_key, ig_object_id, sent_at)
```

`status` enum: `reserved | waiting-pay | closed-wa | expired-released` (canonical, selaras domain.ts).

---

## 2. Reservation State Machine

State canonical (selaras `ReservationStatus` di packages/types/src/domain.ts):

```
                ┌──────────────────────────────────────────────────────┐
                │                                                      │
  komen kode    ▼                                                      │
  terdeteksi  ┌──────────┐  private reply  ┌─────────────┐  buyer chat │ countdown
  + stok ok → │ reserved │ ───terkirim───► │ waiting-pay │ ──di WA──►  │ habis
              └──────────┘                 └─────────────┘  (manual    │
                   │                              │         mark)      ▼
        countdown  │                              │            ┌──────────────┐
        habis      │                   countdown  │            │  closed-wa   │ (terminal sukses)
                   ▼                   habis       ▼            └──────────────┘
            ┌──────────────────┐ ◄─────────────────┘
            │ expired-released │  (terminal gagal — stok dikembalikan)
            └──────────────────┘
```

| Dari | Ke | Trigger | Efek samping |
|------|----|---------|--------------|
| (start) | `reserved` | Worker mendeteksi kode keep/C valid pada post terdaftar, stok `> 0`, lolos dedupe | `stock_left -= 1`; set `expires_at = now + hold_seconds` (default 300s); enqueue `reservation:expire` ber-delay |
| `reserved` | `waiting-pay` | Private reply (berisi wa.me) berhasil terkirim & melewati safety gate | Catat `outbound_log`; mulai menunggu konfirmasi closing |
| `reserved` | `expired-released` | Task `reservation:expire` jalan saat `now >= expires_at` dan status masih `reserved` | `stock_left += 1`; reservasi ditutup gagal |
| `waiting-pay` | `closed-wa` | Admin menandai closed di layar (POST mark-closed) atau sinyal closing manual | `closed_at = now`; stok TIDAK dikembalikan (sudah laku) |
| `waiting-pay` | `expired-released` | Task `reservation:expire` jalan, status masih `waiting-pay`, tidak ada konfirmasi | `stock_left += 1` |

Aturan timeout/countdown:
- `hold_seconds` = konfigurasi Kit seller (default 300). Disimpan per-reservation agar audit konsisten meski default berubah.
- Saat membuat reservasi, worker meng-enqueue task `reservation:expire` dengan `asynq.ProcessIn(hold_seconds)` dan `asynq.TaskID(reservationID)` (idempotent — satu timer per reservasi).
- Transisi terminal (`closed-wa`, `expired-released`) bersifat final; task expire yang menyusul harus no-op bila status sudah terminal (cek status di handler — guard race).
- Auto-release menaikkan kembali `stock_left` hanya dari state non-terminal (`reserved`/`waiting-pay`). Tidak pernah dari `closed-wa`.

Catatan keselarasan FE: mock `ReservationRow.st` memakai literal `'expired'`, sedangkan domain canonical `'expired-released'`. **Keputusan:** API mengembalikan canonical `expired-released`; saat wiring FE nanti, layar memetakan `expired-released → 'expired'` untuk label pendek (atau perbaiki mock). Dicatat sebagai task wiring, bukan perubahan kontrak backend.

---

## 3. Webhook Ingest (comment-to-order)

### 3.1 Verifikasi & subscribe
- `GET /webhooks/meta?hub.mode=subscribe&hub.verify_token=...&hub.challenge=...` → cek `verify_token == META_VERIFY_TOKEN`, balas `hub.challenge` plaintext 200. Kalau salah → 403.
- App di Meta berlangganan field `comments` (dan `messages` untuk fase berikut). Slice ini hanya menangani `comments`.

### 3.2 Terima event
- `POST /webhooks/meta`:
  1. Baca raw body (sebelum unmarshal) → hitung `HMAC-SHA256(body, META_APP_SECRET)` → bandingkan konstan-time dengan header `X-Hub-Signature-256` (`sha256=...`). Gagal → 403, tidak diproses.
  2. Unmarshal payload `comments`. Bentuk relevan:
     ```
     entry[].changes[].field == "comments"
     entry[].changes[].value: { id (comment_id), text, from{id,username}, media{id}, parent_id? }
     ```
  3. Dedupe cepat di ingest: `INSERT INTO processed_comment(ig_comment_id...) ON CONFLICT DO NOTHING`. Bila 0 rows → komentar sudah pernah diproses, **abaikan** (idempotent terhadap retry Meta).
  4. Filter awal murah: hanya enqueue jika `media.id` ada di `catalog_post` aktif milik akun. (Hemat queue; deteksi kode tetap di worker.)
  5. Enqueue task `comment:ingest` (payload = account_id, comment_id, media_id, from, text). **Balas 200 secepatnya** — pemrosesan berat ada di worker (webhook Meta harus cepat, retry kalau non-200).

> Penting: webhook handler TIDAK memanggil Graph API atau menulis reservasi langsung. Ia hanya verifikasi → dedupe → enqueue. Ini menjaga SoC (§12a-3) dan throughput (§5).

### 3.3 Pemrosesan di worker (`comment:ingest`)
1. Muat konteks akun + catalog post + produk.
2. Bangun `workflow.Event{Source: comment, ...}` dan jalankan `Engine.Run`. Engine memanggil node seller (terdaftar via registry):
   - Filter `post-selection` (netral) memastikan media terdaftar.
   - Node seller `DetectKeepCode(text)` → (code, ok). Tidak match → selesai (catat run skipped).
   - Node seller `Reserve` → cek stok, buat reservation `reserved`, decrement, enqueue expire timer.
   - Node seller `PrivateReply` → minta safety gate, lalu `igapi.SendPrivateReply`. Sukses → transisi `waiting-pay`.
3. Semua langkah tercatat sebagai RunLog (untuk layar Runs §9).

---

## 4. Kontrak API REST (→ packages/types)

Envelope umum: `{ "data": <T>, "error": null }` atau `{ "data": null, "error": {code,message} }`.

### 4.1 Webhook Meta (publik, dipanggil Meta)
- `GET  /webhooks/meta` — verify challenge (plaintext).
- `POST /webhooks/meta` — terima event; body = payload Meta; respons `200 {data:{received:true}}`.

### 4.2 Layar Comment-to-Order (dipanggil FE)

Selaras `CommentOrderData` (apps/web/lib/mock/workflows.ts). Endpoint tunggal agregat agar FE tetap satu fetch (`getCommentOrder`):

`GET /api/v1/comment-order?accountId=&postId=` → `CommentOrderResponse`:

```ts
// usulan untuk packages/types/src/comment-order.ts (baru)
export interface IncomingCommentDTO {
  id: Id;
  user: string;            // handle, tanpa '@'  (FE field: u)
  text: string;            // FE: t
  ago: string;             // relative time, server-rendered (FE: tm)
  matchedCode: string | null;  // FE: match
  reserved: boolean;       // FE: ok
  duplicate: boolean;      // FE: dupe
}

export interface ReservationDTO {
  id: Id;
  code: string;
  buyerHandle: string;     // FE: u
  product: string;         // FE: p
  priceLabel: string;      // "Rp 189rb" (FE: price)
  status: ReservationStatus;        // canonical: reserved|waiting-pay|closed-wa|expired-released
  countdownLabel: string;  // "4:52" | "✓ closed" | "— released" (FE: cd)
  expiresAt: ISODateTime;  // sumber untuk countdown live di FE
}

export interface CatalogProductDTO {
  code: string;
  name: string;
  stockLeft: number;       // FE: left
  stockTotal: number;      // FE: total
}

export interface CommentOrderStatDTO {
  key: 'code-detected' | 'reserved-now' | 'closed-wa' | 'expired';
  label: string;
  value: string;
}

export interface CommentOrderResponse {
  postCommentsLabel: string;          // FE: postComments
  comments: IncomingCommentDTO[];
  stats: CommentOrderStatDTO[];
  reservations: ReservationDTO[];
  products: CatalogProductDTO[];
}
```

> Field warna (`color`) di mock dihasilkan FE dari `status`/rasio stok — BUKAN dari backend (presentational, §12a-3 SoC). Backend mengirim data mentah; FE memetakan ke token §11.

### 4.3 Aksi admin
- `POST /api/v1/reservations/{id}/close` → tandai `waiting-pay → closed-wa`. Body kosong. Respons `ReservationDTO`.
- `GET  /api/v1/reservations/{id}` → detail satu reservasi (opsional, untuk drill-down).

### 4.4 Keyword settings (header layar "Keyword settings")
- `GET  /api/v1/comment-order/settings?accountId=` → `{ keywords: string[], holdSeconds: number, replyTemplate: string }`.
- `PUT  /api/v1/comment-order/settings` → update; default keywords dari `KIT_KEYWORDS.seller` (packages/types/src/constants.ts) — satu sumber, jangan hardcode ulang.

---

## 5. Jalur Safety Layer (§10) — high level

Semua outbound IG dalam slice ini (private reply) WAJIB lewat `safety.Gate.Allow(ctx, OutboundReq)` sebelum menyentuh `igapi`. Tidak ada pemanggilan `igapi.Send*` langsung dari Kit/engine.

`OutboundReq` (skeleton): `{ AccountID, Kind: "private-reply"|"dm", TargetUserID, TriggerKey, CommentID, CommentAt }`.

Gate memeriksa (detail diserahkan ke safety engineer):
- Window: private reply harus ≤7 hari sejak `CommentAt` (PRIVATE_REPLY_WINDOW_DAYS); DM lanjutan harus dalam 24h (MESSAGING_WINDOW_HOURS). Lewat window → `Reject` (private reply) atau arahkan opt-in (DM).
- Rate limit: counter per akun terhadap cap §4c (dmPerHour 200 → overflow `Queue`, commentRepliesPerHour 750, commentsPerPostPer5min 30). Default dari `RATE_LIMITS` (constants.ts) — diimpor Go-side sebagai konstanta paralel di `libs/safety` (satu nilai, dua bahasa; jaga selaras lewat commit yang sama §5a).
- Dedupe: key `(AccountID, TargetUserID, TriggerKey)` — cegah private reply/DM ganda untuk pemicu yang sama. Selaras dengan dedupe ingest (`processed_comment`) tapi pada lapis berbeda (komentar vs outbound).
- Auto-pause: ≥80% (AUTO_PAUSE_THRESHOLD) → Gate balas `Queue`/`Reject` + emit SafetyEvent; kill switch global → semua `Reject`.

Decision mapping di runner:
- `Allow` → kirim via igapi → transisi `reserved → waiting-pay`.
- `Queue` → reservasi tetap `reserved`, outbound diantre asynq (overflow), timer expire tetap jalan. (Jika expire mendahului pengiriman, reservasi `expired-released` — perilaku benar: stok tak boleh ditahan tanpa kepastian closing.)
- `Reject` → reservasi tetap `reserved` sampai expire; catat RunLog + SafetyEvent alasannya.

---

## 6. Boundary Engine-Netral vs Seller Kit (§8)

| Kapabilitas | Lokasi | Alasan |
|-------------|--------|--------|
| Verifikasi webhook, parse payload `comments` | `apps/api/internal/webhook` | Transport netral; semua segmen pakai webhook yang sama |
| Queue/task plumbing | `libs/platform` | Infra netral |
| `Event` (comment/dm/story-agnostik), Engine Run, Registry node | `libs/workflow` | Engine netral — tidak tahu kata "keep" atau "produk" |
| Filter `post-selection`, `keyword-match` generik, `time-window` | `libs/workflow` | Primitive netral; Kit mengonfigurasi parameternya |
| Safety gate, window, dedupe, quota | `libs/safety` | Netral; berlaku semua outbound semua Kit |
| Graph API client (reply/private-reply/DM) | `libs/igapi` | Netral; transport IG |
| **Deteksi kode keep/C1/C3** | `libs/kits/seller/keep.go` | Spesifik seller (commerce comment) |
| **ReservationService + state machine** | `libs/kits/seller/reservation.go` | Konsep "reserve stok" milik segmen jualan |
| **Template private reply + wa.me builder** | `libs/kits/seller/privatereply.go` | Copy & handoff WA = aset Kit seller |
| **Stok/produk/catalog** | `libs/kits/seller` + tabel `product` | Konsep dagang, bukan netral |
| Pendaftaran node seller ke engine | `libs/kits/seller/kit.go` → `RegisterNodes(reg)` | Kit menambah preset ke engine via registry; engine tidak impor Kit |

Uji boundary: menambah Booking Kit nanti = membuat `libs/kits/booking` (comment → wa.me/kalender) tanpa menyentuh `libs/workflow`/`libs/safety`/`libs/igapi`. Reservation/stok TIDAK boleh bocor ke engine.

Catatan §4b: seluruh slice berbasis webhook `comments` post/Reel. Tidak ada referensi IG Live, watching count, follower trigger, blast, atau scraping. Node palette FE yang berlabel "Reserve stok (live)" adalah penamaan lama; **harus dibaca sebagai comment-to-order post/Reel** dan idealnya di-rename "Reserve stok (comment-to-order)" saat wiring (catatan untuk frontend-ui-engineer; konsisten dengan §9).

---

## 7. Urutan Implementasi + Dependensi

Fase A — Fondasi (blokir semua):
1. `go.work` + `libs/platform` (config, db pool, queue factory, log, tasks/types). — go-backend-engineer
2. `db/` migrations (00001–00005) + `sqlc.yaml` + query + generate `dbgen`. — go-backend-engineer
3. `libs/igapi` skeleton (Client + ReplyToComment/SendPrivateReply signature; impl HTTP). Verifikasi kemampuan endpoint dengan **ig-platform-guardian** sebelum implementasi. — go-backend-engineer

Fase B — Engine & Safety (paralel setelah A):
4. `libs/workflow` (Engine, Node interface, Registry, Event, RunContext). — go-backend-engineer
5. `libs/safety` (Gate interface + quota/window/dedupe/decision). Detail algoritma → safety engineer. Bergantung: platform (redis). — go-backend-engineer

Fase C — Seller Kit (setelah B):
6. `libs/kits/seller` (keep detect, ReservationService+state machine, private reply+wa.me, RegisterNodes). Bergantung: workflow, safety, igapi, dbgen. — go-backend-engineer

Fase D — Transport (setelah C):
7. `apps/api` webhook handler (verify + parse + dedupe + enqueue). Bergantung: platform, dbgen. — go-backend-engineer
8. `apps/worker` task handlers (`comment:ingest` → runner → engine+kit; `reservation:expire`). Bergantung: workflow, kits/seller, safety, igapi, platform. — go-backend-engineer
9. `apps/api` REST comment-order (read agregat + close + settings). Bergantung: dbgen. — go-backend-engineer

Fase E — Kontrak FE & wiring:
10. `packages/types/src/comment-order.ts` + export di index. — frontend-ui-engineer
11. Ganti `getCommentOrder` di apps/web/lib/mock/api.ts dari mock → fetch `/api/v1/comment-order`; map `expired-released → 'expired'`, derive warna dari status/rasio; countdown live dari `expiresAt`. Rename label node "Reserve stok (live)" → "(comment-to-order)". — frontend-ui-engineer

Jalur kritis: 1 → 2 → 4/5 → 6 → 8. Webhook (7) bisa paralel setelah 2. FE (10/11) bisa mulai begitu kontrak §4.2 disepakati (tidak menunggu impl backend selesai).

---

## 8. Alternatif yang Ditolak

- **Tulis reservasi langsung di webhook handler (tanpa queue).** Ditolak: webhook Meta harus balas cepat; pemrosesan berat + panggilan Graph API memblokir dan memicu retry. Queue (asynq) wajib untuk pacing & overflow (§5/§10).
- **Letakkan deteksi keep/C + reservasi di `libs/workflow`.** Ditolak: melanggar boundary §8 — engine harus netral segmen. Masuk `libs/kits/seller`.
- **Timer countdown via goroutine in-memory.** Ditolak: tidak tahan restart, tidak terdistribusi. Pakai asynq `ProcessIn` + `TaskID` (durable, idempotent).
- **Endpoint REST granular per-list (comments, reservations, products terpisah).** Ditolak untuk MVP: FE memakai satu `getCommentOrder`; satu endpoint agregat = wiring minimal. Granular bisa menyusul bila layar berkembang.
- **Pembayaran/QRIS di slice ini.** Ditolak — fase lanjutan butuh KYC (§5/§12). Closing = handoff `wa.me` saja.
- **Dedupe hanya di safety layer.** Ditolak: dedupe ingest (`processed_comment` unik per `comment_id`) menangkis retry Meta lebih awal & murah; dedupe safety menangani sisi outbound (user,trigger). Dua lapis, tujuan beda.

---

## 9. Catatan untuk Engineer Berikutnya

- Sebelum implementasi `libs/igapi`, konfirmasi ke **ig-platform-guardian**: bentuk payload webhook `comments`, endpoint `POST /{comment-id}/replies` vs private reply `POST /{ig-id}/messages` (recipient `{comment_id}`), dan batas 1 private reply/komentar ≤7 hari. Jangan implementasi di atas asumsi.
- Selaraskan konstanta rate-limit: `RATE_LIMITS` (packages/types/src/constants.ts) dan padanannya di `libs/safety`. Ubah keduanya dalam satu commit (§5a).
- Semua copy default Bahasa Indonesia gaya olshop (§12). Template private reply default sudah dicontohkan di layar FE.

---

## 10. Follow-up / Known Issues (hasil code review — belum dikerjakan)

Slice happy-path lengkap, build+test hijau, dan §4b compliance bersih (tidak menyentuh DO-NOT list). Item berikut adalah **hardening saat beban/kegagalan** yang sengaja DITUNDA (keputusan: kerjakan sebagai follow-up). MAJOR-1 (DM cap tidak ditegakkan) SUDAH diperbaiki.

### MAJOR-1 — Private reply metered ke DM cap (SELESAI ✅)
Private reply dikirim via `POST /{ig-id}/messages` (sebuah DM), jadi harus dihitung ke cap DM (§4c: 200/jam overflow→Queue, 1000/hari), bukan cap comment-reply (750/jam). Diperbaiki di `libs/safety/quota.go` + `gate.go`: `KindPrivateReply` kini meter cap DM; ditambah `KindCommentReply` untuk public reply-comment (§7) yang memakai cap 750/30. Tes `compliance_test.go`/`gate_test.go` diselaraskan.

### MAJOR-0 — Re-review & migrasi ke Instagram Login (`graph.instagram.com`) (SELESAI ✅)
**Diselesaikan 2026-07-05 via ADR-002 (`docs/specs/migrate-instagram-login.md`), AC-1..AC-14 hijau.** Base URL → `graph.instagram.com/v25.0`, OAuth/connect flow + token store (migrasi `00007_account_tokens.sql`) dibangun, resolusi akun berbasis IGSID, refresh scheduler, env `META_*`→`IG_*`. Diverifikasi ig-platform-guardian (§11-R) + code-reviewer (fix C1 kebocoran token log, M1 error DB webhook). Detail keputusan asli di bawah (arsip):

Keputusan platform (CLAUDE.md §4.0): integrasi IG **wajib** via **Instagram API with Instagram Login** di host **`graph.instagram.com`**, **bukan** `graph.facebook.com`/Facebook Login. Slice ini dibangun sebelum keputusan itu ditegaskan, jadi **review ulang & sesuaikan** seluruh jalur IG:
- **`libs/igapi`**: base URL saat ini `https://graph.facebook.com/v21.0` (`client.go:13`) → ganti ke `https://graph.instagram.com` (versi sesuai dok); verifikasi path endpoint (`POST /{ig-id}/messages` private reply, `POST /{ig-comment-id}/replies`, `GET /me`) sesuai dok Instagram Login; header `Authorization: Bearer <IG-user-token>`.
- **OAuth/connect flow** (belum dibangun di slice ini): authorization `instagram.com/oauth/authorize` (scope `instagram_business_*`), exchange `api.instagram.com/oauth/access_token` → long-lived di `graph.instagram.com/access_token`, refresh `graph.instagram.com/refresh_access_token`. Simpan **IG-user-scoped long-lived token** (bukan Page token) di tabel `account`.
- **Webhook**: pastikan langganan field `comments`/`messages` lewat produk **Instagram** (App Dashboard), bukan Messenger/Facebook. HMAC App Secret tetap.
- **Identitas akun**: `account.ig_user_id` = IG-user id (IGSID); resolusi akun (`GetAccountByIgUserID`, lihat M-account) berbasis id ini, bukan Page id. Konfirmasi shape payload webhook `comments` di Instagram Login (field `from.id`, `media.id`) bisa berbeda dari versi Facebook Login.
- **Scopes & rate-limit/window**: cek ulang window 24h DM, 1 private-reply/komentar (≤7 hari), dan cap §4c terhadap dok Instagram Login (bisa ada perbedaan). Audit ulang lewat **ig-platform-guardian** sebelum implementasi connect flow.

### MAJOR-2 — Gate `Queue` menjatuhkan outbound (belum diretry)
`libs/kits/seller/privatereply.go`: pada `DecisionQueue`, action mengembalikan sukses "deferred" tapi tidak ada yang mengirim ulang. §4c/§10 menuntut overflow → antre → kirim saat kuota pulih. **Rencana:** task asynq khusus `outbound:send` (terpisah dari reserve) yang di-enqueue ber-delay saat Queue; reserve sudah terjadi sehingga retry hanya melakukan langkah private-reply (idempotent, tidak double-reserve). Tanpa ini, di bawah beban reply tak terkirim dan reservasi `expired-released`.

### MAJOR-3 — Durabilitas reserve & auto-release
`libs/kits/seller/reservation.go`:
- (a) `DecrementStock` + `CreateReservation` jalan sebagai dua call terpisah tanpa transaksi → bila CreateReservation gagal setelah decrement, stok bocor permanen. **Rencana:** bungkus dalam satu pgx tx (`pool.Begin` → `queries.WithTx(tx)` → decrement → create → commit). Butuh threading pool + WithTx lewat `reservationDB` interface + wiring runner + test double.
- (b) Bila `enqueue(reservation:expire)` gagal (reservation.go:141-146), reservasi persist `reserved` tanpa timer → unit ditahan selamanya. Timer asynq satu-satunya mekanisme expiry; tidak ada DB sweep backstop meski index parsial `reservation_active_expires_at_idx` (00003) dibuat untuk itu, dan worker `main.go` belum register scheduler. **Rencana:** periodic reconcile (asynq Scheduler) yang melepas `reserved`/`waiting-pay` lewat `expires_at` (menutup kegagalan enqueue & kehilangan task Redis).

### MINOR yang layak ditindaklanjuti
- **M4 — 7-day window tak pernah menolak:** `apps/worker/internal/tasks/comment_ingest.go` set `RawKeyCommentAt: time.Now()` alih-alih waktu komentar asli; `payload.go` belum menangkap `entry[].time`/timestamp komentar → cek window 7 hari (§4c) selalu lolos. Tangkap timestamp webhook dan propagasikan.
- **M5 — UUID helper bocor lintas boundary:** `ParseUUID`/`UUIDToString` di `libs/kits/seller/reservation.go` diimpor transport netral `apps/api/internal/commentorder/handler.go` dan diduplikasi di `webhook/handler.go`. Pindahkan ke `libs/platform` (SoC + DRY).
- **M6 — Cast enum rapuh di adapter gate:** `apps/worker/internal/runner/runner.go` melakukan `workflow.DecisionAction(d.Action)` mengandalkan kesejajaran iota safety↔workflow. Ganti dengan `switch` eksplisit.
- **M7 — Env dibaca tersebar:** `comment_ingest.go` baca `IG_ACCOUNT_USER_ID` per-task; resolusi konteks akun sebaiknya sekali saat wiring (dan per-akun dari tabel `account` nanti).
- **N9 — Stat `code-detected` = COUNT reservasi:** komentar yang match kode tapi out-of-stock (tak ada reservasi) tidak terhitung "kode terdeteksi" (`db/query/commentorder_read.sql`). Semantik label perlu disesuaikan.
