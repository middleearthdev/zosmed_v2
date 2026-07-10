# ADR-003 — Login + Onboarding terintegrasi backend (auth Zosmed)

Status: Proposed — siap dieksekusi (menunggu jawab konfirmasi §13)
Tanggal: 2026-07-05
Penulis: System Architect (Zosmed)
Scope: Mengganti `AuthStub` (NO-OP) dengan autentikasi user Zosmed sungguhan, dan mengubah Onboarding dari mock jadi terwire ke backend (persist segmen + status koneksi akun). Menyediakan seed dev agar login + onboarding bisa diselesaikan end-to-end tanpa OAuth Instagram sungguhan.
Referensi: CLAUDE.md §4.0 (Instagram Login), §5/§5a (arsitektur/monorepo), §6 (domain Account), §9 (layar Onboarding "pilih segmen → connect IG"), §11 (design token), §12a (prinsip coding). Menumpang fondasi ADR-002 (connect flow, tabel `account`, envelope respons, dbgen).

> Konteks penting: ADR-002 sudah membangun `connect` flow OAuth Instagram, tabel `account` (dengan token), envelope `{data,error}` (`httpx.respond`), dan pola `dbgen`/sqlc. ADR ini **tidak** menyentuh integrasi Instagram (§4) — murni menambah lapisan identitas user Zosmed **di atas** yang sudah ada, plus menautkan akun IG hasil OAuth ke user yang login.

---

## 0. Ringkasan Keputusan

Enam keputusan mengikat:

1. **Model auth = email + password, sesi server-side.** Password di-hash **bcrypt** (cost 12) di `apps/api/internal/auth`. Sesi = **opaque random token** (32 byte) disimpan **hash-nya** (SHA-256) di tabel `user_session`; token mentah dikirim ke browser lewat **cookie httpOnly** (`zsid`, `SameSite=Lax`, `Secure` di prod, `Path=/`). Bukan JWT — kita butuh **logout/revoke** dan tidak mau menyebar secret signing baru (state.go sudah pakai App Secret untuk CSRF; sesi ≠ CSRF, jadi jangan campur).

2. **Dua entitas terpisah: `app_user` (yang login) vs `account` (akun IG).** Relasi **1 user → 0..1 account** (MVP satu akun IG per user; multi-akun = fase lanjutan). `account.user_id` = FK **nullable** ke `app_user(id)`. Segmen bisnis (`seller`/`creator`/`booking`) milik **user**, bukan account — dipilih saat onboarding **sebelum** connect IG.

3. **Onboarding = state di `app_user`.** `app_user.segment` (nullable sampai dipilih) + `app_user.onboarding_completed_at` (nullable). Onboarding "selesai" bila segmen terisi **dan** account connected, lalu di-stamp lewat `POST /api/v1/onboarding/complete`. FE menentukan tujuan redirect dari `GET /api/v1/auth/me`.

4. **`AuthStub` diganti `RequireUser`.** Middleware baca cookie → lookup `user_session` (join `app_user`) → inject user ke `context`. Dipasang di grup `/api/v1/*` (kecuali sub-grup `/api/v1/auth/*` yang publik). `GET /connect/instagram` (Start) **kini butuh login** agar `user_id` bisa ditautkan ke akun IG; `Callback` tetap publik (identitas user dibawa dalam **signed state**, bukan cookie — Instagram yang memanggil balik).

5. **Same-origin via Next rewrite (bukan CORS cookie lintas situs).** FE (`:3000`) mem-proxy `/api/*` dan `/connect/*` ke API (`:8080`) lewat `next.config` rewrites, sehingga cookie sesi **first-party** (`SameSite=Lax` cukup, tak perlu `SameSite=None`+CORS credentials yang rapuh). Satu keputusan wiring yang menyederhanakan seluruh flow cookie.

6. **Seed dev = program Go `apps/api/cmd/seed`.** Meng-hash password (butuh kode, tak bisa SQL murni) dan meng-upsert: 1 user demo (`demo@zosmed.test` / password diketahui), 1 account IG demo (`status=connected`, token **dummy**, `user_id` = user demo), plus catalog_post + product contoh agar Comment-to-Order & onboarding bisa dites tanpa OAuth sungguhan. Idempotent. **Tanpa secret asli.**

### Acceptance Criteria (Definition of Done §14) — checklist dapat di-grep

