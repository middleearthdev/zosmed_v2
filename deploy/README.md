# Zosmed — Deploy Notes (Instagram Login)

Ini bukan panduan infra lengkap — hanya bagian yang berubah oleh migrasi ke
**Instagram API with Instagram Login** (`graph.instagram.com`, CLAUDE.md §4.0,
ADR-002). Prasyarat non-kode sebelum go-live diringkas di sini.

## 1. Environment variables

Lihat [`.env.example`](./.env.example) untuk daftar lengkap. Variabel baru
sejak ADR-002:

| Var | Wajib | Keterangan |
|---|---|---|
| `IG_APP_ID` | ya | `client_id` OAuth, dari Meta App Dashboard produk **Instagram** |
| `IG_APP_SECRET` | ya | satu App Secret untuk OAuth `client_secret` **dan** HMAC webhook **dan** signing state CSRF (DRY — satu App) |
| `IG_VERIFY_TOKEN` | ya | token verifikasi handshake webhook (`GET /webhooks/meta`) |
| `IG_REDIRECT_URI` | ya | harus identik dengan "Valid OAuth Redirect URI" yang didaftarkan |

Variabel baru sejak ADR-003 (login + onboarding), keduanya **opsional**:

| Var | Default | Keterangan |
|---|---|---|
| `APP_ENV` | `dev` | `prod` mengaktifkan cookie sesi `Secure=true` **dan** membuat `apps/api/cmd/seed` menolak jalan (seed memakai kredensial dummy) |
| `WEB_BASE_URL` | — | base URL frontend, untuk redirect absolut bila suatu saat dibutuhkan; sebagian besar redirect di codebase ini relatif |

Variabel lama berikut **dihapus** — jangan set lagi, aplikasi tidak
membacanya:

- `META_APP_SECRET`, `META_VERIFY_TOKEN` (di-rename ke `IG_APP_SECRET` / `IG_VERIFY_TOKEN`)
- `IG_ACCESS_TOKEN`, `IG_ACCOUNT_USER_ID` (worker, per-akun sekarang dari Postgres via `apps/api/internal/connect`)
- `IG_ACCOUNT_ID` (api, sekarang di-resolve dari `entry.id` webhook via `GetAccountByIgUserID`)

## 2. Setup Meta App Dashboard (produk Instagram, bukan Facebook Login)

