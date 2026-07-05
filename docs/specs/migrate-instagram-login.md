# ADR-002 — Migrasi ke Instagram API with Instagram Login (`graph.instagram.com`)

Status: Accepted — diimplementasikan 2026-07-05 (AC-1..AC-14 hijau; review C1/M1/m4 diterapkan)
Tanggal: 2026-07-05
Penulis: System Architect (Zosmed)
Scope: Resolusi **MAJOR-0** (docs/specs/comment-to-order.md §10). Menyelaraskan seluruh jalur integrasi Instagram dengan keputusan MENGIKAT CLAUDE.md §4.0 — host `graph.instagram.com`, Instagram Login, IG-user-scoped token — dan membangun OAuth/connect flow yang belum ada.
Referensi: CLAUDE.md §4.0 (API surface mengikat), §4a/§4b/§4c (batasan & rate limit), §5/§5a (arsitektur/monorepo), §6 (domain Account), §12a (prinsip coding). Bergantung pada fondasi ADR-001.

> ✅ **STATUS REKONSILIASI (2026-07-05).** Semua item wire-format bertanda **[GUARDIAN]** SUDAH direkonsiliasi terhadap dok resmi Meta oleh ig-platform-guardian. **Sumber otoritatif = §11-R** (tabel RESOLVED G1–G12). Engineer boleh menulis implementasi berdasarkan §11-R; penanda `[GUARDIAN]` inline yang tersisa sudah terjawab di sana (mis. G1→v25.0, G10→tolerant parser). Verdict: semua surface ALLOW, tidak ada yang menyentuh §4b.

---

## 0. Ringkasan Keputusan

Migrasi dilakukan dalam **tiga lapis yang bisa di-decouple**, dengan boundary `libs/igapi` (transport netral, stateless) tetap terjaga:

1. **Transport (`libs/igapi`)** — ganti base URL `graph.facebook.com/v21.0` → `graph.instagram.com` (versi **[GUARDIAN]**), sesuaikan wording/doc (bukan "Page token" → "IG-user token"), verifikasi path 3 endpoint yang sudah dipakai. Tambah **transport OAuth stateless** (`oauth.go`) untuk exchange/refresh token — tetap tanpa DB, tanpa HTTP server, tanpa Kit.
2. **State & Connect flow (`apps/api/internal/connect` + tabel `account` + scheduler)** — BARU. Endpoint REST connect+callback, penyimpanan long-lived IG-user token per akun di Postgres, dan refresh terjadwal via asynq. Ini yang **belum ada sama sekali**.
3. **Resolusi akun berbasis IGSID** — webhook & worker berhenti memakai env single-account (`IG_ACCESS_TOKEN`/`IG_ACCOUNT_USER_ID`/`IG_ACCOUNT_ID`); token + ig_user_id di-lookup dari tabel `account` lewat `entry.id` (IGSID). Ini sekaligus menutup MINOR **M7** di ADR-001.

Prinsip pembagian tetap §5a/§12a: `libs/igapi` **netral & stateless** (transport IG + OAuth wire, tidak tahu Kit/segmen, tidak menyentuh DB). Semua **state** (token store, sesi OAuth, penjadwalan refresh) hidup di `apps/api`/`apps/worker` + `libs/platform`. Engine (`workflow`/`safety`) tidak tersentuh.

### Acceptance Criteria (Definition of Done — §14) — checklist dapat diverifikasi