- [ ] **AC-1** Tabel `app_user(id, email UNIQUE, password_hash, segment NULL, onboarding_completed_at NULL, created_at)` ada via migrasi goose; `email` unik case-insensitive (disimpan lower-case). `password_hash` **tidak pernah** muncul di response JSON mana pun (`grep -rn "password_hash" apps/web packages` = 0).
- [ ] **AC-2** Tabel `user_session(token_hash PK, user_id FK, created_at, expires_at)`; token **mentah** tak pernah disimpan (hanya SHA-256). `grep -rn "token_hash" db/query/session.sql` ada; kolom bernama `token` mentah = tidak ada.
- [ ] **AC-3** `account.user_id uuid REFERENCES app_user(id)` (nullable) ditambah via migrasi; connect callback mengisi `user_id` dari state terverifikasi.
- [ ] **AC-4** Password di-hash **bcrypt** (`golang.org/x/crypto/bcrypt`, cost ≥12) di `apps/api/internal/auth/password.go`; plaintext password **tidak** pernah di-log (`grep` audit) & tidak masuk payload task/DB.
- [ ] **AC-5** Endpoint: `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me` berjalan dengan envelope `{data,error}` (pola `httpx.JSON`/`httpx.Err`). Login sukses → `Set-Cookie: zsid=...; HttpOnly; SameSite=Lax; Path=/`.
- [ ] **AC-6** `RequireUser` menggantikan `AuthStub`: request ke `/api/v1/comment-order` tanpa cookie valid → `401 {error:{code:"unauthorized"}}`. `grep -n "AuthStub" apps/api` = 0 (dihapus, bukan dibiarkan).
- [ ] **AC-7** `PUT /api/v1/onboarding/segment` menyimpan `app_user.segment` (validasi enum `seller|creator|booking`); `POST /api/v1/onboarding/complete` menolak (`409 onboarding_incomplete`) bila segmen kosong atau account belum `connected`, dan men-stamp `onboarding_completed_at` bila lengkap.
- [ ] **AC-8** `GET /api/v1/auth/me` mengembalikan `{user:{id,email,segment,onboardingCompleted}, account:{status,handle,displayName}|null}` — **tanpa** `password_hash`/`access_token`.
- [ ] **AC-9** `GET /connect/instagram` di belakang `RequireUser`; `user_id` di-embed ke signed state (`connect/state.go`) dan diekstrak di `Callback` → `UpsertAccountFromOAuth` menyimpan `user_id`. Callback tetap publik.
- [ ] **AC-10** FE: `middleware.ts` (kini `proxy.ts`, ADR-008) redirect ke `/login` untuk path terproteksi tanpa cookie sesi; `(app)/layout.tsx` fetch `/auth/me` → `/login` bila 401, `/onboarding` bila `onboardingCompleted=false`, render bila lengkap. Path `/login`+cookie valid → redirect `/dashboard`.
- [ ] **AC-11** FE Onboarding terwire: pilih segmen → `PUT /onboarding/segment`; tombol "Hubungkan Instagram" → `/connect/instagram` (real, ADR-002); status akun dari `/auth/me` (bukan `mockAccount`); tombol selesai → `/onboarding/complete` → redirect `/dashboard`. Copy Bahasa Indonesia (§11/§12).
- [ ] **AC-12** `next.config` rewrites mem-proxy `/api/*` & `/connect/*` ke API base; cookie sesi first-party. Tidak ada `SameSite=None` di kode.
- [ ] **AC-13** `go run ./apps/api/cmd/seed` (dari root) idempotent membuat user demo + account demo (connected, token dummy, tertaut user) + catalog/product; menjalankan dua kali tidak menduplikasi/eror. Token dummy bukan kredensial asli.
- [ ] **AC-14** Boundary §5a/§12a: logika auth di `apps/api/internal/auth` (handler↔business↔store terpisah); handler tak menulis SQL (lewat `store.go`→`dbgen`); engine (`libs/workflow`/`libs/safety`) & `libs/igapi` tak tersentuh; `packages/types` diperbarui selaras (`MeResponse` dll) dalam commit yang sama.
- [ ] **AC-15** §4 bersih: tidak ada kemampuan IG baru; tidak menyentuh §4b. Auth murni identitas internal Zosmed.

---

## 1. Scope & Non-Goals

**In scope:** tabel `app_user`/`user_session` + FK `account.user_id`; hashing bcrypt; sesi cookie; endpoint register/login/logout/me; onboarding segment + complete; `RequireUser` menggantikan `AuthStub`; menautkan user ke akun IG via state OAuth; guard route FE + rewrites; seed dev.

