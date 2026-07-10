# Katalog Node Workflow — Zosmed

> Referensi lengkap semua node yang tersedia di Workflow Builder: fungsi, konfigurasi, status, dan batasannya.
> Sumber kebenaran kode: `libs/workflow/nodes/catalog.go` (Go) & `packages/types/src/workflow.ts` (`NODE_CATALOG`, FE). Node spesialis seller di `libs/kits/seller/`.
> Semua node sudah disaring agar **100% feasible** terhadap batasan Instagram API (CLAUDE.md §4). Tidak ada node yang menyentuh DO-NOT list §4b (new-follower, auto-follow, follow-status, IG-Live viewer/komentar live, blast DM massal).

---

## 1. Konsep dasar

Sebuah **workflow** adalah graf berarah: **Trigger → Filter(s) → Action(s)**.

- **Trigger** — pemicu; dari mana event masuk (komentar, DM, story, iklan). Tiap workflow **wajib** minimal 1 trigger.
- **Filter** — syarat lanjut; event diteruskan hanya bila lolos (kata kunci, window 24 jam, post tertentu, jam operasional). Opsional.
- **Action** — hasil; apa yang dikerjakan (balas komentar, kirim DM, link WA, reservasi stok, webhook). Tiap workflow **wajib** minimal 1 action.

Setiap eksekusi menghasilkan **RunLog** (lihat layar *Runs*). Semua action outbound ke Instagram **wajib** lewat Safety/Rate-Limit layer (CLAUDE.md §10) — tidak ada pengiriman langsung.

### Status node: Runnable vs "Segera hadir"

| Status | Arti |
|---|---|
| **Runnable** ✅ | Bisa dieksekusi runtime sekarang (ada factory + jalur ingest/service pendukung). |
| **Segera hadir** 🔒 | Tampil di palette dengan badge, tapi **disabled** untuk drag & aktivasi. Subsystem pendukungnya belum ada. Workflow yang bergantung pada trigger non-runnable **ditolak** saat activate (alasan `trigger_not_runnable`). |

### Engine netral vs Seller Kit

Sebagian besar node **netral segmen** (engine, dipakai semua Kit). Dua node adalah bagian **Seller Kit** (`libs/kits/seller`): `comment-to-order` dan `reserve-stock`. Onboarding memuat Kit sesuai segmen (jualan/edukasi/jasa).

---

## 2. Triggers (pemicu)

| Node (`node_type`) | Label | Fungsi | Sumber event | Status |
|---|---|---|---|---|
| `comment-received` | Komentar IG masuk | Memicu saat ada komentar baru di **post/Reel**. | webhook `comments` | ✅ Runnable |
| `comment-to-order` | Comment-to-Order (keep/C) | Memicu **hanya** saat komentar memuat kode keep/C (mis. `C1`) → jalur reservasi stok. **Seller Kit.** | webhook `comments` | ✅ Runnable |
| `dm-received` | DM IG masuk | Memicu saat **user memulai** DM duluan. Membuka/menyegarkan window 24 jam. | webhook `messages` | ✅ Runnable |
| `story-reply` | Balasan Story | Memicu saat user membalas Story akun. Membuka window 24 jam. | webhook `messages` (subtype) | ✅ Runnable |
| `story-mention` | Mention di Story | Memicu saat akun di-mention di Story orang lain. Membuka window 24 jam. | webhook `mentions` | ✅ Runnable |
| `click-to-dm-ad` | Klik iklan Click-to-DM | Memicu dari entry point iklan Click-to-DM yang membuka percakapan. | ad-referral event | ✅ Runnable |

**Catatan penting:** kedua trigger komentar berbasis **webhook `comments` pada post/Reel — BUKAN IG Live** (§4b.4–5 — Live tidak diekspos API secara andal).

### Konfigurasi trigger

- **`click-to-dm-ad`** — `adRef` (teks, opsional): batasi trigger ke satu iklan Click-to-DM tertentu via kode referral. Kosong = semua klik iklan.
- Trigger lain tidak punya config user-facing.

---

## 3. Filters (syarat lanjut)

| Node (`node_type`) | Label | Fungsi | Status |
|---|---|---|---|
| `keyword-match` | Cocokkan kata kunci | Lanjut hanya jika teks event memuat **salah satu** kata kunci. | ✅ Runnable |
| `conversation-state` | Status percakapan | Cek apakah percakapan masih dalam **window 24 jam**. | ✅ Runnable |
| `post-selection` | Pilih post tertentu | Lanjut hanya jika komentar berasal dari `media_id` tertentu. | ✅ Runnable |
| `time-window` | Jendela waktu | Lanjut hanya pada hari & rentang jam tertentu (jam operasional). | ✅ Runnable |
| `intent` | Intent: ragu / trust | Deteksi keraguan pembeli ("real kak?", "ga tipu2 kan?"). | 🔒 Segera |

### Konfigurasi filter

| Node | Field | Tipe | Default | Keterangan |
|---|---|---|---|---|
| `keyword-match` | `keywords` | list (chip) | `[]` | Daftar kata; lanjut bila teks memuat salah satunya. Kosong = semua event. Case-insensitive (disembunyikan dari UI, default true). |
| `conversation-state` | `requireOpen` | boolean | `true` | `true` = lanjut hanya bila window 24 jam **masih terbuka**. `false` = lanjut justru bila sudah lewat 24 jam. |
| `post-selection` | `mediaIds` | list (chip) | `[]` | ID post/Reel yang diizinkan. Kosong = semua post. |
| `time-window` | `days` | weekdays | `[]` | Hari aktif (`"0"`=Minggu … `"6"`=Sabtu). Kosong = setiap hari. |
| | `startMinute` | time | — | Jam mulai (disimpan sebagai menit sejak 00:00). |
| | `endMinute` | time | — | Jam selesai (menit sejak 00:00). |
| | `timezone` | select | `Asia/Jakarta` | `WIB` / `WITA (Asia/Makassar)` / `WIT (Asia/Jayapura)`. |

