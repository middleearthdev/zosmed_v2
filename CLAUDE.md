# CLAUDE.md — Zosmed

> Konteks proyek untuk agen coding (Claude Code / Cowork). Baca file ini **sebelum** menulis kode apa pun.
> Tujuannya: agen paham produk, domain, dan **batasan API Instagram** secara utuh, lalu hanya membangun yang benar-benar feasible.

---

## 1. Ringkasan Produk

**Zosmed** adalah platform otomasi yang **mengubah komentar & DM Instagram jadi hasil**: sebuah **engine percakapan netral** + **visual workflow builder** yang dimulai dari komentar/DM Instagram, mengelolanya jadi percakapan terstruktur, lalu meneruskan ke channel tempat hasil benar-benar terjadi (terutama **WhatsApp**).

**Tesis platform:** _"Comment → Outcome"_ — **ubah komentar jadi hasil**. Apa wujud "hasil" tergantung segmen pengguna (jualan, edukasi, atau jasa).

**Tagline vertikal penjual:** **"COMMENT → CASH."** — tetap dipakai untuk segmen seller (lihat Seller Kit §8.1), bukan tagline seluruh platform.

### Model produk: satu engine netral + beberapa "Kit" per segmen

Zosmed **bukan** chat tool generik (itu head-to-head melawan ManyChat dan kehilangan moat Indonesia), tapi juga **tidak mengunci diri ke penjual saja**. Model paling kuat: **satu engine netral** (workflow §7 + safety §10 + AI persona) dipakai bersama, lalu **Kit spesialis per segmen** di atasnya (§8):

- **Seller Kit** — jualan: keep/C, trust-kit anti-penipuan, commerce calendar. **Moat Indonesia.**
- **Creator Kit** — edukasi/creator: lead-magnet delivery, link-in-DM, waitlist, handoff ke newsletter/komunitas, kode afiliasi.
- **Booking Kit** — jasa lokal: komen → WA/kalender untuk janji temu.

Onboarding cukup bertanya **"kamu jualan, edukasi, atau jasa?"** lalu memuat Kit yang relevan. **Engine, safety layer, dan AI persona dipakai bersama semua Kit** — Kit hanya menambah preset node, template, intent, dan aset.

Zosmed **bukan** sekadar "auto-reply DM". Posisinya: _comment-to-outcome closer_ yang dimulai dari komentar Instagram dan berakhir di hasil nyata (transaksi / lead / booking) — sesuatu yang tidak dilakukan kompetitor global yang berhenti di inbox IG.

**Prinsip desain produk #1:** setiap fitur harus berdiri di atas kemampuan **Instagram Graph API resmi**. Jangan pernah menjanjikan kemampuan yang tidak ada datanya (lihat §4). Kalau sebuah fitur butuh data yang tidak diekspos API, fitur itu tidak dibangun — atau dipindah ke channel lain (mis. TikTok) sebagai integrasi terpisah.

---

## 2. Target Pengguna & Konteks Pasar

**Pengguna:** olshop / UMKM / brand kecil-menengah Indonesia yang berjualan via Instagram (sering merangkap TikTok & WhatsApp), biasanya mengandalkan "admin olshop" untuk balas chat manual.

**Realita pasar yang membentuk produk:**

- **WhatsApp adalah channel closing.** Di IG orang riset/tanya; transaksi diselesaikan lewat chat, mayoritas pindah ke WA. Karena itu **WhatsApp handoff = fitur inti, bukan tambahan.**
- **Live & comment commerce besar.** Banyak penjualan terjadi lewat komentar ("keep", "C1") dan live.
- **Siklus belanja tertanggal:** tanggal **gajian/tanggal muda**, **Harbolnas** (9.9–12.12), **Ramadan/Lebaran/THR**.
- **COD dominan** → sumber retur & order fiktif.
- **Kepercayaan jadi penghambat beli:** "real kak?", "ga tipu2 kan?" → butuh bukti (testimoni, real-pict, resi).
- **Bahasa khas:** informal, slang, typo, code-switch ID/EN, sapaan "kak/sis/gan", budaya **nego**. AI generik terdengar robotik → AI yang fasih gaya olshop adalah moat.

---