**Non-goals (jangan dikerjakan di ADR ini):**
- **OAuth sosial / magic link / OTP** — email+password cukup untuk MVP. Ditandai fase lanjutan.
- **Reset password / verifikasi email / lupa password** — follow-up (butuh email sender). MVP: register langsung aktif.
- **Multi-akun IG per user & multi-tenant/team** — MVP satu akun per user; `team` di §9 fase lanjutan.
- **Rate-limit brute-force / lockout login** — dicatat sebagai hardening (§11), tidak memblok MVP.
- **Enkripsi token IG at-rest** — sudah dicatat non-goal di ADR-002 §1; tetap.
- **Payment/billing gating** — di luar scope (§13 CLAUDE).

---

## 2. Model Data & Migrasi

### 2.1 Keadaan sekarang
`account` (ADR-001/002): `id, ig_user_id UNIQUE, handle, display_name, status, created_at, access_token, token_type, scopes, token_expires_at, token_refreshed_at`. **Belum ada** konsep user Zosmed. FE `mockAccount.kit='seller'` — segmen belum dipersist di mana pun.

### 2.2 Migrasi baru (goose; nama tabel hindari kata kunci Postgres → `app_user`, `user_session`)

**`db/migrations/00008_app_user.sql`**
```sql
-- +goose Up
CREATE TABLE app_user (
    id                     uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  text        NOT NULL UNIQUE,       -- disimpan lower-case (normalisasi di app)
    password_hash          text        NOT NULL,              -- bcrypt; tak pernah dikirim ke FE
    segment                text            CHECK (segment IN ('seller','creator','booking')), -- NULL sblm onboarding
    onboarding_completed_at timestamptz,                       -- NULL = onboarding belum selesai
    created_at             timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE app_user;
```

**`db/migrations/00009_user_session.sql`**
```sql
-- +goose Up
CREATE TABLE user_session (
    token_hash text        PRIMARY KEY,                        -- SHA-256(raw token); raw token hanya di cookie
    user_id    uuid        NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX user_session_user_idx ON user_session (user_id);   -- logout-all / cleanup per user
-- +goose Down
DROP TABLE user_session;
```

**`db/migrations/00010_account_user.sql`**
```sql
-- +goose Up
-- Tautkan akun IG (hasil OAuth) ke user Zosmed. Nullable: akun lama/seed bisa
-- ada sebelum ada user; connect callback mengisi user_id dari signed state.
ALTER TABLE account
    ADD COLUMN user_id uuid REFERENCES app_user(id) ON DELETE SET NULL;
CREATE INDEX account_user_idx ON account (user_id);
-- +goose Down
DROP INDEX account_user_idx;
ALTER TABLE account DROP COLUMN user_id;
```

> **Relasi:** MVP 1 user ↔ 0..1 account. Tidak dibuat UNIQUE pada `account.user_id` sekarang (fase lanjutan multi-akun); aplikasi menjaga invarian "satu akun per user" via `GetAccountByUserID` (ambil satu). Bila ingin ketat MVP, tambah `UNIQUE(user_id)` — **PERLU KONFIRMASI §13**.

### 2.3 Query sqlc (baru + modifikasi) → regenerasi `dbgen`

**`db/query/auth.sql` (BARU)**
```sql
-- name: CreateUser :one
INSERT INTO app_user (email, password_hash) VALUES (@email, @password_hash) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM app_user WHERE email = @email;

-- name: GetUserByID :one
SELECT * FROM app_user WHERE id = @id;

-- name: SetUserSegment :one
UPDATE app_user SET segment = @segment WHERE id = @id RETURNING *;

-- name: CompleteOnboarding :one
UPDATE app_user SET onboarding_completed_at = now() WHERE id = @id RETURNING *;
```

**`db/query/session.sql` (BARU)**
```sql
-- name: CreateSession :exec
INSERT INTO user_session (token_hash, user_id, expires_at) VALUES (@token_hash, @user_id, @expires_at);

-- name: GetSessionUser :one
-- Session lookup untuk RequireUser: token_hash → user (join), tolak yang kadaluarsa.
SELECT u.* FROM user_session s
JOIN app_user u ON u.id = s.user_id
WHERE s.token_hash = @token_hash AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM user_session WHERE token_hash = @token_hash;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_session WHERE expires_at <= now();
```

**`db/query/account.sql` (MODIFIKASI)**
- `UpsertAccountFromOAuth` → **tambah kolom `user_id`** (`@user_id`) di INSERT dan `DO UPDATE SET user_id = EXCLUDED.user_id`. Callers (connect store) mengoper `user_id`.
- Tambah:
```sql
-- name: GetAccountByUserID :one
-- Untuk /auth/me: satu akun IG milik user (MVP satu akun). LIMIT 1.
SELECT * FROM account WHERE user_id = @user_id ORDER BY created_at ASC LIMIT 1;
```