- [ ] **AC-1** `libs/igapi` tidak lagi mengandung string `graph.facebook.com` di mana pun; `defaultBaseURL == "https://graph.instagram.com"` (+ versi bila dok mensyaratkan **[GUARDIAN]**). `grep -r "graph.facebook.com" libs/ apps/` = 0 hasil.
- [ ] **AC-2** Header outbound `Authorization: Bearer <IG-user-token>` (tetap). Doc-comment igapi tidak lagi menyebut "Page access token"/"business page"; diganti "IG user access token"/"akun profesional".
- [ ] **AC-3** Endpoint terverifikasi vs dok Instagram Login (**[GUARDIAN]**): public reply `POST /{ig-comment-id}/replies`, private reply `POST /{ig-user-id}/messages` (recipient=`comment_id`), DM `POST /{ig-user-id}/messages` (recipient=`id`), `GET /me`. Perbedaan path/param apa pun dari temuan guardian diterapkan.
- [ ] **AC-4** OAuth transport (`libs/igapi/oauth.go`) menyediakan: `ExchangeCode` (short-lived), `ExchangeLongLived` (short→long), `RefreshLongLived`. Stateless, host per §3.2 **[GUARDIAN]**, unit-test dengan httptest server (pola `NewWithBaseURL`).
- [ ] **AC-5** Endpoint REST `GET /connect/instagram` (redirect ke authorize) dan `GET /connect/instagram/callback` (exchange + persist) berjalan; state param anti-CSRF diverifikasi; callback menyimpan long-lived token + expiry + scopes + ig_user_id ke `account`.
- [ ] **AC-6** Migrasi DB menambah kolom token di `account` (`access_token`, `token_type`, `scopes`, `token_expires_at`, `token_refreshed_at`) + tidak ada kolom "page token". `account.ig_user_id` = IGSID (sudah ada, UNIQUE).
- [ ] **AC-7** Query `GetAccountByIgUserID`, `UpsertAccountFromOAuth`, `UpdateAccountToken`, `ListAccountsDueForRefresh` ada di `db/query/account.sql` + tergenerate `dbgen`.
- [ ] **AC-8** Refresh terjadwal: task periodic `token:refresh-sweep` (asynq Scheduler) memperpanjang token yang mendekati kedaluwarsa; token gagal-refresh menandai `account.status='expired'` (bukan crash). Refresh mematuhi syarat umur token minimum **[GUARDIAN]**.
- [ ] **AC-9** Webhook resolusi akun: handler me-lookup akun via `entry.id` (IGSID) → `GetAccountByIgUserID`; tidak lagi membaca `IG_ACCOUNT_ID` dari env. Comment untuk akun tak dikenal → di-skip aman (bukan 500).
- [ ] **AC-10** Worker membangun `igapi.Client` dari token per-akun hasil lookup DB (bukan `IG_ACCESS_TOKEN` env); `igUserID` sender dari `account.ig_user_id` (bukan `IG_ACCOUNT_USER_ID` env). Token **tidak** pernah dimasukkan ke payload task/Redis.
- [ ] **AC-11** Config: `IG_APP_ID`, `IG_APP_SECRET`, `IG_REDIRECT_URI` divalidasi saat startup; `META_APP_SECRET`/`META_VERIFY_TOKEN` di-rename → `IG_APP_SECRET`/`IG_VERIFY_TOKEN` (satu App Secret untuk OAuth client_secret **dan** HMAC webhook — DRY). Env single-account lama dihapus.
- [ ] **AC-12** Webhook langganan field via produk **Instagram** (bukan Messenger/Facebook) terdokumentasi di deploy/README; HMAC `X-Hub-Signature-256` pakai App Secret (**[GUARDIAN]**: konfirmasi sumber sama).
- [ ] **AC-13** Boundary utuh: `libs/igapi` tidak mengimpor `apps/*`, tidak menyentuh `pgx`/DB, tidak tahu Kit. Engine (`workflow`/`safety`) tak berubah. Build+test seluruh workspace hijau.
- [ ] **AC-14** §4b bersih: tidak ada follower trigger, blast, scraping, auto-follow, atau referensi IG Live yang lahir dari migrasi ini.

---

## 1. Scope & Non-Goals

**In scope:** base URL + doc igapi, OAuth transport, connect REST flow, token store + migrasi, refresh scheduler, resolusi akun berbasis IGSID, config env, boundary check.

**Non-goals (jangan dikerjakan di ADR ini):**
- Enkripsi token at-rest end-to-end (envelope encryption) — dicatat sebagai **hardening follow-up** (§4.4), tidak memblok fungsional MVP.
- Multi-akun UI/settings di frontend — cukup endpoint + satu akun uji dulu.
- Perbaikan MAJOR-2/MAJOR-3 ADR-001 (outbound retry, tx reserve) — terpisah; ADR ini hanya **meminjam** scheduler asynq yang sama (§5) sehingga MAJOR-3(b) reconcile bisa menumpang infrastruktur yang sama nanti.
- App Review Meta submission — proses non-kode; dicatat sebagai prasyarat produksi.

---

## 2. Perubahan `libs/igapi` (file-per-file)

`libs/igapi` tetap thin, stateless, netral. Tidak ada perubahan struktur `Client` (masih `httpClient/accessToken/baseURL`). Yang berubah: konstanta base URL, doc-comment, dan penambahan satu file OAuth transport.

### 2.1 `client.go`
- **`defaultBaseURL`**: `"https://graph.facebook.com/v21.0"` → **`"https://graph.instagram.com/v25.0"`** (RESOLVED G1 — versi live v25.0, ber-segmen versi). Simpan sebagai konstanta tunggal (DRY) — semua path lain relatif terhadapnya.
- Doc-comment `Client`/`New`: "IG page access token" → "IG user access token (IG-user-scoped long-lived, §4.0)". Hapus istilah "business page".
- `NewWithBaseURL` **dipertahankan** (dipakai unit test httptest — lihat client_test.go). Tidak ada perubahan mekanik.
- `post(...)`: tidak berubah (Authorization Bearer + JSON). Header sudah benar.

### 2.2 `comments.go`
- `ReplyToComment` → `POST /{comment-id}/replies` body `{message}`. Path & body **[GUARDIAN]** (Instagram Login: sebagian dok memakai field `message`, konfirmasi). Kemungkinan besar tidak berubah selain host. Update doc-comment rate-limit (tetap merujuk §4c).

### 2.3 `messages.go`
- `SendPrivateReply` → `POST /{ig-user-id}/messages` recipient `{comment_id}`. `SendDM` → recipient `{id}`. **[GUARDIAN]**: pada `graph.instagram.com`, konfirmasi (a) path `/{ig-user-id}/messages` vs `/me/messages`, (b) apakah `recipient.comment_id` tetap mekanisme private reply, (c) apakah butuh `messaging_product`/param tambahan. Update doc "business page" → "akun profesional (sender = account.ig_user_id)".