## 3. Glosarium Istilah (wajib dipahami agen)

| Istilah                   | Arti                                                                            |
| ------------------------- | ------------------------------------------------------------------------------- |
| **keep / C1 / C3**        | Kode klaim barang yang diketik pelanggan di komentar untuk memesan/menahan item |
| **PO**                    | Pre-order; barang dibuat/dikirim setelah pesanan terkumpul                      |
| **ready**                 | Stok ready, bukan PO                                                            |
| **real-pict**             | Foto produk asli (bukan katalog) sebagai bukti                                  |
| **resi**                  | Nomor pengiriman; bukti barang dikirim                                          |
| **COD**                   | Cash on delivery                                                                |
| **ongkir**                | Ongkos kirim                                                                    |
| **gajian / tanggal muda** | Periode awal bulan saat daya beli naik (~tgl 25–1)                              |
| **Harbolnas**             | Hari Belanja Online Nasional (9.9, 10.10, 11.11, 12.12)                         |
| **nego**                  | Tawar-menawar harga                                                             |
| **olshop**                | Online shop                                                                     |

---

## 4. ⚠️ BATASAN PLATFORM INSTAGRAM — BACA SEBELUM CODING

Bagian ini adalah **sumber kebenaran** soal apa yang boleh dibangun. Semua menargetkan **Instagram API with Instagram Login** (host **`graph.instagram.com`**), akun **Business/Creator** saja, sebagian butuh App Review untuk Advanced Access.

### 4.0. 🔑 API Surface: Instagram Login (`graph.instagram.com`) — BUKAN Facebook Login

> **KEPUTUSAN MENGIKAT.** Zosmed integrasi Instagram **HANYA** lewat **Instagram API with Instagram Login** di host **`graph.instagram.com`**. **JANGAN** memakai **Instagram API with Facebook Login** (`graph.facebook.com`), Facebook Login, Facebook Page token, atau Page-scoped flow apa pun. Semua fitur & spesifikasi (§4a, §5, §7, §8, libs/igapi) **wajib** mengikuti model ini.

- **Login model:** **Business Login for Instagram** — user login langsung dengan akun Instagram profesional (Business/Creator), **tanpa** perlu Facebook Page atau akun Facebook tertaut.
- **Host API:** semua panggilan ke **`https://graph.instagram.com`** (mis. `GET /me`, `GET /{ig-id}/...`, `POST /{ig-id}/messages`, `POST /{ig-comment-id}/replies`). **Tidak ada** panggilan ke `graph.facebook.com`.
- **OAuth / token:**
  - Authorization window: `https://www.instagram.com/oauth/authorize` (scope `instagram_business_*`).
  - Tukar `code` → short-lived token di `https://api.instagram.com/oauth/access_token`.
  - Tukar/ perpanjang ke **long-lived token (≈60 hari)** di `https://graph.instagram.com/access_token`, refresh via `https://graph.instagram.com/refresh_access_token`.
  - Token = **Instagram User access token** (IG-user-scoped), **bukan** Page access token.
- **Scopes (Instagram Login):** `instagram_business_basic`, `instagram_business_manage_comments`, `instagram_business_manage_messages`, `instagram_business_content_publish`, `instagram_business_manage_insights`. (Pakai nama scope `instagram_business_*`, **bukan** scope Facebook lama seperti `instagram_basic`/`pages_*`/`instagram_manage_*`.)
- **Webhook:** berlangganan field (`comments`, `messages`, `mentions`) lewat produk **Instagram** di App Dashboard (bukan produk Messenger/Facebook). Verifikasi payload via **App Secret** (HMAC-SHA256) — sama mekaniknya, beda sumber langganan.
- **Identitas akun:** `id` akun di webhook & API adalah **Instagram-scoped user id** (IGSID untuk DM). Resolusi akun internal harus berbasis id ini, bukan Page id.

> Implikasi kode: `libs/igapi` base URL = `https://graph.instagram.com`. OAuth/connect flow, token store/refresh, dan webhook subscription semua mengikuti Instagram Login. Setiap referensi lama ke `graph.facebook.com`/Facebook Login adalah **bug** dan harus diperbaiki.

### 4a. ✅ Yang DIDUKUNG (boleh dibangun)