> Setelah edit query/migrasi: `cd db && sqlc generate` (out: `libs/platform/dbgen`). Verifikasi struct `AppUser`, `UserSession` tergenerate.

---

## 3. Backend — Paket `apps/api/internal/auth` (BARU)

SoC ketat (§12a-3): handler (transport) ↔ business (hash/sesi) ↔ store (dbgen). Handler tak menulis SQL.

```
apps/api/internal/auth/
├── password.go     # Hash(pw) / Verify(hash, pw) — bcrypt cost 12. Satu-satunya tempat hashing.
├── session.go      # newToken() (32B rand), hashToken() (sha256), TTL const, setCookie/clearCookie,
│                   # readCookie(r). Nama cookie "zsid" sebagai konstanta (DRY).
├── store.go        # Store adapter dbgen: CreateUser, UserByEmail, UserByID, CreateSession,
│                   #   SessionUser, DeleteSession, SetSegment, Complete, AccountByUserID.
├── middleware.go   # RequireUser(next) — baca cookie → SessionUser → inject ctx. UserFromContext(ctx).
├── handler.go      # Register, Login, Logout, Me  (transport + envelope)
├── onboarding.go   # PutSegment, Complete       (transport + envelope)
└── dto.go          # request/response shapes (selaras packages/types)
```

**Titik desain kunci:**
- `session.go` — cookie flags dari config (`Secure` = `AppEnv=="prod"`). TTL sesi: **30 hari** (konstanta `sessionTTL`). Token mentah = `base64url(32 rand bytes)`; simpan `sha256(token)` di DB, kirim mentah di cookie. Verifikasi = hash cookie lalu `GetSessionUser`.
- `middleware.go` — `RequireUser` gagal (no cookie / sesi invalid/kadaluarsa) → `httpx.Err(w,401,"unauthorized",...)`. Sukses → `context.WithValue(user)`. `UserFromContext` dipakai handler `/api/v1/*` **dan** connect Start.
- **Anti over-abstraction (§12a-4):** tidak ada interface `AuthService`/`TokenProvider` generik — cukup fungsi & satu `Store` konkret. Interface muncul bila ada implementasi kedua (belum ada).
- **Reuse:** tidak menyalin logika HMAC dari `connect/state.go` — itu untuk CSRF (stateless), sesi kita stateful (revocable). Dua kebutuhan berbeda; jangan paksa DRY yang bukan konsep sama (§12a-4).

---

## 4. Kontrak Endpoint (FE ↔ BE)

Semua memakai envelope `{data,error}` (`httpx.JSON`/`httpx.Err`). Publik = tanpa `RequireUser`.

### 4.1 Auth (publik kecuali `/me`)

| Method | Path | Auth | Body | Sukses (data) |
|---|---|---|---|---|
| POST | `/api/v1/auth/register` | publik | `{email, password, segment?}` | `201` `{user}` + `Set-Cookie zsid` |
| POST | `/api/v1/auth/login` | publik | `{email, password}` | `200` `{user, account}` + `Set-Cookie zsid` |
| POST | `/api/v1/auth/logout` | cookie | — | `200` `{ok:true}` + cookie di-expire |
| GET | `/api/v1/auth/me` | `RequireUser` | — | `200` `{user, account}` / `401` |

**Error:** `login` gagal → `401 {code:"invalid_credentials", message:"Email atau password salah"}`. `register` email dipakai → `409 {code:"email_taken"}`. Body invalid → `400 {code:"invalid_request"}`.

### 4.2 Onboarding (semua `RequireUser`)

| Method | Path | Body | Sukses |
|---|---|---|---|
| PUT | `/api/v1/onboarding/segment` | `{segment:"seller"\|"creator"\|"booking"}` | `200 {user}` |
| POST | `/api/v1/onboarding/complete` | — | `200 {user}` / `409 {code:"onboarding_incomplete", message}` |

`complete` menolak bila `segment==null` (`reason:"segment_missing"`) atau account bukan `connected` (`reason:"account_not_connected"`) — FE menampilkan langkah yang kurang.

### 4.3 Shape JSON (kanonik — dicerminkan di `packages/types`)

```jsonc
// user
{ "id":"uuid", "email":"demo@zosmed.test", "segment":"seller"|null, "onboardingCompleted": true }
// account (null bila belum connect)
{ "status":"connected"|"expired"|"disconnected", "handle":"olshop.aurora", "displayName":"Aurora Olshop" }
// GET /auth/me
{ "data": { "user": {..}, "account": {..}|null }, "error": null }
```