### 2.4 `types.go`
- Rename doc paket: "Meta Instagram Graph API" → "Instagram API with Instagram Login (graph.instagram.com)".
- **`GraphErrorResponse`/`GraphError`**: bentuk envelope error di `graph.instagram.com` **[GUARDIAN]** — kemungkinan tetap `{"error":{message,type,code,fbtrace_id}}`, tapi `fbtrace_id` bisa berbeda nama. Pertahankan struct; sesuaikan tag hanya jika guardian menemukan beda. (Rename tipe menjadi `IGError` bersifat kosmetik — **opsional**, jangan jadi beban migrasi.)
- `replyRequest`/`messagesRequest`/`messagesResponse`: sesuaikan hanya bila guardian menemukan perbedaan field.

### 2.5 `oauth.go` (BARU — transport OAuth stateless)
Rumah untuk 3 panggilan token. **Bukan** bagian dari `Client` (host berbeda dari `graph.instagram.com`; lihat §3.2). Bebas-state, tanpa DB, tanpa cookie/sesi (itu urusan `apps/api/internal/connect`).

Signature yang diusulkan (netral, argumen eksplisit — hindari over-abstraction §12a-4):

```
type OAuthConfig struct { AppID, AppSecret, RedirectURI string }

// URL authorize untuk redirect (tidak memanggil jaringan).
func (o OAuthConfig) AuthorizeURL(state string, scopes []string) string

// code → short-lived token (host api.instagram.com [GUARDIAN]).
func (o OAuthConfig) ExchangeCode(ctx, code string) (ShortLivedToken, error)

// short-lived → long-lived (~60h, host graph.instagram.com/access_token [GUARDIAN]).
func (o OAuthConfig) ExchangeLongLived(ctx, shortToken string) (LongLivedToken, error)

// perpanjang long-lived (graph.instagram.com/refresh_access_token [GUARDIAN]).
func (o OAuthConfig) RefreshLongLived(ctx, longToken string) (LongLivedToken, error)

type LongLivedToken struct { AccessToken, TokenType string; ExpiresIn int64 }
```

Host per-fungsi disimpan sebagai konstanta di `oauth.go` (satu tempat). `ExpiresIn` (detik) dikonversi ke `token_expires_at = now + ExpiresIn` oleh **pemanggil** (connect handler / scheduler), bukan di dalam igapi — igapi tak menyimpan waktu. `ig_user_id` (IGSID) di-resolve dengan `GET /me?fields=user_id,username` **[GUARDIAN]** memakai `Client` biasa setelah token didapat (bisa method `Client.Me(ctx)` baru di `client.go`/`me.go`).

> **Boundary:** `oauth.go` hanya melakukan HTTP + parse JSON. Tidak ada `pgx`, tidak ada `net/http.Handler`. Ini menjaga igapi tetap importable oleh siapa pun tanpa menyeret dependency app/DB (§5a).

---

## 3. OAuth / Connect Flow (BARU)

### 3.1 Sequence (Business Login for Instagram)

```
 User (browser)        apps/api /connect            libs/igapi/oauth        Instagram            Postgres
      │                       │                            │                    │                   │
      │  GET /connect/instagram                            │                    │                   │
      ├──────────────────────►│  buat state (nonce),       │                    │                   │
      │                       │  simpan state (signed      │                    │                   │
      │                       │  cookie / short TTL redis) │                    │                   │
      │                       │  AuthorizeURL(state,scopes)│                    │                   │
      │  302 → instagram.com/oauth/authorize               │                    │                   │
      │◄──────────────────────┤                            │                    │                   │
      │                                                                          │                   │
      │  user login IG + grant scopes  ────────────────────────────────────────►│                   │
      │  302 → REDIRECT_URI?code=...&state=...                                    │                   │
      │◄─────────────────────────────────────────────────────────────────────── ┤                   │
      │  GET /connect/instagram/callback?code&state        │                    │                   │
      ├──────────────────────►│ verifikasi state (CSRF)    │                    │                   │
      │                       │ ExchangeCode(code) ───────►│ POST api.instagram │                   │
      │                       │                            │  .com/oauth/...    │                   │
      │                       │◄── short-lived token ──────┤                    │                   │
      │                       │ ExchangeLongLived ────────►│ graph.instagram    │                   │
      │                       │◄── long-lived (~60d) ──────┤  .com/access_token │                   │
      │                       │ Client.Me() ──────────────►│ GET /me (user_id)  │                   │
      │                       │◄── ig_user_id, username ───┤                    │                   │
      │                       │ UpsertAccountFromOAuth ─────────────────────────────────────────────►│
      │  302 → /settings?connected=1                        │                    │                   │
      │◄──────────────────────┤                            │                    │                   │
```

### 3.2 Endpoint & host OAuth  **[GUARDIAN — semua host/param di blok ini]**

| Langkah | Host / URL (asumsi) | Param kunci |
|---|---|---|
| authorize | `https://www.instagram.com/oauth/authorize` | `client_id`, `redirect_uri`, `response_type=code`, `scope`, `state` |
| exchange code | `POST https://api.instagram.com/oauth/access_token` | `client_id`, `client_secret`, `grant_type=authorization_code`, `redirect_uri`, `code` |
| short→long | `GET https://graph.instagram.com/access_token` | `grant_type=ig_exchange_token`, `client_secret`, `access_token` |
| refresh | `GET https://graph.instagram.com/refresh_access_token` | `grant_type=ig_refresh_token`, `access_token` |
| identitas | `GET https://graph.instagram.com/me` | `fields=user_id,username` |