- **Webhook komentar** pada post/Reel milik akun → trigger comment-to-DM & comment-to-order.
- **Webhook pesan (DM)** masuk (user yang memulai).
- **Story reply** & **Story mention** (event messaging).
- **Balas komentar publik** (`POST /{comment-id}/replies`) & **sembunyikan/hapus komentar**.
- **Private reply** ke komentar (`POST /{ig-id}/messages`, satu balasan DM per komentar).
- **Kirim DM** dalam window yang berlaku (lihat 4c).
- **Metrik milik sendiri:** insight post/Reel/Story (reach, views, likes, comments, saves), analitik akun, demografi follower (butuh 100+ follower).
- **@mention** via webhook; **business discovery** (data publik akun bisnis lain).
- **Opt-in / one-time notification** sebagai cara "broadcast" yang sah (bukan blast bebas).

### 4b. ❌ Yang TIDAK DIDUKUNG — JANGAN PERNAH DIBANGUN (DO-NOT list)

Agen **tidak boleh** membuat fitur/endpoint/klaim UI berikut. Ini bukan soal effort — datanya memang tidak ada di API.

1. **Trigger "new follower"** — tidak ada webhook follower baru.
2. **Auto-follow / unfollow user** — API tidak bisa follow/unfollow akun lain.
3. **Cek apakah user tertentu mem-follow kita** ("follow status") — tidak ada endpoint.
4. **Jumlah penonton IG Live ("watching count")** — tidak diekspos sama sekali.
5. **Komentar IG Live secara real-time** — bukan permukaan resmi yang andal (Live Shopping ditutup sejak 2023). **keep/C harus berbasis komentar post/Reel, bukan live.** (Versi live sejati = track TikTok terpisah, fase lanjutan.)
6. **Blast DM massal** ke semua follower — dilarang; hanya opt-in/one-time notification.
7. **Scraping** data IG di luar OAuth (akun personal, engagement orang lain) — di luar cakupan & melanggar ToS.

> Jika sebuah permintaan fitur menyentuh daftar ini, agen **berhenti** dan menandai bahwa itu tidak feasible, bukan mengakali dengan workaround tidak resmi.

### 4c. Aturan Messaging & Rate Limit (tegakkan di engine, bukan opsional)

- **Window 24 jam:** DM standar hanya boleh ke user yang berinteraksi dalam 24 jam terakhir; setelah itu user harus re-engage.
- **Satu private reply per komentar**, dikirim dalam **≤7 hari** sejak komentar.
- **Rate limit praktis (default sistem):**
  - Comment replies/jam: cap **750** (batas teknis Meta).
  - DM/jam: cap **200** (aman; sisanya **antre/queue overflow**).
  - DM/hari: cap **1.000** (behaviour-based soft limit).
  - Comments per post / 5 menit: cap **30** (human-paced).
  - AI tokens/hari: cap **1.000.000** (cost guard, soft).
- **Dedupe:** jangan kirim DM ganda ke user yang sama untuk pemicu yang sama.
- **Auto-pause** saat mendekati limit (mis. ≥80% kuota), **cooldown** otomatis, dan **kill switch** manual.
- **DM ke non-follower** bisa masuk folder _message requests_, bukan inbox utama.
- **Produksi butuh App Review Meta** + status partner yang relevan.

---

## 5. Arsitektur Sistem (acuan, boleh disesuaikan)

```
Instagram (graph.instagram.com — Instagram Login + Webhooks)   ← BUKAN graph.facebook.com (§4.0)
        │  events: comments, messages, mentions, story replies
        ▼
[Webhook Ingest]  ──►  [Event Queue]  ──►  [Workflow Engine]
 (verify signature)     (Redis/asynq)       (jalankan node: trigger→filter→action)
                                               │
            ┌──────────────────────────────────┼───────────────────────────┐
            ▼                                   ▼                           ▼
   [Rate-Limit/Safety]               [AI Service (olshop)]         [Outbound Senders]
   (window 24h, 200/hr,              (LLM + persona prompt)        - IG reply/DM (graph.instagram.com)
    dedupe, auto-pause)                                            - WhatsApp handoff (wa.me)
            │                                                       - Webhook keluar
            ▼
       [Postgres]  ← contacts, conversations, workflows, reservations, events, opt-ins
```

