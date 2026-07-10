# Skenario Tes Manual — Workflow Builder & Engine (ADR-004 / 005 / 006 / 007)

> Cara menguji workflow end-to-end: bikin workflow di builder → aktifkan → kirim webhook simulasi → cek Runs & hasil outbound.
> Katalog node & fungsinya: `docs/workflow-nodes.md`. Setup lengkap: `docs/how-to-run.md`.

## Prasyarat

| Kebutuhan | Untuk |
|---|---|
| **PostgreSQL** + `createdb zosmed` + migrasi + **seed** | Data akun/katalog IG dummy |
| **Redis nyala** | Webhook ingest = enqueue-first (ADR-007) — **wajib**, kalau mati event tidak diproses |
| **Worker jalan** (`apps/worker`) | Mengeksekusi task `comment:ingest` / `dm:ingest` / `outbound:send` / `reservation:*` |
| **API jalan** (`apps/api`) | Menerima webhook + REST builder |
| **Web jalan** (opsional) | Tes builder via UI di `http://localhost:3000` |
| **Postman collection** `deploy/zosmed.postman_collection.json` | Semua request sudah siap pakai nilai seed |

> Urutan start: `Redis` + `Postgres` → migrasi → seed → `worker` → `api` (+ `web`). Lihat `scripts/dev.sh`.

## Kenapa bisa tes tanpa Instagram sungguhan

- **Seed** menyediakan akun IG dummy: `entry.id = SEED-IG-0001`, `media.id = SEED-MEDIA-0001`, keep code `C1`/`C2`.
- **Signature HMAC** webhook dihitung otomatis oleh pre-request script Postman dari `igAppSecret` — samakan dengan `IG_APP_SECRET` di `.env`.
- Outbound IG **tidak** benar-benar memanggil `graph.instagram.com` di mode dev/test — cek hasilnya lewat **Runs** (RunLog) dan state DB (reservasi), bukan dari inbox IG asli.

---

## Alur uji tercepat (happy path)

Di Postman: **`Login`** → folder *Workflows (Builder)*: **`Create Workflow`** → **`Save Workflow (graph)`** → **`Activate Workflow`** → folder *Webhooks (Meta)*: **`Webhook Receive (comments)`** → **`List Runs (per akun)`**.

---

## Skenario A — Builder & lifecycle (via Postman *Workflows (Builder)* atau UI)

| # | Aksi | Ekspektasi |
|---|---|---|
| A1 | `Login` | 200, cookie `zsid` HttpOnly tersimpan |
| A2 | `Create Workflow` (name + segment) | 201, `{id, status:"draft"}` |
| A3 | `Save Workflow (graph)` — kirim nodes+edges | 200; server hapus+insert transaksional (replace penuh) |
| A4 | `Get Workflow (graph)` | 200; graf yang barusan disimpan kembali utuh |
| A5 | `Activate Workflow` (graf valid: ≥1 trigger runnable + ≥1 action) | 200, `status:"live"` |
| A6 | `Pause Workflow` | 200, `status:"paused"` |
| A7 | `List Workflows` | 200; workflow tampil dengan `nodeCount` & `status` benar |
| A8 | `Delete Workflow` | 200, `{deleted:true}` |

**Via UI (`/workflows`):** drag node dari palette → sambungkan edge → isi inspector → **Simpan** → **Aktifkan**. Node "segera hadir" (badge) tidak bisa di-drag/aktifkan.

---

## Skenario B — Validasi Activate (harus ditolak 422)

Susun graf cacat lalu `Activate Workflow`. Ekspektasi **422 `validation_failed`** dengan `reason`:

| # | Graf | `reason` |
|---|---|---|
| B1 | Tanpa trigger | `no_trigger` |
| B2 | Tanpa action | `no_action` |
| B3 | Ada node tipe asing | `unknown_node_type` |
| B4 | Trigger satu-satunya = node "segera hadir" (mis. `intent` sebagai satu-satunya jalur) | `trigger_not_runnable` |
| B5 | Koneksi membentuk lingkaran | `cycle` |

Copy Bahasa Indonesia per alasan muncul di UI (lihat `VALIDATION_FAILURE_MESSAGES`).

---

## Skenario C — Trigger komentar → action (end-to-end)

**Setup workflow (live):** `comment-received` → `keyword-match` (`keywords: ["keep"]`) → `reply-comment` (`template: "Halo kak {nama}, cek DM ya 🙏"`).

| # | Aksi | Ekspektasi |
|---|---|---|
| C1 | `Webhook Receive (comments)` — teks memuat "keep" | 200 dari webhook; worker jalankan run → **1 RunLog `success`** di `List Runs` |
| C2 | Cek langkah run | trigger `comment-received` ✓ → filter `keyword-match` ✓ → action `reply-comment` ✓ (`{nama}` tersubstitusi handle pengomen) |
| C3 | Webhook komentar **tanpa** kata "keep" | 200; run **`skipped`** di filter (tidak ada balasan terkirim) |
| C4 | Kirim ulang payload dengan `comment_id` **sama** | Event **di-skip** (dedupe) — tidak ada run kedua |
| C5 | Redis mati saat kirim webhook | Webhook tetap balas 200, tapi ledger **tidak** ditulis → kirim ulang payload sama setelah Redis nyala → **diproses** (event tidak hilang, ADR-007) |

**Variasi filter:**
- `post-selection` (`mediaIds: ["SEED-MEDIA-0001"]`) → lanjut; `mediaIds: ["lain"]` → `skipped`.
- `time-window` di luar jam aktif → `skipped`.

