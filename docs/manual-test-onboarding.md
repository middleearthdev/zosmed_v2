# Skenario Tes Manual — Login & Onboarding (ADR-003)

Panduan menguji alur **login → onboarding** end-to-end di lokal. Setup & cara menjalankan ada di [`how-to-run.md`](./how-to-run.md).

## Prasyarat

- API (`:8080`) + Web (`:3000`) jalan, Postgres nyala, `APP_ENV=dev`.
- **Migrasi + seed sudah dijalankan** (`go run ./apps/api/cmd/seed`).

## Kenapa bisa tes tanpa Instagram sungguhan

Seed membuat user demo yang **akun Instagram-nya sudah `connected`** (token dummy). Jadi seluruh onboarding (pilih segmen → selesai) bisa diuji **tanpa** OAuth Instagram nyata. Login demo:

- **Email:** `demo@zosmed.test`
- **Password:** `demo12345`
- Kondisi awal: `segment = null`, `onboardingCompleted = false`, akun `@olshop.aurora` `connected`.

---

## Skenario UI (via browser di http://localhost:3000)

### A. Guard route — belum login
1. Buka `http://localhost:3000/dashboard` (atau `/`).
2. **Ekspektasi:** redirect ke **`/login`**.

### B. Login gagal (password salah)
1. Di `/login`, isi `demo@zosmed.test` + password **salah**.
2. **Ekspektasi:** tetap di `/login`, muncul pesan **"Email atau password salah"** (HTTP 401). Pesan & waktu respons sama untuk "email tak ada" maupun "password salah" (anti user-enumeration).

### C. Login sukses → masuk onboarding
1. Login `demo@zosmed.test` / `demo12345`.
2. **Ekspektasi:** redirect ke **`/onboarding`** (karena onboarding belum selesai). Cookie **`zsid`** terset — cek DevTools → Application → Cookies → `HttpOnly ✓`, `SameSite=Lax`.

### D. Onboarding — alur inti
1. **Step 1 — pilih segmen:** klik **"Jualan"** (Creator/Booking harus tampil "SOON"/disabled).
   - **Ekspektasi:** tersimpan (`PUT /api/v1/onboarding/segment`), lanjut ke step berikut.
2. **Step 2 — Hubungkan Instagram:** karena seed sudah menautkan akun, status tampil **"Terhubung" `@olshop.aurora`** (pill lime).
   - **Ekspektasi:** tidak perlu klik connect / OAuth. *(Jika diklik, akan redirect ke instagram.com sungguhan dan gagal tanpa Meta app — jadi lewati.)*
3. **Step 3 — Selesaikan:** klik tombol selesai/"Go live".
   - **Ekspektasi:** segmen terisi **DAN** akun `connected` → `POST /api/v1/onboarding/complete` sukses → redirect ke **`/dashboard`**.

### E. Guard membalik setelah selesai
1. Sudah login + onboarding selesai, buka `/login` atau `/onboarding`.
   - **Ekspektasi:** redirect ke **`/dashboard`**.
2. Klik **"Keluar"** (sidebar).
   - **Ekspektasi:** sesi di-revoke, cookie `zsid` hilang → redirect ke **`/login`**. Buka `/dashboard` lagi → kembali ke `/login`.

### F. Register user baru (batasan)
1. `/register` → daftar email baru → otomatis login → ke **`/onboarding`**.
2. **Batasan:** user baru **belum** punya akun IG connected. Step "Hubungkan Instagram" me-redirect ke **instagram.com sungguhan**; tanpa Meta app + test user + redirect publik (mis. ngrok), tak bisa selesai — `complete` akan **409 `account_not_connected`**.
   - **Untuk menguji onboarding tuntas tanpa Instagram nyata → pakai user seed (skenario C–E).**

---

## Skenario API (via curl — smoke test backend)

```bash
# 1. Login → simpan cookie ke cookies.txt
curl -i -c cookies.txt -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@zosmed.test","password":"demo12345"}'
# Ekspektasi: 200, header Set-Cookie: zsid=...; HttpOnly

# 2. Siapa saya (pakai cookie)
curl -s -b cookies.txt localhost:8080/api/v1/auth/me | jq
# Ekspektasi: {data:{user:{email,segment,onboardingCompleted},account:{status:"connected",handle:"olshop.aurora",...}},error:null}

# 3. Tanpa cookie → 401
curl -i localhost:8080/api/v1/auth/me
# Ekspektasi: 401 unauthorized

# 4. Password salah → 401 invalid_credentials
curl -i -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@zosmed.test","password":"salah"}'

# 5. Onboarding: set segmen → complete
curl -s -b cookies.txt -X PUT localhost:8080/api/v1/onboarding/segment \
  -H 'Content-Type: application/json' -d '{"segment":"seller"}' | jq
curl -s -b cookies.txt -X POST localhost:8080/api/v1/onboarding/complete | jq
# Ekspektasi: 200 {user: onboardingCompleted:true}  (akun seed sudah connected)

# 6. Logout → sesi revoke
curl -i -b cookies.txt -X POST localhost:8080/api/v1/auth/logout
curl -i -b cookies.txt localhost:8080/api/v1/auth/me   # sekarang 401
```

### Validasi tambahan (opsional)
```bash
# Register password >72 byte → 400 (bukan 500)
curl -i -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"x@zosmed.test\",\"password\":\"$(python3 -c 'print("a"*73)')\"}"

# Endpoint sensitif tanpa login → 401
curl -i localhost:8080/api/v1/comment-order
```

---

## Skenario via Postman

Import `deploy/zosmed.postman_collection.json`, folder **Auth & Onboarding**. Cookie `zsid` ditangani otomatis oleh cookie jar Postman.

Urutan: **Login** → **Get Me** (segment null, account connected) → **Set Segment** (seller) → **Complete Onboarding** (200) → **Get Me** (onboardingCompleted true) → **Logout** → **Get Me** (401).

---

## Reset ke kondisi awal (ulangi tes)

```bash
# hapus user + sesi, seed ulang (akun/katalog tetap idempotent)
psql "$DB_URL" -c 'TRUNCATE app_user, user_session CASCADE;'
go run ./apps/api/cmd/seed
```
Atau reset penuh: lihat [`how-to-run.md`](./how-to-run.md) §6.

---

## Ringkasan ekspektasi

| # | Aksi | Ekspektasi |
|---|---|---|
| A | Akses `/dashboard` belum login | Redirect `/login` |
| B | Login password salah | 401, "Email atau password salah" |
| C | Login demo sukses | Redirect `/onboarding`, cookie `zsid` HttpOnly |
| D | Pilih segmen → selesaikan | `/dashboard` |
| E | Buka `/login` saat sudah selesai | Redirect `/dashboard`; logout → `/login` |
| F | Register user baru | Onboarding jalan, tapi connect butuh IG nyata (409 tanpa itu) |