**Stack default yang disarankan** (ubah bila tim punya preferensi):

- Frontend: **Next.js + React + TypeScript**, Tailwind. Tema dark + aksen lime (lihat §10).
- Backend: **Go (Golang)** — REST (chi/echo/gin) atau gRPC. Cocok untuk webhook throughput tinggi & concurrency (goroutine) saat traffic komentar melonjak.
- Queue: **Redis + asynq** (task queue Go; penting untuk pacing & queue overflow DM). Alternatif: River (Postgres-backed).
- DB: **Postgres** dengan **sqlc + pgx** (query type-safe, idiomatik Go). Alternatif: GORM bila tim lebih suka ORM. Migrasi via **goose** atau **golang-migrate**.
- Integrasi: **Instagram API with Instagram Login** di host **`graph.instagram.com`** (§4.0 — BUKAN `graph.facebook.com`/Facebook Login), wa.me deep link (MVP), payment gateway = **fase lanjutan** (butuh KYC, jangan di MVP).

### 5a. Struktur Repository — MONOREPO

Seluruh kode (backend Go, frontend Next.js, worker, skema DB, shared lib) berada di **satu repository** (`zosmed_v2/`). Tujuannya: satu sumber kebenaran, atomic change lintas service, dan reuse tipe/kontrak antar bagian.

```
zosmed_v2/
├── apps/
│   ├── api/                 # Go: REST/webhook server (cmd/api/main.go + internal/)
│   ├── worker/              # Go: asynq worker (cmd/worker/main.go + internal/)
│   └── web/                 # Next.js + React + TS frontend
├── packages/                # shared TypeScript (FE/tooling)
│   ├── ui/                  # komponen UI + design tokens §11
│   ├── types/               # tipe kontrak API (selaras dgn backend)
│   ├── config/              # tsconfig/eslint/tailwind preset bersama
│   └── kits/                # UI preset per segmen (seller/creator/booking)
├── libs/                    # shared Go (dipakai api + worker)
│   ├── igapi/               # client Instagram API (Instagram Login, base graph.instagram.com §4.0)
│   ├── safety/              # rate-limit/queue/window layer §10
│   ├── workflow/            # engine NETRAL: trigger→filter→action
│   └── kits/                # preset per segmen di atas engine
│       ├── seller/          #   Seller Kit §8.1
│       ├── creator/         #   Creator Kit §8.2
│       └── booking/         #   Booking Kit §8.3
├── db/
│   ├── migrations/          # goose (up/down)
│   ├── query/               # query.sql untuk sqlc
│   └── sqlc.yaml
├── deploy/                  # Dockerfile, docker-compose, infra
├── go.work                  # Go workspace (apps/api, apps/worker, libs/*)
├── package.json             # workspaces JS via Bun (apps/web, packages/*)
├── turbo.json               # orkestrasi build/lint/test sisi JS
└── CLAUDE.md
```

**Tooling monorepo:**

- **Sisi Go:** `go.work` (Go workspaces) menautkan `apps/api`, `apps/worker`, dan `libs/*` tanpa publish module. Shared Go code masuk `libs/`, bukan di-copy antar app.
- **Sisi JS/TS:** **Bun workspaces** (field `workspaces` di `package.json` root) + **Turborepo** (`bunx turbo run build|lint|test`). Bun sebagai package manager & runtime (`bun install`, `bun run <script>`). Frontend di `apps/web`, kode reusable di `packages/*`.
- **Boundary:** `apps/*` boleh impor dari `libs/*` (Go) atau `packages/*` (TS); `libs/`/`packages/` **tidak** boleh impor balik dari `apps/`. Hindari dependensi melingkar.
- **Engine vs Kit:** `kits/*` boleh impor engine (`workflow`, `safety`, `igapi`); engine **tidak boleh** tahu soal Kit (engine netral segmen). Tambah segmen = tambah modul di `libs/kits/` + `packages/kits/`, tanpa menyentuh engine.
- **Kontrak API:** tipe request/response dijaga selaras antara `apps/api` (Go) dan `packages/types` (TS) — saat mengubah satu sisi, perbarui sisi lain dalam commit yang sama.