---

## 4. Actions (hasil)

| Node (`node_type`) | Label | Fungsi | Endpoint/mekanisme | Status |
|---|---|---|---|---|
| `reply-comment` | Balas komentar publik | Kirim balasan **publik** ke komentar. | `POST /{comment-id}/replies` | ✅ Runnable |
| `send-dm` | Kirim DM | Kirim direct message (kena window 24 jam). | `POST /{ig-id}/messages` | ✅ Runnable |
| `send-whatsapp-link` | Kirim link WhatsApp | Private reply berisi link `wa.me` terisi otomatis. **Nol API IG untuk WA.** | private reply + URL `wa.me` | ✅ Runnable |
| `reserve-stock` | Reserve stok | Tahan stok + countdown untuk comment-to-order. **Seller Kit.** | logika internal | ✅ Runnable |
| `outbound-webhook` | Webhook keluar | Kirim event ke backend eksternal milik pengguna. | HTTP POST (ber-guard SSRF) | ✅ Runnable |
| `ai-reply` | Balasan AI (olshop) | Balasan otomatis pakai persona AI olshop. | LLM → Send API | 🔒 Segera |
| `send-trust-kit` | Kirim trust-kit | Kirim testimoni/real-pict/resi saat terdeteksi keraguan. | aset internal | 🔒 Segera |
| `notify-optin` | Notifikasi opt-in | One-time notification ke user yang sudah opt-in. | one-time notification | 🔒 Segera |
| `handoff-human` | Hand-off ke admin | Alihkan percakapan ke admin manusia. | assign internal | 🔒 Segera |
| `tag-contact` | Tag kontak | Tambah tag ke kontak (CRM internal). | CRM internal | 🔒 Segera |

### Konfigurasi action

| Node | Field | Tipe | Wajib | Keterangan |
|---|---|---|---|---|
| `reply-comment` | `template` | textarea | — | Placeholder `{nama}` (nama pengomen). Kosong = balasan bawaan. |
| `send-dm` | `template` | textarea | — | Placeholder `{nama}`. **Hanya mengirim bila ada percakapan DM/Story dalam window 24 jam.** Dipasang pada workflow yang dipicu komentar → node otomatis **di-skip** (tidak error). |
| `send-whatsapp-link` | `phone` | phone | ✅ | Nomor admin untuk closing. Boleh `0812…` atau `62812…` (dinormalisasi ke E.164). |
| | `template` | textarea | — | Placeholder `{nama}`, `{wa_link}`. |
| `outbound-webhook` | `url` | text | ✅ | URL `https` non-IG. Diguard SSRF di backend. |
| | `includeSignature` | boolean | — | Tanda tangani request (HMAC). |
| | `secret` | text | — | Kunci verifikasi tanda tangan (isi bila HMAC aktif). |
| `reserve-stock` | — | — | — | Config di-bind di startup (service reservasi + nomor WA), bukan per-node. |

---

## 5. Batasan platform yang ditegakkan engine (CLAUDE.md §4c, §10)

Node action tidak "polos" mengirim — Safety layer menegakkan aturan berikut:

- **Window 24 jam** — `send-dm` hanya boleh ke user yang berinteraksi dalam 24 jam terakhir. Di luar itu → di-skip / arahkan opt-in.
- **1 private reply per komentar**, dikirim ≤7 hari sejak komentar.
- **Rate limit default:** comment replies **750/jam**, DM **200/jam** (overflow → **antre**, bukan ditolak — ADR-007), DM **1.000/hari**, comments/post **30 per 5 menit**.
- **Dedupe** per (kind, akun, user, trigger) — tidak kirim ganda untuk pemicu yang sama.
- **Auto-pause** ≥80% kuota + cooldown; **kill switch** manual global.

---

## 6. Aturan validasi saat Activate

Saat `POST /api/v1/workflows/{id}/activate`, workflow ditolak (422 `validation_failed`) bila:

| Alasan | Arti |
|---|---|
| `no_trigger` | Tidak ada trigger. |
| `no_action` | Tidak ada action. |
| `unknown_node_type` | Ada node dengan tipe yang tak dikenal katalog. |
| `trigger_not_runnable` | Satu-satunya trigger masih "segera hadir" (non-runnable). |
| `cycle` | Koneksi antar node membentuk lingkaran. |

---

## 7. Ringkasan node runnable (yang bisa dipakai sekarang)

- **Trigger:** `comment-received`, `comment-to-order`, `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad`
- **Filter:** `keyword-match`, `conversation-state`, `post-selection`, `time-window`
- **Action:** `reply-comment`, `send-dm`, `send-whatsapp-link`, `reserve-stock`, `outbound-webhook`

Node "segera hadir" (`intent`, `ai-reply`, `send-trust-kit`, `notify-optin`, `handoff-human`, `tag-contact`) tampil di palette untuk visibilitas roadmap, tapi belum bisa diaktifkan.

---

> **Cara menguji node-node ini end-to-end:** lihat `docs/manual-test-workflow.md`.