> **Tidak pernah** disertakan: `password_hash`, `access_token`, `token_*`. Store menyeleksi field aman saat memetakan ke DTO (jangan kirim row `dbgen.Account`/`dbgen.AppUser` mentah).

---

## 5. Diagram Alur

### 5.1 Login
```
Browser            Next (/login)        API /api/v1/auth        Postgres
  │  isi form          │                      │                    │
  ├───────────────────►│ POST /api/v1/auth/login (via rewrite)     │
  │                     ├─────────────────────►│ GetUserByEmail ───►│
  │                     │                      │◄── user + hash ────┤
  │                     │                      │ bcrypt.Verify      │
  │                     │                      │ newToken()         │
  │                     │                      │ CreateSession ────►│ (token_hash)
  │  Set-Cookie zsid    │◄── 200 {user,account}│ + Set-Cookie       │
  │◄────────────────────┤                      │                    │
  │  redirect: onboardingCompleted? /dashboard : /onboarding        │
```

### 5.2 Onboarding (pilih segmen → connect IG → selesai)
```
(app diproteksi RequireUser + guard FE)

  /onboarding ──GET /auth/me──► {user.segment, account.status}
     │
     ├─ pilih "Jualan/Edukasi/Jasa" ──PUT /onboarding/segment──► app_user.segment
     │
     ├─ "Hubungkan Instagram" ──GET /connect/instagram (RequireUser)
     │        └─► state=sign(nonce.ts.userID) ─► instagram.com/oauth/authorize
     │                                             └─► callback ─► UpsertAccount(user_id) ─► /onboarding?connected=1
     │
     └─ "Selesai" ──POST /onboarding/complete──►  cek segment≠null & account.connected
                                                    ├─ ok  → stamp onboarding_completed_at → /dashboard
                                                    └─ belum → 409 (tampilkan langkah kurang)
```

### 5.3 Guard route (setiap request)
```
                 ┌─ cookie zsid ada? ── tidak ─► redirect /login
 request ──────► │
                 └─ ya ─► (app)/layout: GET /auth/me
                              ├─ 401 ─────────────► redirect /login (cookie basi)
                              ├─ onboardingCompleted=false ─► redirect /onboarding
                              └─ true ─────────────► render app
 (/login|/register + cookie valid ─► redirect /dashboard)
```

---

## 6. Perubahan Router & Wiring (`apps/api`)

- **`httpx/middleware.go`** — hapus `AuthStub` (AC-6). `RequireUser` tinggal di `internal/auth` (butuh Store; tak boleh di `httpx` yang bebas-dependency DB).
- **`httpx/router.go`** — `Routes` +field: `Register, Login, Logout, Me, PutSegment, CompleteOnboarding http.HandlerFunc`, dan `RequireUser func(http.Handler) http.Handler` (di-inject dari main agar `httpx` tak impor `auth`/DB — hindari import cycle, pola sama seperti handler lain di-inject sebagai `http.HandlerFunc`).
  - Auth routes publik: `r.Post("/api/v1/auth/register")`, `login`, `logout` (logout boleh publik; no-op bila tak ada cookie).
  - Grup terproteksi: `r.Group(func(r){ r.Use(routes.RequireUser); r.Get("/api/v1/auth/me"...); comment-order & reservations & settings & onboarding... })`.
  - `GET /connect/instagram` **dipindah ke belakang `RequireUser`** (butuh user); `GET /connect/instagram/callback` **tetap publik**.
- **`cmd/api/main.go`** — bangun `authStore := auth.NewStore(queries)`, `authHandler := auth.New(authStore, cfg, log)`, `requireUser := auth.RequireUser(authStore)`. Inject ke `httpx.Routes`. Connect `Start` sekarang membaca `auth.UserFromContext`.
- **`connect/state.go`** — `NewState(appSecret, userID string)` → format `"<nonce>.<ts>.<userID>.<hmac>"`; `VerifyState(state, appSecret) (userID string, ok bool)`. HMAC menutupi `nonce.ts.userID`.
- **`connect/handler.go`** — `Start`: `user := auth.UserFromContext(ctx)`; `NewState(secret, user.ID)`. `Callback`: `userID, ok := VerifyState(...)`; oper ke `UpsertAccountParams.UserID`. Redirect sukses → `/onboarding?connected=1` (bukan `/settings`) saat berasal dari onboarding — atau tetap `/settings?connected=1` untuk re-connect; **PERLU KONFIRMASI §13** (bisa pakai param `next` di state). Default aman: redirect `/onboarding?connected=1`.
- **`connect/store.go`** — `UpsertAccountParams` +`UserID string`; teruskan ke `dbgen.UpsertAccountFromOAuthParams.UserID` (pgtype.UUID).
- **`config.go`** — tambah `AppEnv` (`APP_ENV`, default `dev` → cookie `Secure=false`) & `WebBaseURL` (`WEB_BASE_URL`, untuk redirect absolut bila perlu). Keduanya opsional dengan default; tidak memaksa env baru wajib.