---

## 6. Model Domain (entitas inti)

- **Account** — akun IG Business/Creator yang terhubung via **Instagram Login** (§4.0): IG-user-scoped long-lived token + refresh, IG user id (IGSID), status. **Bukan** Page token.
- **Workflow** — graph dari nodes (trigger → filters → actions) + status (draft/live/paused).
- **Node** — unit di workflow (lihat katalog §7).
- **Contact** — pelanggan (IG user id, nama, tag, riwayat, status window 24h).
- **Conversation** — thread DM + state window (open/closed), sumber (comment/DM/story).
- **Reservation** — untuk comment-to-order: code (C1/C3…), produk, status (`reserved`/`waiting-pay`/`closed-wa`/`expired-released`), countdown.
- **OptIn** — daftar user yang opt-in untuk notifikasi (untuk commerce calendar).
- **Event/RunLog** — jejak eksekusi node (untuk Runs & audit).
- **TrustAsset** — pustaka testimoni/real-pict/resi untuk trust-kit.

---

## 7. Workflow Engine — Katalog Node (feasible only)

Palette ini sudah disaring agar 100% sesuai API. Jangan menambah node yang melanggar §4b.

### Triggers

| Node                             | Sumber data        | Catatan                       |
| -------------------------------- | ------------------ | ----------------------------- |
| IG Comment received              | webhook `comments` | komentar di post/Reel         |
| IG DM received                   | webhook `messages` | user memulai duluan           |
| Story reply                      | webhook `messages` | membuka window 24h            |
| Story mention                    | webhook `mentions` | hanya saat di-mention         |
| **Comment-to-order (post/Reel)** | webhook `comments` | deteksi kode keep/C → reserve |
| Click-to-DM ad                   | entry point iklan  | mulai percakapan sah          |

### Filters

| Node                 | Implementasi                                 |
| -------------------- | -------------------------------------------- |
| Keyword match        | logika server (termasuk regex)               |
| Conversation state   | cek apakah dalam window 24h                  |
| Intent: ragu / trust | klasifikasi intent ("real kak?", "ga tipu2") |
| Post selection       | filter `media_id` dari payload               |
| Time window          | logika server                                |

### Actions

| Node                                | Endpoint / mekanisme          | Batasan                                               |
| ----------------------------------- | ----------------------------- | ----------------------------------------------------- |
| Reply comment                       | `POST /{comment-id}/replies`  | rate limit; publik                                    |
| Send DM                             | `POST /{ig-id}/messages`      | window 24h / 1 private reply per komentar (7 hari)    |
| AI reply (olshop)                   | LLM → Send API                | tetap kena window 24h                                 |
| **Kirim link WhatsApp**             | bentuk URL `wa.me/62…?text=…` | **tanpa API**, prefilled `{nama}`/`{produk}`/`{post}` |
| **Kirim trust-kit**                 | kirim aset internal           | dipicu intent "ragu"                                  |
| **Reserve stok (comment-to-order)** | logika internal               | countdown + auto-release                              |
| Notify opt-in                       | one-time notification         | hanya user yang opt-in                                |
| Hand-off to human                   | assign ke admin               | mis. saat refund/komplain                             |
| Tag contact                         | CRM internal                  | bebas                                                 |
| Webhook (keluar)                    | backend pengguna              | bebas                                                 |

---

## 8. Sistem Kit — Engine Netral + Spesialis per Segmen

Engine (workflow §7 + safety §10 + AI persona) bersifat **netral segmen**. Di atasnya berdiri **Kit**: paket berisi preset node, template, intent classifier, aset, dan UI yang dipreset untuk satu segmen. Semua Kit memakai **engine, safety layer, dan AI persona yang sama** — yang berbeda hanya preset & aset. Onboarding memuat Kit sesuai jawaban "jualan, edukasi, atau jasa?".

> **Aturan untuk SEMUA Kit:** sebuah Kit adalah _konfigurasi di atas engine_, **bukan** jalur baru yang mem-bypass guardrail. Tiap Kit tetap (a) tunduk §4 — hanya kemampuan API resmi; (b) seluruh outbound lewat safety layer §10; (c) copy default Bahasa Indonesia. Menambah Kit **tidak boleh** memperkenalkan kemampuan IG yang ada di DO-NOT list §4b.