---

## Skenario D — WhatsApp handoff (Seller Kit inti)

**Setup:** `comment-received` → `send-whatsapp-link` (`phone: "081234567890"`, `template: "Halo kak {nama}! Lanjut WA ya: {wa_link}"`).

| # | Aksi | Ekspektasi |
|---|---|---|
| D1 | `Webhook Receive (comments)` | Run `success`; private reply berisi link `wa.me/6281234567890?text=…` (nomor dinormalisasi `0…`→`62…`), `{nama}`/`{wa_link}` tersubstitusi |
| D2 | Cek link | Format `https://wa.me/62…?text=<encoded>` — **tanpa** panggil API WhatsApp |

---

## Skenario E — Comment-to-Order (keep/C, Seller Kit)

**Setup:** `comment-to-order` → `reserve-stock`. (Atau pakai layar **Comment-to-Order** langsung.)

| # | Aksi | Ekspektasi |
|---|---|---|
| E1 | `Webhook Receive (comments)` dengan kode `C1` | Run `success`; **reservasi dibuat** (`status: reserved`), stok berkurang, countdown mulai |
| E2 | `Get Comment-Order (agregat layar)` | Komentar tampil `matchedCode:"C1"`, `reserved:true` |
| E3 | Kirim komentar `C1` **lagi dari komentar yang sama** (`comment_id` sama) | **Idempoten** — tidak ada reservasi ganda, stok **tidak** dikurangi dua kali (migrasi 00017 UNIQUE, ADR-007 #6) |
| E4 | Komentar dengan kode **tak dikenal** (mis. `C99`) | Trigger tidak lanjut (bukan keep code valid) → tidak ada reservasi |
| E5 | Biarkan countdown habis tanpa bayar | Task `reservation:expire` → `status: expired-released`, stok dikembalikan |
| E6 | `Close Reservation (waiting-pay → closed-wa)` | 200, `status: closed-wa` |

---

## Skenario F — Trigger DM / Story & window 24 jam

**Setup:** `dm-received` → `send-dm` (`template: "Halo kak {nama}, ada yang bisa dibantu?"`).

| # | Aksi | Ekspektasi |
|---|---|---|
| F1 | `Webhook Receive (DM masuk)` | Run `success`; window 24 jam **terbuka**; `send-dm` terkirim |
| F2 | `Webhook Receive (story reply)` / `(story mention)` | Run `success`; window 24 jam terbuka |
| F3 | `Webhook Receive (click-to-DM ad referral)` | Run `success` (trigger `click-to-dm-ad`) |
| F4 | Pasang `send-dm` pada workflow yang dipicu **komentar** (bukan DM/Story) | `send-dm` **di-skip** (tidak ada `last_interaction_at` / window) — run tetap jalan, DM tidak terkirim (ADR-006 R4) |
| F5 | Filter `conversation-state` (`requireOpen:true`) saat window sudah lewat 24 jam | `skipped` |

---

## Skenario G — Safety & Rate-Limit (ADR-007, CLAUDE.md §10)

| # | Aksi | Ekspektasi |
|---|---|---|
| G1 | Kirim banyak DM melebihi **200/jam** | Kelebihan **di-antre** (task `outbound:send`), bukan ditolak; terkirim setelah kuota reset. Re-cek gate tiap dequeue |
| G2 | Task outbound melewati **deadline** (TTL §4c: private-reply 7×24h, DM 24h, comment-reply 6h) | **Di-drop** (tidak kirim telat), bukan retry selamanya |
| G3 | Dua trigger berbeda-kind pada komentar sama (mis. `reply-comment` + `send-whatsapp-link`) | **Keduanya terkirim** — dedupe key memuat Kind, tidak saling menutup (fix collision ADR-007 #6) |
| G4 | Aktifkan **kill switch** global | Semua outbound berhenti; task di-drop dengan alasan kill-switch |
| G5 | Kuota mencapai ≥80% | **Auto-pause** + cooldown; gauge di Safety Center menunjukkan status |

---

## Reset ke kondisi awal (ulangi tes)

```bash
# hapus workflow/run/reservasi buatan tes, seed ulang (akun/katalog idempotent)
# lihat scripts/ untuk skrip reset yang tersedia; atau drop+recreate db lalu migrasi+seed
```
Untuk event "baru" tanpa reset: **ganti `comment_id`/`mid`** di payload webhook (kalau sama → kena dedupe, di-skip).

---

## Ringkasan ekspektasi

| Area | Inti yang diverifikasi |
|---|---|
| **Builder** (A) | CRUD + activate/pause; replace-penuh transaksional; node "segera" disabled |
| **Validasi** (B) | Activate menolak graf cacat dengan `reason` tepat |
| **Comment → action** (C) | Trigger→filter→action jalan; `skipped` saat filter gagal; **dedupe**; **enqueue-first** (event tak hilang) |
| **WA handoff** (D) | Link `wa.me` terbentuk benar, nomor ternormalisasi, nol API WA |
| **Comment-to-Order** (E) | Reservasi keep/C; **idempoten** per komentar; auto-release; close-wa |
| **DM/Story + window** (F) | Window 24 jam terbuka oleh DM/story; `send-dm` skip di flow comment |
| **Safety** (G) | Overflow **antre** bukan tolak; deadline drop; dedupe per-kind; kill switch; auto-pause |

> Semua outbound IG lewat Safety layer (§10). Semua trigger berbasis webhook resmi `comments`/`messages`/`mentions` — **tidak ada** data IG Live atau kapabilitas DO-NOT list §4b.
