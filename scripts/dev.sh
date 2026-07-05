#!/usr/bin/env bash
# Zosmed — jalankan seluruh stack dev sekaligus (API + Worker + Web).
#
#   ./scripts/dev.sh              # load .env → migrasi → seed → API+Worker+Web
#   SKIP_SETUP=1 ./scripts/dev.sh # lewati migrasi+seed, langsung nyalakan service
#   SKIP_WORKER=1 ./scripts/dev.sh# tanpa worker (cukup untuk uji login/onboarding)
#
# Ctrl+C mematikan semua proses sekaligus (kill 0 → seluruh process group).
# Prasyarat: Postgres & Redis sudah jalan; goose, go, bun terpasang.

set -euo pipefail
cd "$(dirname "$0")/.."   # pindah ke root repo

# ── 1. Load .env ──────────────────────────────────────────────────────────────
if [[ ! -f .env ]]; then
  echo "✗ .env tidak ditemukan. Salin dari deploy/.env.example lalu isi." >&2
  exit 1
fi
set -a && source .env && set +a
echo "✓ .env dimuat (APP_ENV=${APP_ENV:-dev})"

# Peringatan: '@' / ':' / '/' di dalam password DB_URL harus di-encode (mis. @ -> %40).
if [[ "${DB_URL:-}" =~ ://[^/]*@[^/@]*@ ]]; then
  echo "⚠  DB_URL sepertinya punya '@' di password — encode jadi %40 kalau koneksi gagal." >&2
fi

# ── 2. Dependencies FE (sekali) ───────────────────────────────────────────────
if [[ ! -d node_modules ]]; then
  echo "→ bun install (pertama kali)…"
  bun install
fi

# ── 3. Migrasi + seed (idempotent; lewati dengan SKIP_SETUP=1) ────────────────
if [[ "${SKIP_SETUP:-0}" != "1" ]]; then
  echo "→ goose migrate up…"
  goose -dir db/migrations postgres "$DB_URL" up
  echo "→ seed data dev…"
  go run ./apps/api/cmd/seed
fi

# ── 4. Nyalakan service, cleanup bersama saat keluar ─────────────────────────
trap 'echo; echo "→ menghentikan semua service…"; kill 0' SIGINT SIGTERM EXIT

echo "→ API   http://localhost:${PORT:-8080}"
go run ./apps/api/cmd/api &

if [[ "${SKIP_WORKER:-0}" != "1" ]]; then
  echo "→ Worker (asynq)"
  go run ./apps/worker/cmd/worker &
fi

echo "→ Web   http://localhost:3000"
( cd apps/web && bun run dev ) &

echo
echo "── Semua service jalan. Buka http://localhost:3000 · Ctrl+C untuk stop ──"
wait