---

## 7. Seed Dev — `apps/api/cmd/seed/main.go` (BARU)

Program Go idempotent (butuh Go karena harus bcrypt-hash password). Dijalankan dari root: `go run ./apps/api/cmd/seed` (baca `DB_URL` dari env, pola sama `main.go`).

Isi:
1. **User demo** — email `demo@zosmed.test`, password `demo12345` (di-hash bcrypt). Upsert by email (skip bila ada). Cetak kredensial ke stdout untuk dev.
2. **Account demo** — `ig_user_id="SEED-IG-0001"`, `handle="olshop.aurora"`, `display_name="Aurora Olshop"`, `status='connected'`, `access_token="SEED-DUMMY-TOKEN-not-a-real-ig-token"`, `token_expires_at = now()+60d`, `scopes = DefaultScopes`, `user_id = user demo`. Pakai `UpsertAccountFromOAuth` (idempotent by `ig_user_id`).
3. **Catalog + product demo** (opsional, agar Comment-to-Order tampil): 1 `catalog_post` untuk account demo + 2 `product` (mis. `C1`,`C2`). Pakai query katalog yang ada; skip bila sudah ada.
4. Segmen user demo **sengaja dibiarkan NULL** + `onboarding_completed_at` NULL → memungkinkan menguji flow onboarding penuh. (Sediakan flag `-complete` untuk seed user yang sudah lolos onboarding bila perlu tes dashboard langsung.)

> **Keamanan seed:** token & password dummy jelas-bertanda; **jangan** dipakai di staging/prod. Catat di `deploy/README`: seed hanya untuk `APP_ENV=dev`. Program menolak jalan bila `APP_ENV=prod`.

---

## 8. Frontend (`apps/web`)

### 8.1 File-per-file
- **`next.config.(ts|mjs)`** (MODIF) — `rewrites()`: `{source:'/api/:path*', destination: '${API_BASE}/api/:path*'}` dan `{source:'/connect/:path*', destination:'${API_BASE}/connect/:path*'}`. Menjadikan cookie first-party (AC-12).
- **`middleware.ts`** (BARU) — cek cookie `zsid` untuk path terproteksi (`/dashboard`,`/workflows`,`/inbox`,... dan `/onboarding`) → tanpa cookie redirect `/login`. Path `/login`,`/register` + cookie ada → redirect `/dashboard`. (Coarse; verifikasi halus di layout.) _(Next 16 / ADR-008: file ini kini bernama **`proxy.ts`** dengan export `proxy`; `config.matcher` & logika guard identik.)_
- **`lib/auth.ts`** (BARU) — server helper `getMe()` (fetch `/api/v1/auth/me`, **forward cookie** via `next/headers cookies()` di server component) mengembalikan `MeResponse|null`; client actions `login/register/logout/saveSegment/completeOnboarding` (fetch `credentials:'include'`).
- **`app/login/page.tsx`** + **`app/login/LoginForm.tsx`** (BARU) — form email+password (client), pakai design token §11 (dark+lime), copy ID. Sukses → `router.push` sesuai `onboardingCompleted`.
- **`app/register/page.tsx`** + form (BARU, opsional bila register diaktifkan).
- **`app/onboarding/page.tsx`** (MODIF) — server: `getMe()`; render `OnboardingClient` dengan `user`/`account`. Hapus `getAccount()` mock + `mockAccount`.
- **`app/onboarding/OnboardingClient.tsx`** (BARU) — client: pilih segmen (`saveSegment`), tombol connect (`href=/connect/instagram`), tombol "Selesai" (`completeOnboarding` → push `/dashboard`); state langkah dari props. Reuse `InstagramConnectStatus` untuk status akun.
- **`app/(app)/layout.tsx`** (MODIF) — `getMe()`; `null`→`redirect('/login')`; `!onboardingCompleted`→`redirect('/onboarding')`; lalu `KitProvider segment={user.segment}` + `AppShell`. Hapus ketergantungan `getAccount()` mock untuk auth.
- **`lib/mock/api.ts`** (MODIF) — `getAccount()` diubah: ambil dari `getMe()` (real) dengan fallback mock hanya untuk layar non-auth (mengikuti pola `getCommentOrder`). Segmen/kit kini dari `user.segment`.
- **`packages/types/src/domain.ts`** (MODIF) + **`index.ts`** — tambah `AppUser`/`SessionUser` (`{id,email,segment:Segment|null,onboardingCompleted:boolean}`), `MeResponse` (`{user, account: AccountStatus|null}`), `AccountStatus` (`Pick<Account,'status'|'handle'|'displayName'>`). `Segment` sudah ada.