Scopes **[GUARDIAN]** (pakai nama `instagram_business_*`, bukan scope Facebook lama): `instagram_business_basic`, `instagram_business_manage_comments`, `instagram_business_manage_messages`, (+`instagram_business_content_publish`, `instagram_business_manage_insights` untuk fase lanjutan). Default scope set didefinisikan **satu kali** sebagai konstanta di `libs/igapi/oauth.go` dan diimpor connect handler (DRY §12a-1).

### 3.3 Scaffold `apps/api/internal/connect/` (BARU)

```
apps/api/internal/connect/
├── handler.go     # Start (GET /connect/instagram) + Callback (GET .../callback)
├── state.go       # buat/verifikasi state anti-CSRF (HMAC(state, IG_APP_SECRET) atau redis TTL)
└── store.go       # ConnectStore: adapter dbgen (UpsertAccountFromOAuth, UpdateAccountToken)
```

- `handler.go` menerima `igapi.OAuthConfig`, `ConnectStore`, dan `*slog.Logger` (constructor `connect.New(...)`). Tidak menulis SQL langsung (SoC §12a-3) — lewat `store.go` → `dbgen`.
- **State anti-CSRF:** pilihan sederhana MVP = signed opaque state (HMAC dengan App Secret + timestamp, TTL pendek). Tidak butuh tabel baru. **[GUARDIAN]** tidak relevan (murni internal).
- Router (`httpx/router.go`): tambah `Routes.ConnectStart` & `Routes.ConnectCallback` sebagai `http.HandlerFunc`, di-mount **tanpa** `AuthStub` untuk callback (Instagram memanggil balik), tapi `Start` boleh di belakang auth user (sesi login Zosmed). MVP: keduanya publik dengan proteksi state. Catat sebagai keputusan wiring.

### 3.4 Perubahan `httpx/router.go` & `apps/api/cmd/api/main.go`
- `Routes`: +`ConnectStart`, +`ConnectCallback`.
- `main.go`: bangun `igapi.OAuthConfig{cfg.IGAppID, cfg.IGAppSecret, cfg.IGRedirectURI}`, `connect.New(oauthCfg, connectStore, log)`, daftarkan 2 route. Hapus blok `IG_ACCOUNT_ID` env (§6).

---

## 4. Token Store & Migrasi DB

### 4.1 Keadaan sekarang
`db/migrations/00001_accounts.sql` `account(id, ig_user_id UNIQUE, handle, display_name, status, created_at)` — **belum ada** kolom token/refresh/expiry. `ig_user_id` sudah cocok sebagai IGSID (UNIQUE) — dipertahankan.

### 4.2 Migrasi baru `db/migrations/00006_account_tokens.sql`
`ALTER TABLE account ADD` (nullable dulu, karena akun bisa dibuat sebelum konek — tapi di flow ini akun lahir dari callback, jadi bisa NOT NULL DEFAULT '' lalu diisi):

```
access_token        text        NOT NULL DEFAULT ''   -- IG-user-scoped long-lived token
token_type          text        NOT NULL DEFAULT 'bearer'
scopes              text[]      NOT NULL DEFAULT '{}'  -- instagram_business_*
token_expires_at    timestamptz                        -- now + expires_in (nullable sblm konek)
token_refreshed_at  timestamptz
```

- **Tidak ada** kolom `page_token`/`page_id` — §4.0 melarang Page token.
- Index untuk sweep refresh: `CREATE INDEX account_refresh_due_idx ON account (token_expires_at) WHERE status = 'connected';`
- `status` CHECK tetap `('connected','expired','disconnected')` — cocok untuk siklus refresh gagal → `expired`.

### 4.3 Query `db/query/account.sql` (BARU) → `dbgen`
- `GetAccountByIgUserID(ig_user_id) :one` — untuk webhook resolusi (IGSID → account row + token).
- `GetAccountByID(id) :one` — untuk worker (AccountID UUID → token + ig_user_id).
- `UpsertAccountFromOAuth(ig_user_id, handle, display_name, access_token, token_type, scopes, token_expires_at) :one` — `ON CONFLICT (ig_user_id) DO UPDATE` (re-connect memperbarui token).
- `UpdateAccountToken(id, access_token, token_expires_at, token_refreshed_at) :exec` — dipakai scheduler.
- `MarkAccountExpired(id) :exec` — refresh gagal.
- `ListAccountsDueForRefresh(threshold timestamptz) :many` — `WHERE status='connected' AND token_expires_at < threshold`.

> Token disimpan sebagai satu-satunya sumber kredensial per akun. Worker/connect **tidak** menyimpan token di tempat lain (DRY). Env `IG_ACCESS_TOKEN` dihapus.

### 4.4 Hardening follow-up (di luar scope, dicatat)
Enkripsi token at-rest (envelope encryption dengan `ENCRYPTION_KEY`) via helper `libs/platform/secret`. MVP menyimpan token apa adanya di kolom `access_token` dengan catatan TODO + pembatasan akses DB. Tidak memblok fungsional. Ditandai untuk security review sebelum produksi.

---

## 5. Refresh Terjadwal (asynq Scheduler)

Long-lived token ~60 hari; harus di-refresh sebelum kedaluwarsa dan (biasanya) setelah berumur minimal tertentu **[GUARDIAN]**.

