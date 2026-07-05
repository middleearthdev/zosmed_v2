# Cara Menjalankan Zosmed (Dev Lokal)

Panduan menjalankan stack dev: **API** (`apps/api`), **Worker** (`apps/worker`), dan **Web** (`apps/web`). Untuk uji **login + onboarding**, worker tidak wajib.

> Skenario pengujian ada di [`manual-test-onboarding.md`](./manual-test-onboarding.md).

---

## 1. Prasyarat

| Kebutuhan | Cara siapkan | Wajib untuk |
|---|---|---|
| **PostgreSQL** (+ 1 database) | `brew services start postgresql` lalu `createdb zosmed` — atau Docker | Semua (API ping DB saat start) |
| **Redis** | `brew services start redis` — atau `docker run -p 6379:6379 redis` | Worker; API (koneksi lazy) |
| **Go** (workspace `go.work`) | sudah ada di repo | API, Worker, Seed |
| **goose** (migrasi DB) | `go install github.com/pressly/goose/v3/cmd/goose@latest` | Migrasi |
| **bun** (deps + dev server FE) | `curl -fsSL https://bun.sh/install \| bash` | Web |

> Tidak ada `docker-compose`/`Makefile` — Postgres & Redis kamu jalankan sendiri.

---

## 2. Environment variables

Aplikasi **Go tidak otomatis membaca `.env`** — variabel harus di-export ke shell sebelum `go run`. (Next/`apps/web` membaca `.env` sendiri, tidak perlu langkah ini.)

1. Salin & isi env (nilai IG boleh dummy untuk uji onboarding lewat seed):
   ```bash
   cp deploy/.env.example .env    # lalu sesuaikan DB_URL dengan Postgres lokalmu
   ```
   Minimal yang penting:
   - `DB_URL` — kredensial Postgres lokal. Jika password mengandung `@` `:` `/`, **encode** (mis. `@` → `%40`).
   - `APP_ENV=dev` — WAJIB dev di lokal (cookie sesi non-`Secure` agar jalan di `http://localhost`, dan seed diizinkan berjalan).
   - `IG_APP_ID/IG_APP_SECRET/IG_VERIFY_TOKEN/IG_REDIRECT_URI` — boleh dummy (tak dipakai kecuali OAuth Instagram nyata).

2. Muat ke shell **(ulangi di tiap terminal baru)**:
   ```bash
   set -a && source .env && set +a
   ```
   - `set -a` = auto-export semua variabel berikutnya ke proses anak (`go run`).
   - `source .env` = eksekusi isi `.env` di shell saat ini.
   - `set +a` = matikan auto-export lagi.

   Cek berhasil: `echo $DB_URL` harus memunculkan nilainya.

---

## 3. Migrasi + Seed (sekali di awal)

```bash
# buat tabel
goose -dir db/migrations postgres "$DB_URL" up

# isi data dev: user demo + akun IG demo (connected, token dummy) + contoh katalog
go run ./apps/api/cmd/seed
```

Seed mencetak kredensial: **`demo@zosmed.test` / `demo12345`**, akun `@olshop.aurora` (status `connected`), onboarding **belum selesai** (siap untuk menguji alur). Keduanya **idempotent** — aman diulang.

Varian: `go run ./apps/api/cmd/seed -complete` → user demo langsung "onboarded" (segment=seller), untuk menguji layar yang mengasumsikan onboarding selesai.

---

## 4. Menjalankan service

Semua opsi mengasumsikan env sudah dimuat (§2) dan Postgres nyala.

### Opsi A — dua terminal (paling jelas)
```bash
# Terminal 1 — API (:8080)
go run ./apps/api/cmd/api

# Terminal 2 — Web (:3000)
cd apps/web && bun run dev
```

### Opsi B — satu terminal, mati bareng saat Ctrl+C
```bash
set -a && source .env && set +a
trap 'kill 0' EXIT
go run ./apps/api/cmd/api &
( cd apps/web && bun run dev ) &
wait
```

### Opsi C — script all-in-one
`scripts/dev.sh` membungkus load env → migrasi → seed → API + Worker + Web, dengan cleanup Ctrl+C:
```bash
bash scripts/dev.sh                 # semua
SKIP_SETUP=1 bash scripts/dev.sh    # lewati migrasi+seed
SKIP_WORKER=1 bash scripts/dev.sh   # tanpa worker (cukup untuk login/onboarding)
```

### Worker (opsional)
Hanya untuk comment-to-order (deteksi keep/C, reservasi, retry). **Tidak** dipakai login/onboarding.
```bash
go run ./apps/worker/cmd/worker
```

---

## 5. URL & Port

| Service | URL |
|---|---|
| Web (Next) | http://localhost:3000 |
| API (chi) | http://localhost:8080 |

Web mem-proxy `/api/*` dan `/connect/*` ke API (`next.config.ts` rewrites) supaya cookie sesi `zsid` first-party — jadi dari browser kamu cukup akses `:3000`.

---

## 6. Reset database (mulai bersih)

```bash
goose -dir db/migrations postgres "$DB_URL" reset   # drop semua
goose -dir db/migrations postgres "$DB_URL" up       # buat lagi
go run ./apps/api/cmd/seed                            # seed ulang
```
Atau cukup hapus baris: `TRUNCATE app_user, user_session CASCADE;` lalu seed lagi.

---

## 7. Troubleshooting

| Gejala | Penyebab / solusi |
|---|---|
| `config: required env var X is not set` | Env belum dimuat — jalankan `set -a && source .env && set +a`. |
| API gagal start, `db: ping` | Postgres mati / `DB_URL` salah / password ber-`@` belum di-encode (`%40`). |
| Login sukses tapi `/auth/me` 401 di browser | `APP_ENV=prod` di lokal → cookie `Secure` tak terkirim via http. Set `APP_ENV=dev`. |
| Web tak bisa hit API | Pastikan API di `:8080` dan `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`) benar. |
| Seed menolak jalan | `APP_ENV=prod` — seed sengaja menolak DB produksi. Set `dev`. |