1. Buat/pilih App di [developers.facebook.com](https://developers.facebook.com/apps).
2. Tambahkan produk **Instagram** (bukan **Facebook Login** atau **Messenger**).
   Ini penting: seluruh flow OAuth Zosmed memakai *Business Login for
   Instagram*, host `graph.instagram.com` (CLAUDE.md §4.0), bukan Page token.
3. Di pengaturan produk Instagram, set:
   - **Valid OAuth Redirect URIs**: sama persis dengan `IG_REDIRECT_URI`
     (mis. `https://api.zosmed.example.com/connect/instagram/callback`).
   - **Deauthorize callback URL** / **Data deletion callback URL**: sesuai
     kebijakan produk (di luar scope ADR-002).
4. Catat **Instagram App ID** dan **Instagram App Secret** → isi
   `IG_APP_ID` / `IG_APP_SECRET`.

## 3. Langganan webhook via produk Instagram

Berlangganan field webhook dilakukan lewat **produk Instagram**, bukan
Messenger/Facebook (RESOLVED G11):

```
POST https://graph.instagram.com/v25.0/me/subscribed_apps
  ?subscribed_fields=comments,messages
  &access_token=<IG-user-access-token-akun-yang-connect>
```

Field yang didukung Zosmed (CLAUDE.md §4a — jangan berlangganan field lain):
`comments`, `messages`, `mentions`.

Verifikasi handshake (`GET /webhooks/meta`) memakai `IG_VERIFY_TOKEN`; body
webhook diverifikasi dengan HMAC-SHA256 `X-Hub-Signature-256` memakai
`IG_APP_SECRET` yang sama (RESOLVED G11 — tidak ada secret terpisah untuk Page).

## 4. Menghubungkan akun (connect flow)

Sejak ADR-003, `GET /connect/instagram` **butuh login Zosmed** terlebih
dahulu (lihat §4a di bawah) — identitas user yang login ditautkan ke akun IG
lewat signed state, bukan sekadar CSRF nonce.

Setelah App terdaftar, server berjalan, dan user sudah login:

1. Arahkan pengguna ke `GET /connect/instagram` (di belakang cookie sesi
   `zsid` — 401 tanpa login). Handler membaca user dari context, membangun
   signed state yang membawa `user_id`, lalu redirect ke layar consent
   Instagram (ADR-002 §3 / ADR-003 §6).
2. Setelah pengguna login + izinkan akses, Instagram redirect balik ke
   `GET /connect/instagram/callback?code=...&state=...` (endpoint ini tetap
   publik — Instagram yang memanggilnya, bukan browser dengan cookie sesi).
3. Server memverifikasi state → mengekstrak `user_id` → menukar `code` →
   token pendek → token panjang (~60 hari) → `GET /me` untuk IGSID, lalu
   menyimpan semuanya ke tabel `account` (kolom `access_token`,
   `token_expires_at`, `user_id`, dll — lihat
   `db/migrations/00007_account_tokens.sql` dan `00010_account_user.sql`).
4. Redirect setelah connect tergantung status onboarding user: belum
   selesai → `/onboarding?connected=1`; sudah selesai (re-connect dari
   Settings) → `/settings?connected=1`.
5. Token diperpanjang otomatis setiap ~6 jam oleh `apps/worker` (task
   periodic `token:refresh-sweep`, ADR-002 §5). Refresh gagal → akun ditandai
   `status = 'expired'` (bukan crash) dan berhenti mengirim outbound sampai
   pengguna connect ulang.

> ⚠️ **Scheduler = SATU instance.** `apps/worker` menjalankan `asynq.Scheduler`
> untuk task periodic (`token:refresh-sweep` tiap ~6 jam, `reservation:reconcile`
> tiap 1 menit). Bila worker di-scale >1 replika, jalankan scheduler hanya di
> SATU instance (atau di balik leader-lock) agar task periodic tidak ter-enqueue
> ganda. Task-nya idempotent (aman), tapi enqueue ganda itu mubazir.

## 4a. Login + Onboarding Zosmed (ADR-003)

Terpisah dari identitas akun Instagram (§4): `app_user` adalah identitas
login Zosmed sendiri (email + password, sesi server-side). Lihat
`docs/specs/auth-login-onboarding.md` untuk detail penuh.

- **Endpoint:** `POST /api/v1/auth/{register,login,logout}` (publik),
  `GET /api/v1/auth/me` (butuh cookie), `PUT /api/v1/onboarding/segment` +
  `POST /api/v1/onboarding/complete` (butuh cookie).
- **Sesi:** cookie httpOnly `zsid` (opaque token 32 byte; hanya SHA-256-nya
  yang disimpan di tabel `user_session`), `SameSite=Lax`, `Secure` mengikuti
  `APP_ENV=prod`, TTL 30 hari.
- **Onboarding selesai** = `app_user.segment` terisi **dan** akun IG
  berstatus `connected`, di-stamp lewat `POST /api/v1/onboarding/complete`
  (409 `onboarding_incomplete` + `reason` bila salah satu belum lengkap).

### Seed data dev

`apps/api/cmd/seed` mengisi database dev dengan user demo, akun IG demo
(token dummy — **tidak akan bisa memanggil Graph API sungguhan**), dan satu
post + dua produk contoh untuk Comment-to-Order. Idempotent — aman dijalankan
berulang kali.

```bash
# dari root repo, dengan .env sudah di-source / env var di atas ter-set
go run ./apps/api/cmd/seed            # onboarding user demo dibiarkan belum selesai
go run ./apps/api/cmd/seed -complete  # user demo langsung "onboarded" (segment=seller)
```

Login yang dihasilkan: **`demo@zosmed.test` / `demo12345`** (kredensial
dummy, jangan dipakai di luar dev). Seed **menolak jalan** bila
`APP_ENV=prod`.

## 5. Checklist App Review (sebelum produksi)

Mode standard access (dev/test users) cukup untuk development, TAPI
produksi butuh **App Review** untuk scope berikut (RESOLVED G7):

- [ ] `instagram_business_basic` — biasanya standard access
- [ ] `instagram_business_manage_comments` — **Advanced Access, App Review wajib**
- [ ] `instagram_business_manage_messages` — **Advanced Access, App Review wajib**
- [ ] Uji end-to-end dengan **test user** IG Business/Creator sebelum submit review
- [ ] Siapkan screencast + use-case description sesuai kebutuhan reviewer Meta
      (comment-to-order, private reply, WA handoff — CLAUDE.md §7/§8.1)
- [ ] Verifikasi rate-limit sebenarnya via header `X-Business-Use-Case-Usage`
      setelah App Review (angka default CLAUDE.md §4c dipakai sementara)
- [ ] Validasi payload webhook `comments` di sandbox: field `id` vs
      `comment_id` (ADR-002 §6.3/G10) — parser sudah toleran, tapi konfirmasi
      shape asli sebelum go-live

## 6. Catatan keamanan

- Token IG-user tersimpan sebagai plaintext di kolom `account.access_token`
  untuk MVP (ADR-002 §4.4 — enkripsi at-rest adalah hardening follow-up,
  bukan blocker fungsional). Batasi akses DB produksi sampai hardening ini
  selesai.
- Token tidak pernah masuk payload asynq/Redis — worker selalu lookup
  token dari Postgres per task (ADR-002 §6.2).