- **Task periodic baru** `token:refresh-sweep` didefinisikan di `libs/platform/tasks/types.go` (satu sumber nama task, dipakai scheduler + handler — DRY, pola sama seperti ADR-001).
- **Scheduler** di `apps/worker/cmd/worker/main.go`: tambah `asynq.Scheduler` (atau `PeriodicTaskManager`) yang meng-enqueue `token:refresh-sweep` mis. tiap 6–12 jam. Ini **infrastruktur scheduler pertama** di worker — MAJOR-3(b) (reservation reconcile) nanti menumpang scheduler yang sama (jangan bikin dua).
- **Handler** `apps/worker/internal/tasks/token_refresh.go`:
  1. `ListAccountsDueForRefresh(now + REFRESH_LEAD)` (mis. lead 7 hari).
  2. Per akun: `oauth.RefreshLongLived(token)` → sukses `UpdateAccountToken`; gagal → `MarkAccountExpired` + log (bukan error fatal seluruh sweep).
  3. Idempotent & aman diulang (refresh berkali-kali tetap valid).
- Handler ini butuh `igapi.OAuthConfig` (dari config) + `dbgen`. Ditambahkan ke `runner` atau di-wire langsung di worker main.

```
 asynq Scheduler ──(tiap 6-12h)──► enqueue token:refresh-sweep
                                            │
                                            ▼
                        token_refresh handler ── ListAccountsDueForRefresh
                                            │
                     ┌──────────────────────┼───────────────────────┐
                     ▼ sukses               ▼ gagal                  ▼ dst
             UpdateAccountToken      MarkAccountExpired(status)   (per-akun, non-fatal)
```

---

## 6. Webhook & Resolusi Akun berbasis IGSID

### 6.1 Perubahan resolusi akun (menutup M7 ADR-001)
Saat ini `apps/api/cmd/api/main.go` mengambil `accountID` dari env `IG_ACCOUNT_ID` dan mengoper UUID statis ke `webhook.New`. Ganti:

- Webhook handler (`apps/api/internal/webhook/handler.go`) menerima `dbgen.Querier` (sudah punya `queries`) dan me-resolve akun **per entry**: `entry.id` (IGSID string) → `GetAccountByIgUserID` → `account.id` (UUID) untuk payload task. Entry milik akun tak dikenal → skip aman + log debug (bukan 500).
- `CommentIngestPayload` (libs/platform/tasks) **sudah** membawa `AccountID string` (UUID). Isi dari hasil lookup, bukan env. Token **tidak** dimasukkan (keamanan) — worker lookup token sendiri.
- Hapus argumen `accountID` statis dari `webhook.New(...)`.

### 6.2 Worker memakai token per-akun (menutup ketergantungan env)
`apps/worker/internal/tasks/comment_ingest.go`:
- Ganti `igapi.New(h.r.IGToken)` → lookup `GetAccountByID(p.AccountID)` → `igapi.New(account.AccessToken)`.
- Ganti `os.Getenv("IG_ACCOUNT_USER_ID")` → `account.IgUserID`.
- `runner.Runner.IGToken` (field statis) **dihapus**; `runner.New(...)` tak lagi menerima `igToken`. Client dibangun per-run dari token akun (sesuai desain "satu Client per akun" di doc `client.go`).
- Akun `status != 'connected'` (mis. `expired`) → skip + log (jangan kirim dengan token mati).

### 6.3 Shape payload webhook (RESOLVED G10/G11)
`payload.go` `CommentValue{id, text, from{id,username}, media{id}, parent_id}` — sebagian besar tetap, DENGAN satu perubahan wajib:
- **⚠️ AMBIGUITAS DOK META (WAJIB tolerant parser):** field id komentar bisa `id` **atau** `comment_id` — halaman *webhook reference* Meta memakai `id`, halaman *examples* memakai `comment_id`, dan tidak bisa dipastikan tanpa payload live. **Tambah field `CommentID string json:"comment_id"` di `CommentValue`** dan di `ExtractComments` fallback: `if cv.ID == "" { cv.ID = cv.CommentID }`. **Jangan pilih satu buta.**
- `object == "instagram"` (bukan `"page"`) — parser saat ini tidak memvalidasi `object`, itu justru robust; biarkan.
- `from.id` = IGSID commenter, `media.id` ada. `from.username` bisa kosong (bergantung permission) — pastikan `{nama}` handoff punya fallback bila `username` kosong (handler sudah aman menyimpan `ContactHandle` kosong).
- `entry[].time` = waktu **notifikasi**, BUKAN `created_time` komentar. Untuk **M4** (window 7 hari): propagasikan `entry.time` sebagai proxy + dokumentasikan keterbatasan; akurasi penuh butuh `GET /{comment-id}?fields=timestamp` (opsional, bukan blocker ADR ini).
- HMAC `X-Hub-Signature-256` memakai **App Secret** yang sama (RESOLVED G11 — bukan Page-scoped secret); `verify.go` tak berubah.

### 6.4 Langganan webhook via produk Instagram
Non-kode tapi wajib: subskripsi field `comments`/`messages` dilakukan di App Dashboard **produk Instagram** (bukan Messenger/Facebook). Dokumentasikan di `deploy/README` + checklist onboarding App Review. **[GUARDIAN]** konfirmasi daftar field yang tersedia di produk Instagram.