### 8.1 Seller Kit — jualan (moat Indonesia)

Tagline vertikal: **COMMENT → CASH**. Semua aktif via workflow node, **tanpa integrasi berbayar/dokumen** di MVP.

1. **Handoff ke WhatsApp** — DM IG menyodorkan link `wa.me` berisi konteks terisi (nama, produk, dari post mana). Closing pindah ke WA. _Implementasi: bentuk URL ber-encode. Nol API._

2. **AI olshop persona** — varian persona dari engine AI bersama (dipakai semua Kit): sapaan kak/sis, slang olshop, code-switch ID/EN, sedikit Jawa/Sunda, tanggapi nego, emoji secukupnya, formal=off. _Implementasi: system prompt + few-shot pada engine AI yang ada (lihat §11)._

3. **Trust-kit anti-penipuan** — saat terdeteksi keraguan ("real kak?", "ga tipu2 kan", "ada testi?", "amanah?", "COD bisa?"), auto-kirim aset: testimoni, real-pict, bukti resi. _Implementasi: pustaka aset + trigger intent. Nol API eksternal._

4. **Comment-to-Order (keep/C engine)** — pelanggan komen kode "keep"/"C1" di **post/Reel** → auto private-reply → **reserve stok + countdown** (mis. tahan 5 menit) → **closing via WhatsApp** → auto-release jika tak dibayar. Status reservasi: `reserved` → `waiting-pay` → `closed-wa` / `expired-released`. _Catatan: berbasis komentar post/Reel, BUKAN IG Live (lihat §4b.4–5). Pakai webhook `comments` resmi._

5. **Commerce calendar autopilot** — campaign terjadwal ke momen Indonesia (gajian, Flash sale tengah bulan, Harbolnas 12.12, Ramadan/THR/Lebaran) lewat **notify opt-in**. _Implementasi: scheduler + template + one-time notification._

> Honorable mention (fase lanjutan, butuh API key): **auto cek ongkir** (RajaOngkir/Biteship). Tidak di MVP.

### 8.2 Creator Kit — edukasi/creator

Untuk creator/edukator yang mengubah komentar jadi audiens & lead. Wujud "hasil" = lead/subscriber, bukan transaksi. Semua berdiri di kemampuan resmi yang **sama persis** (comment webhook → private reply/DM berisi link, opt-in):

- **Lead-magnet delivery** — komen kata kunci ("MAU", judul ebook) → auto private-reply + DM link materi/freebie.
- **Link-in-DM** — kirim link (linktree/landing/produk) lewat DM, alih-alih "link in bio".
- **Waitlist / opt-in** — kumpulkan pendaftar lewat opt-in (one-time notification) untuk peluncuran/kelas.
- **Handoff ke newsletter/komunitas** — DM berisi link join newsletter / Telegram / Discord / WA community.
- **Kode afiliasi** — kirim kode/link afiliasi personal sebagai teks/DM.

### 8.3 Booking Kit — jasa lokal

Untuk jasa lokal (klinik, salon, bengkel, fotografer, les, dll): komentar/DM → janji temu. Wujud "hasil" = booking.

- **Comment-to-Booking** — komen kata kunci ("BOOKING", "JADWAL") → auto private-reply + DM.
- **Handoff ke WhatsApp / kalender** — DM berisi link `wa.me` prefilled atau link kalender (Calendly/Google Calendar) untuk pilih slot. Penjadwalan terjadi **di luar IG** (tanpa API IG tambahan, pola sama seperti wa.me).
- **Reminder via opt-in** — pengingat janji lewat one-time notification (hanya user yang opt-in).
- **AI persona jasa** — varian persona bernuansa layanan (tetap ramah) dari engine AI yang sama.

> **Feasibility (§4):** Creator & Booking Kit **tidak** memperkenalkan kemampuan IG baru. Keduanya memakai webhook comment/DM, private reply, DM-with-link, dan opt-in yang sudah **ALLOW** di §4a. Integrasi kalender/newsletter = deep link eksternal, **bukan** API IG. Tetap wajib lewat safety layer §10 (window 24h, rate limit, dedupe).