### 8.2 Catatan SoC FE (§12a-3)
- Data-fetching di `lib/auth.ts` & server components; komponen presentational (`LoginForm`, `OnboardingClient`) terima props, tak fetch di tengah JSX.
- Pemetaan status akun tetap lewat `lib/account-status.ts` (DRY, sudah ada) + reuse `InstagramConnectStatus`.

---

## 9. Boundary & SoC (§5a/§12a)

| Kapabilitas | Lokasi | Alasan |
|---|---|---|
| Hash/verify password, token/cookie sesi | `apps/api/internal/auth` (password.go/session.go) | Business auth milik app; satu pintu |
| RequireUser middleware | `apps/api/internal/auth/middleware.go` | Butuh Store(DB); tak boleh di `httpx` bebas-DB |
| Persist user/session/link akun | `db/query/*.sql` → `dbgen`; `auth/store.go`, `connect/store.go` | Data access; handler tak tulis SQL |
| Envelope respons, request-id, recover | `httpx` (tetap) | Transport util netral |
| OAuth wire IG | `libs/igapi` (tetap, tak tersentuh) | Netral; tak tahu user Zosmed |
| Segmen/Kit selection | `app_user.segment` + FE `KitProvider` | Segmen = atribut user, bukan engine |

Yang **tidak** berubah: `libs/workflow`, `libs/safety`, `libs/igapi`, `libs/kits/*`. Auth adalah lapisan identitas di `apps/api` + tabel baru; engine tetap netral.

---

## 10. Urutan Implementasi + Dependensi + Pemilik