---

## 7. Config (`libs/platform/config/config.go`)

Perubahan env (validasi startup). Konsolidasi App Secret: satu nilai untuk OAuth `client_secret` **dan** HMAC webhook (§4.0 — satu app) → **DRY**, hindari dua env yang harus sinkron.

| Env lama | Env baru | Catatan |
|---|---|---|
| `META_APP_SECRET` | `IG_APP_SECRET` | Dipakai: (a) OAuth `client_secret`, (b) HMAC `X-Hub-Signature-256`, (c) sign state CSRF. |
| `META_VERIFY_TOKEN` | `IG_VERIFY_TOKEN` | Verifikasi challenge webhook (tetap fungsinya). |
| — | `IG_APP_ID` (baru) | OAuth `client_id`. |
| — | `IG_REDIRECT_URI` (baru) | Harus cocok persis dengan yang didaftarkan di App Dashboard. |
| `IG_ACCESS_TOKEN` (worker env) | **dihapus** | Token dari DB per akun (§6.2). |
| `IG_ACCOUNT_USER_ID` (worker env) | **dihapus** | Dari `account.ig_user_id`. |
| `IG_ACCOUNT_ID` (api env) | **dihapus** | Resolusi via IGSID (§6.1). |
| `DB_URL`,`REDIS_URL`,`WA_PHONE`,`PORT` | (tetap) | — |

`config.Config` + `validate()` diperbarui: tambah `IGAppID`, `IGAppSecret`, `IGRedirectURI` (required); rename field `MetaAppSecret`→`IGAppSecret`, `MetaVerifyToken`→`IGVerifyToken`. Semua pembaca (`webhook.New`, `connect.New`, scheduler) memakai field baru. Update `config_test.go`.

> Karena rename menyentuh banyak call-site (webhook), lakukan dalam **satu commit** (§5a — jaga selaras). Update `deploy/` `.env.example`/compose.

---

## 8. Boundary & SoC (§5a/§12a)

| Kapabilitas | Lokasi | Alasan |
|---|---|---|
| HTTP call ke graph.instagram.com (reply/DM/me) | `libs/igapi` (netral, stateless) | Transport IG; tak tahu Kit/DB |
| OAuth exchange/refresh (HTTP wire) | `libs/igapi/oauth.go` (netral, stateless) | Wire OAuth; tetap tanpa state/DB |
| Sesi/state CSRF, redirect, handler HTTP connect | `apps/api/internal/connect` | State & transport HTTP milik app |
| Persist token, upsert akun, query | `db/query/account.sql` → `dbgen`; adapter `connect/store.go` | Data access; handler tak tulis SQL |
| Penjadwalan refresh | `apps/worker` scheduler + `tasks/token_refresh.go` | Orkestrasi/infra worker |
| Nama task/periodic | `libs/platform/tasks` | Satu sumber, dipakai api+worker (DRY) |
| Resolusi akun via IGSID | `apps/api/internal/webhook` + `apps/worker` | Transport; pakai `dbgen` |

Yang **tidak** berubah: `libs/workflow`, `libs/safety`, `libs/kits/seller` (logika Kit netral terhadap host IG). Migrasi ini murni menyentuh transport + state akun. Uji boundary: `libs/igapi` tetap tanpa import `pgx`/`apps/*`/`kits/*`.

Anti over-abstraction (§12a-4): OAuth diekspos sebagai 3-4 fungsi eksplisit pada `OAuthConfig`, **bukan** interface `TokenProvider` generik (baru satu implementasi). Interface muncul nanti bila ada channel kedua (mis. TikTok fase 3), bukan sekarang.

---

## 9. Urutan Implementasi + Dependensi

Migrasi disusun agar **lapis transport bisa mendarat lebih dulu** (memperbaiki bug §4.0 langsung) sebelum connect flow yang lebih besar.

**Fase A — Transport igapi (cepat, unblock guardrail §4.0):**
1. `libs/igapi/client.go` base URL + doc; `comments.go`/`messages.go`/`types.go` doc & (bila perlu) wire dari temuan **[GUARDIAN]**. Update `client_test.go` (host, error envelope). — go-backend-engineer
2. `libs/igapi/oauth.go` + `Client.Me()` (`me.go`) + unit test httptest. Bergantung: keputusan host **[GUARDIAN §3.2]**. — go-backend-engineer

**Fase B — Config & DB (paralel dengan A):**
3. `config.go` rename+tambah env + `config_test.go`; update `deploy/.env.example`. — go-backend-engineer
4. `db/migrations/00006_account_tokens.sql` + `db/query/account.sql` + `sqlc generate`. — go-backend-engineer

**Fase C — Connect flow (setelah A2 + B):**
5. `apps/api/internal/connect` (handler/state/store); tambah routes di `httpx/router.go`; wire di `apps/api/cmd/api/main.go` (+ hapus `IG_ACCOUNT_ID`). Bergantung: 2,3,4. — go-backend-engineer

**Fase D — Resolusi akun IGSID (setelah B):**
6. Webhook resolve `entry.id` → `GetAccountByIgUserID`; hapus `accountID` statis dari `webhook.New`. Bergantung: 4. — go-backend-engineer
7. Worker `comment_ingest.go` lookup token/ig_user_id per akun via `GetAccountByID`; hapus `Runner.IGToken` & env single-account; skip akun non-connected. Bergantung: 4. — go-backend-engineer