---

## 9. Daftar Layar & Navigasi

Sidebar (key di kode): `dashboard`, `workflows`, `inbox`, `ai`, `contacts`, `analytics`, `safety`, `templates`, `settings`, `team`, `notifications`.

Layar tambahan: **Landing**, **Onboarding** (mulai dari **pilih segmen** — _jualan / edukasi / jasa_ → memuat Kit terkait §8 — lalu connect IG OAuth), **Workflow Builder** (node canvas), **Comment-to-Order**, **Kit Center** (kelola Kit aktif: Seller/Creator/Booking), **Billing**, **Safety Center**.

Catatan UI: pill "● LIVE" pada Runs/preview berarti **workflow sedang aktif berjalan** (warna lime/status), **bukan** siaran Instagram Live. Jangan menambah elemen apa pun yang menyiratkan data IG Live.

---

## 10. Safety & Rate-Limit Engine (komponen wajib)

Bangun sebagai layer di depan semua outbound sender. Mengacu angka di §4c.

- Penghitung kuota per akun (comment replies/jam, DM/jam, DM/hari, comments/post/5min, AI tokens/hari).
- **Queue overflow**: DM di atas 200/jam → antre, bukan ditolak.
- **Window-aware**: tolak/override aksi DM di luar window 24h; arahkan ke opt-in jika perlu.
- **Dedupe** per (user, trigger).
- **Auto-pause** ≥80% kuota + cooldown; **kill switch** manual global.
- Tampilkan ke pengguna: gauge "200/200 dm·hr", log "auto-pause · rate near limit", milestone ("1.000 auto-replies bulan ini").

---

## 11. Design System / Token (dari design final)

- **Tema:** dark. Background `#0a0a0a`, teks `#f4f4f0`, muted `#a3a39c` / `#66665f` / `#3a3a40`.
- **Aksen lime (brand):** `oklch(0.85 0.16 75)` (ZZ_LIME) — warna utama Zosmed.
- **WhatsApp green:** `oklch(0.82 0.2 145)`.
- **Alert/pink:** `oklch(0.78 0.2 0)`. **Info/blue:** `oklch(0.78 0.16 240)`.
- **Font:** Geist (`var(--font-sans)`), mono untuk angka/teknis (`var(--font-mono)`).
- Gaya: editorial, banyak `mono` untuk label teknis, pill berstatus, kartu rounded. Pertahankan konsistensi dengan artboard Direction A (Dark + Lime).

---

## 12. Konvensi & Guardrails untuk Agen

- **Monorepo** (§5a): backend Go (`apps/api`, `apps/worker`) + frontend Next.js (`apps/web`) + shared lib (`libs/` Go, `packages/` TS) dalam satu repo. Shared code masuk `libs/`/`packages/`, jangan copy-paste antar app. Hormati boundary: `apps/*` impor `libs/*`/`packages/*`, tidak sebaliknya.
- **Engine netral + Kit** (§8): engine (`libs/workflow`, `libs/safety`, AI persona) tidak boleh tahu soal segmen. Fitur per-segmen masuk **Kit** (`libs/kits/<segmen>` + `packages/kits/<segmen>`) sebagai _konfigurasi_ di atas engine — bukan jalur yang mem-bypass §4/§10. Sebelum menaruh logika di engine, tanya: "ini netral untuk semua segmen, atau milik satu Kit?"
- **Go** untuk backend (idiomatik: package layout `cmd/` + `internal/`, error wrapping, context-aware), **TypeScript** strict untuk frontend. Nama domain pakai istilah produk (`Reservation`, `Contact`, `OptIn`).
- Sebelum membangun fitur yang menyentuh Instagram, **cek §4**. Kalau ada di DO-NOT list (§4b) → hentikan & laporkan, jangan workaround.
- **Jangan** integrasikan payment gateway / fitur ber-KYC di MVP. Langkah bayar = handoff WhatsApp.
- **Jangan** menulis kode yang mengasumsikan data IG Live (penonton, komentar live).
- Setiap outbound IG **wajib** lewat Safety/Rate-Limit layer (§10). Tidak ada pengiriman langsung yang mem-bypass pacing.
- Semua copy default berbahasa Indonesia, gaya ramah olshop.
- Tulis kode dengan asumsi **App Review belum selesai** untuk dev: pakai test users / mode standar dulu.