**Fase A — DB & tipe (fondasi):**
1. Migrasi `00008`/`00009`/`00010` + modif `db/query/account.sql` + `auth.sql`/`session.sql`; `sqlc generate`. — **go-backend-engineer**
2. `packages/types`: `AppUser`/`MeResponse`/`AccountStatus`. — **frontend-ui-engineer** (koordinasi shape dgn #4 dalam satu commit, §5a)

**Fase B — Auth backend (setelah A1):**
3. `internal/auth/{password,session,store,middleware}.go`. — go-backend-engineer
4. `internal/auth/{handler,onboarding,dto}.go`. Bergantung: 3. — go-backend-engineer
5. Router+main wiring: hapus `AuthStub`, mount auth routes + `RequireUser`, pindah `/connect/instagram` ke belakang auth. Bergantung: 4. — go-backend-engineer

**Fase C — Tautkan user↔account (setelah A1 + B3):**
6. `connect/state.go` embed userID; `connect/handler.go` Start(ctx user)+Callback(userID); `connect/store.go` UserID. Bergantung: 5. — go-backend-engineer

**Fase D — Seed (setelah A1 + B3):**
7. `apps/api/cmd/seed/main.go` (user+account+catalog demo, idempotent, tolak prod). Bergantung: 3 (hashing) + query. — go-backend-engineer

**Fase E — Frontend (setelah B4 kontrak final):**
8. `next.config` rewrites + `middleware.ts` (kini `proxy.ts`, ADR-008) + `lib/auth.ts`. — frontend-ui-engineer
9. `app/login` (+register opsional). Bergantung: 8. — frontend-ui-engineer
10. `app/onboarding` rewire (`OnboardingClient`) + `(app)/layout.tsx` guard + `lib/mock/api.ts` getAccount real. Bergantung: 8, 2. — frontend-ui-engineer

**Fase F — Dok:**
11. `deploy/README`: env baru (`APP_ENV`,`WEB_BASE_URL`), cara seed, catatan cookie/rewrite. — go-backend-engineer

Jalur kritis: A1 → B(3,4,5) → C6. FE (E) menunggu kontrak B4 final. Seed (D) & FE (E) bisa paralel setelah B4.

---

## 11. Hardening follow-up (dicatat, di luar scope)
- Argon2id menggantikan bcrypt (bila diinginkan) — swap lokal di `password.go`.
- Rate-limit/lockout login (brute-force) + audit log percobaan gagal.
- Reset password + verifikasi email (butuh email sender).
- Rotasi sesi saat privilege change; `DeleteExpiredSessions` dijadwalkan (numpang asynq Scheduler ADR-002 §5).
- `UNIQUE(account.user_id)` bila memutuskan kunci ketat satu-akun-per-user.
- Enkripsi token IG at-rest (ADR-002 §4.4).

---

## 12. Alternatif yang Ditolak

- **JWT stateless (tanpa tabel sesi).** Ditolak: tak bisa logout/revoke tanpa blocklist (yang toh butuh state), dan menambah secret signing baru. Sesi server-side lebih sederhana-namun-benar untuk MVP + langsung mendukung logout (AC-5).
- **Pakai ulang HMAC `connect/state.go` untuk cookie sesi.** Ditolak: itu CSRF stateless (sekali-pakai, TTL 10 mnt); sesi butuh revocable & TTL panjang. Memaksa DRY dua konsep berbeda = kopling salah (§12a-4).
- **Simpan raw session token di DB.** Ditolak: kebocoran dump DB = pembajakan sesi. Simpan `sha256(token)`; cookie bawa mentah (AC-2).
- **Segmen disimpan di `account`, bukan `app_user`.** Ditolak: segmen dipilih **sebelum** connect IG (§9), dan milik user/bisnis, bukan akun IG. Menaruh di account membuat onboarding tak bisa maju tanpa OAuth.
- **CORS + cookie `SameSite=None` lintas origin (:3000↔:8080).** Ditolak: rapuh (butuh Secure+HTTPS di dev, preflight, `credentials:'include'` di tiap fetch). Next rewrites menjadikan first-party → `SameSite=Lax` cukup (AC-12).
- **Taruh `RequireUser` di `httpx`.** Ditolak: `httpx` harus bebas dependency DB (dipakai webhook/connect). Middleware butuh Store → tinggal di `internal/auth`, di-inject ke router sebagai `func(http.Handler)http.Handler` (hindari import cycle).
- **Seed via `db/seed/dev.sql` (psql).** Ditolak: password bcrypt tak bisa di-hash di SQL murni. Program Go `cmd/seed` meng-hash + idempotent + bisa menolak `APP_ENV=prod`.
- **Buat `AuthService` interface generik sekarang.** Ditolak (§12a-4): satu implementasi. Fungsi + satu Store konkret cukup.

---

## 13. PERLU KONFIRMASI (ke user, sebelum eksekusi)

1. **Register aktif atau invite-only?** MVP mengaktifkan `POST /auth/register` publik. Bila ingin dev-only (hanya seed), matikan route register & andalkan seed. (Default plan: register aktif.)
2. **Satu akun IG per user (ketat)?** Bila ya, tambah `UNIQUE(account.user_id)` di migrasi `00010`. Default plan: tidak ketat (siapkan multi-akun fase lanjutan).
3. **Redirect pasca-connect:** default `/onboarding?connected=1` (dari onboarding) — apakah re-connect dari Settings harus balik `/settings?connected=1`? (Bisa via param `next` di state; default aman: selalu `/onboarding` bila belum onboarding-complete, `/settings` bila sudah.)
4. **Kredensial seed** (`demo@zosmed.test` / `demo12345`) — cukup atau mau nilai lain?

---

## 14. Catatan untuk Engineer Berikutnya
- **Satu commit untuk kontrak selaras** (§5a): perubahan DTO backend (`auth/dto.go`) dan `packages/types` (`MeResponse`) harus sinkron.
- **Jangan pernah** kirim `dbgen.AppUser`/`dbgen.Account` mentah ke JSON — petakan ke DTO aman (buang `password_hash`/`access_token`). Ini AC-1/AC-8; grep saat review.
- **Jangan log** password/token. Login gagal cukup log email + "invalid_credentials", bukan password.
- `RequireUser` di-inject ke router sebagai fungsi (hindari import cycle `httpx`↔`auth`), pola sama seperti handler lain sudah `http.HandlerFunc`.
- Cookie: `HttpOnly` wajib, `Secure` mengikuti `APP_ENV`, `SameSite=Lax`, `Path=/`, `Max-Age`=TTL. Nama `zsid` sebagai konstanta (dipakai session.go + `proxy.ts` FE, dulu `middleware.ts` pra-ADR-008 — dokumentasikan agar konsisten).
- Server component FE **wajib forward cookie** ke `/auth/me` (`cookies()` dari `next/headers`), kalau tidak sesi tak terbaca di SSR.
- Setelah selesai: `grep -rn "AuthStub" apps/` = 0; `grep -rn "password_hash\|access_token" apps/web packages` = 0; `go run ./apps/api/cmd/seed` dua kali tanpa eror.
- Tidak menyentuh `libs/igapi`/engine (§4/§8). Ini murni identitas internal.