**Fase E — Refresh scheduler (setelah B + A2):**
8. `libs/platform/tasks` tambah `TaskTokenRefreshSweep`; `apps/worker` register `asynq.Scheduler`; `tasks/token_refresh.go`. Bergantung: 2,4. — go-backend-engineer

**Fase F — Dokumentasi & FE (paralel):**
9. `deploy/README` langganan webhook produk Instagram + checklist App Review + var env baru. — go-backend-engineer
10. FE: layar Settings/Onboarding tombol "Hubungkan Instagram" → `GET /connect/instagram`; tampilkan status akun (`connected/expired`). Copy Bahasa Indonesia. — frontend-ui-engineer

Jalur kritis: **1 (unblock §4.0)** dapat merdeka lebih dulu. Untuk connect utuh: 2 → 4 → 5. 6/7 bergantung hanya pada 4. Semua item wire-format tertahan sampai rekonsiliasi **[GUARDIAN]**.

---

## 10. Alternatif yang Ditolak

- **Taruh OAuth+token store di dalam `libs/igapi`.** Ditolak: menyeret `pgx`/DB & sesi HTTP ke paket transport netral, melanggar §5a (igapi harus importable tanpa dependency app/DB) dan SoC §12a-3. OAuth **wire** boleh di igapi (stateless); **state** tidak.
- **Simpan token di payload task asynq (Redis).** Ditolak: kredensial jangka-panjang bocor ke broker; sulit di-rotate. Worker lookup token dari DB per-task (§6.2).
- **Pertahankan env single-account (`IG_ACCESS_TOKEN`).** Ditolak: bertentangan dengan model per-akun §6/§4.0, menghalangi multi-akun, dan meninggalkan M7. Token per akun di DB.
- **Refresh token via goroutine timer in-memory.** Ditolak: tak tahan restart/terdistribusi (sama alasan ADR-001 menolak timer in-memory). Pakai asynq Scheduler durable.
- **Dua env terpisah `META_APP_SECRET` + `IG_APP_SECRET`.** Ditolak: satu app = satu App Secret dipakai OAuth & HMAC; dua env yang harus sinkron = sumber bug (DRY §12a-1). Konsolidasi ke `IG_APP_SECRET`.
- **Bikin interface `TokenProvider`/`OAuthProvider` generik sekarang.** Ditolak (§12a-4): baru satu implementasi (IG). Abstraksi menunggu pemakai kedua (TikTok fase 3).
- **Tunda perbaikan base URL sampai connect flow selesai.** Ditolak: base URL `graph.facebook.com` adalah **pelanggaran guardrail §4.0 aktif**; Fase A mendaratkannya lebih dulu, independen dari connect flow.

---

## 11. Daftar Konsolidasi — PERLU KONFIRMASI GUARDIAN

Semua item ini **harus** direkonsiliasi dengan ig-platform-guardian sebelum kode wire-format ditulis. Nomor untuk rujuk-silang.

- **G1** Base URL `graph.instagram.com` — versionless atau butuh segmen versi? (§2.1)
- **G2** Path & body public reply `POST /{ig-comment-id}/replies` (field `message`?). (§2.2)
- **G3** Private reply: `POST /{ig-user-id}/messages` vs `/me/messages`; `recipient.comment_id` tetap mekanismenya? Param tambahan (`messaging_product`)? (§2.3)
- **G4** DM: `recipient.id` + window/param di Instagram Login. (§2.3)
- **G5** Bentuk error envelope `graph.instagram.com` (`error.message/type/code/fbtrace_id`?). (§2.4)
- **G6** OAuth hosts & param persis: authorize (`www.instagram.com/oauth/authorize`), exchange (`api.instagram.com/oauth/access_token`), long-lived & refresh (`graph.instagram.com/access_token` / `refresh_access_token`), `grant_type` values. (§3.2)
- **G7** Nama scope `instagram_business_*` yang valid + mana yang butuh Advanced Access/App Review. (§3.2)
- **G8** `GET /me?fields=user_id,username` — field mana yang mengembalikan IGSID untuk disimpan sebagai `account.ig_user_id`. (§2.5/§3.1)
- **G9** Syarat refresh: umur token minimum sebelum boleh refresh & masa berlaku (~60 hari?). (§5)
- **G10** Shape payload webhook `comments` di Instagram Login: `object=="instagram"`, `from.id/username`, `media.id`, ketersediaan `username`; `entry.time`. (§6.3)
- **G11** HMAC `X-Hub-Signature-256` memakai App Secret yang sama (bukan Page secret); langganan field via produk Instagram + daftar field tersedia (`comments`,`messages`,`mentions`). (§6.3/§6.4)
- **G12** Konfirmasi tidak ada langkah wajib yang menyentuh §4b (mis. tidak butuh Page/Facebook token tersembunyi). Bila dok menuntut sesuatu yang melanggar §4.0/§4b → STOP & eskalasi.

---

## 11-R. REKONSILIASI GUARDIAN (RESOLVED — verifikasi terhadap dok resmi Meta)