### 12a. Prinsip Coding (wajib, berlaku untuk semua agen)

Tujuan: kode yang mudah dibaca, mudah diubah, dan tidak menumpuk utang teknis. Empat prinsip ini setara pentingnya dengan guardrail platform.

1. **Hindari redundansi (DRY).** Jangan menduplikasi logika, query, tipe, atau konstanta. Kalau pola yang sama muncul ≥2 kali, ekstrak ke satu sumber: helper di `libs/*` (Go) atau util/hook di `packages/*` (TS). Konstanta domain (rate limit §4c, status reservasi, kata kunci Kit) didefinisikan **satu kali** dan diimpor — jangan hardcode ulang di tiap tempat.

2. **Komponen & fungsi reusable.** Sebelum menulis baru, **cari dulu** yang sudah ada (Grep) dan pakai/perluas. UI reusable → `packages/ui`; logic Go reusable → `libs/*`. Komponen kecil, satu tanggung jawab, di-parameterisasi lewat props/argumen — bukan copy-paste lalu diedit sedikit.

3. **Separation of Concerns (SoC).** Pisahkan lapisan dengan tegas, jangan dicampur:
   - **Go:** handler/transport (HTTP, webhook) ↔ business logic (`internal`/`libs`) ↔ data access (sqlc). Handler tidak menulis SQL; business logic tidak tahu detail HTTP; engine tidak tahu detail Kit (§8).
   - **Frontend:** presentational component (tampilan) ↔ state/data-fetching (hook) ↔ tipe kontrak (`packages/types`). Komponen tidak fetch langsung di tengah render JSX.
   - Outbound IG selalu lewat satu pintu (safety layer §10), bukan tersebar di banyak tempat.

4. **Jangan over-abstraction.** Abstraksi mengikuti kebutuhan nyata, bukan antisipasi. **Rule of three**: ekstrak abstraksi setelah pola terbukti berulang, bukan saat pertama kali. Hindari generic/interface/factory/wrapper yang tidak punya ≥2 pemakai konkret. Lebih baik kode eksplisit yang sedikit panjang daripada lapisan indireksi yang menyembunyikan alur. **DRY ≠ alasan untuk abstraksi prematur** — gabungkan hanya yang benar-benar konsep yang sama, bukan yang kebetulan mirip.

> Keseimbangan: prinsip 1 (DRY) dan prinsip 4 (anti over-abstraction) saling menyeimbangkan. Hilangkan duplikasi nyata, tapi jangan memaksakan kesatuan pada hal yang hanya kebetulan serupa. Saat ragu, pilih yang **paling mudah dibaca dan dihapus**.

---

## 13. Roadmap Fase

- **MVP (sekarang):** OAuth IG via **Instagram Login** (`graph.instagram.com`, §4.0), webhook ingest, **engine netral** (workflow + katalog node §7), Safety layer §10, AI persona, **Seller Kit §8.1** (WA handoff, AI olshop, trust-kit, comment-to-order, commerce calendar), onboarding pilih-segmen, layar inti §9. Semua gratis/nol-dokumen.
- **Fase 2:** **Creator Kit §8.2** + **Booking Kit §8.3** (multi-segment di atas engine yang sama), Kit Center, auto cek ongkir (API key), payment link/QRIS (setelah KYC), inbox multi-agen, analytics attribution lanjutan.
- **Fase 3:** track **TikTok Live** (keep/C versi live sejati, integrasi terpisah), reseller funnel, COD risk-score, Kit tambahan per segmen baru.

---

## 14. Definition of Done (untuk fitur apa pun)

Sebuah fitur dianggap selesai jika: (a) berdiri di atas kemampuan API resmi (§4a) dan tidak menyentuh §4b; (b) seluruh outbound melewati Safety layer; (c) menghormati window 24h & 1-reply-per-komentar; (d) copy default Bahasa Indonesia gaya olshop; (e) konsisten dengan design token §11; (f) mematuhi prinsip coding §12a — tanpa duplikasi/redundansi, memakai komponen reusable, SoC terjaga, dan tanpa abstraksi prematur.