> ig-platform-guardian sudah memverifikasi seluruh item §11 terhadap dok live Meta (audit 2026-07-05). Ini **sumber otoritatif** untuk wire-format — engineer boleh menulis kode berdasarkan tabel ini. Verdict keseluruhan: **semua surface ALLOW; tidak ada yang menyentuh §4b.**

| # | Resolusi terkonfirmasi | Sumber dok |
|---|---|---|
| **G1** | Host `graph.instagram.com`, versi **v25.0** (ber-segmen versi) → `graph.instagram.com/v25.0`. Header `Authorization: Bearer <IG-user-token>` sudah benar (client.go:57). | instagram-platform/…/messaging-api |
| **G2** | Public reply `POST /{comment-id}/replies` body `{message}` — **path sama, host beda**. Tidak ada perubahan kode selain base URL. | ig-comment/replies |
| **G3** | Private reply `POST /{ig-id}/messages` recipient `{comment_id}` — **identik** di Instagram Login. `messages.go` sudah benar. Window 7 hari, 1 pesan/commenter. Meter ke cap DM (MAJOR-1 sudah). | private-replies |
| **G4** | DM `POST /{ig-id}/messages` body `{"recipient":{"id":"<IGSID>"},"message":{"text":"..."}}`, response `{recipient_id,message_id}`. Window 24 jam. Shape `types.go` sudah benar. | messaging-api |
| **G5** | Error envelope tetap `{"error":{message,type,code,fbtrace_id}}` (mis. OAuthException code 190). **Tidak ada perubahan struct.** | messaging-api |
| **G6** | OAuth hosts/param **persis seperti asumsi §3.2** — authorize `www.instagram.com/oauth/authorize`; exchange `POST api.instagram.com/oauth/access_token`; long-lived `GET graph.instagram.com/access_token` (`grant_type=ig_exchange_token`); refresh `GET graph.instagram.com/refresh_access_token` (`grant_type=ig_refresh_token`). Code valid 1 jam, single-use. | business-login |
| **G7** | Scope slice: `instagram_business_basic`, `instagram_business_manage_comments`, `instagram_business_manage_messages`. Scope lama `instagram_basic`/`pages_*` deprecated 27-Jan-2025. **App Review wajib** untuk `manage_messages`+`manage_comments` di produksi; dev pakai standard access + test users. | instagram-api-with-instagram-login |
| **G8** | `GET /me?fields=user_id,username,account_type` → `user_id` = IGSID untuk disimpan sebagai `account.ig_user_id`. | messaging-api |
| **G9** | Long-lived token **60 hari**. Refresh syarat: token **≥24 jam** umur, masih valid, scope `instagram_business_basic` granted. Scheduler harus hormati umur min 24 jam. | business-login |
| **G10** | ⚠️ **Field id komentar `id` vs `comment_id` ambigu di dok Meta** → parser toleran dua-duanya (lihat §6.3). `object=="instagram"`. `from.username` bisa kosong. `entry.time` = waktu notifikasi (proxy untuk M4). | webhooks/examples |
| **G11** | HMAC `X-Hub-Signature-256` (`sha256=<hex>`) pakai **App Secret sama** — `verify.go` tak berubah. Langganan field `comments`/`messages` via produk **Instagram** (`POST /me/subscribed_apps?subscribed_fields=comments`). | webhooks |
| **G12** | **Tidak ada** langkah wajib yang menyentuh §4b. Semua surface = kemampuan resmi §4a. Migrasi boleh lanjut. | — |

**Butuh konfirmasi payload live / App Review (bukan blocker kode, catat sebagai risiko):**
1. Field `id` vs `comment_id` webhook `comments` — validasi dengan payload sandbox sebelum go-live (parser toleran sudah menutup risiko).
2. Advanced Access `manage_messages`/`manage_comments` — App Review sebelum produksi; dev pakai test users.
3. Angka pasti rate-limit private reply/jam — verifikasi via header `X-Business-Use-Case-Usage` saat App Review; default §4c dipakai sementara.

---

## 12. Catatan untuk Engineer Berikutnya

- **Jangan** menulis wire-format apa pun (host/param/scope/payload) sampai item **[GUARDIAN] §11** direkonsiliasi oleh parent. Fase A boleh mulai pada bagian non-wire (rename doc, struktur `oauth.go`, test scaffold) sambil menunggu.
- Rename env (`META_*`→`IG_*`) menyentuh `config.go`, `config_test.go`, `webhook.New`, `deploy/`. Satu commit (§5a).
- Setelah migrasi: `grep -rn "graph.facebook.com" .` dan `grep -rn "IG_ACCESS_TOKEN\|IG_ACCOUNT_USER_ID\|IG_ACCOUNT_ID" .` harus 0 (kecuali dokumen historis). Ini bagian AC-1/AC-11.
- Token = rahasia. Jangan log token; jangan taruh di payload task; batasi field select query yang mengembalikan `access_token`.
- Copy default UI connect Bahasa Indonesia gaya ramah (§12/§11). Status akun pakai token/pill §11.
- MAJOR-3(b) (reservation reconcile) dan item ini sama-sama butuh asynq Scheduler — bangun scheduler **sekali** di worker main (§5), jangan duplikasi.
- Sinkron dengan ADR-001: MAJOR-0 di §10 comment-to-order.md dianggap **selesai** ketika AC-1..AC-14 hijau; perbarui status di dokumen itu.
