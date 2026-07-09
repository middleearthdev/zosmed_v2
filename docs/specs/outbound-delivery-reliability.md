# ADR-007 — Outbound-Delivery Reliability (retry generik, ordering ingest, idempotensi run)

Status: Implemented (commit menyusul)
Tanggal: 2026-07-08
Penulis: System Architect (Zosmed)
Scope: Menuntaskan **3 temuan sistemik** yang ditunda dari code review ADR-006 (commit 835678d) karena bersifat lintas **safety + ingest**, bukan patch DM-only. (#3) **Retry generik untuk verdict `Queue`** dari safety Gate — saat ini hanya seller-kit `outbound:send` yang punya retry; node aksi generik (`send-dm`, `reply-comment`, `send-whatsapp-link`) menganggap `Queue` sebagai "ditunda" tanpa antre → pesan hilang, melanggar §10 ("overflow → antre, bukan ditolak"). (#4) **Ordering dedupe→enqueue** di webhook ingest — insert ledger sukses lalu enqueue gagal = event tercatat "processed" tapi tak pernah diproses, dan retry Meta di-drop dedupe → hilang permanen. (#6) **Idempotensi retry worker ingest** — asynq re-run task yang error → workflow live dijalankan ulang → outbound/efek-samping ganda.
Referensi: CLAUDE.md §4c (window 24h, 1-private-reply/komentar ≤7 hari, overflow→queue, dedupe, auto-pause), §5 (arsitektur), §8 (engine netral + Kit), §10 (safety one-door), §12a (prinsip coding — reuse asynq, no over-abstraction), §14 (DoD). Penerus ADR-001 (comment-to-order), ADR-004/005 (workflow builder + engine wiring), ADR-006 (messaging/story ingest). **Bukan** kode produksi — dokumen desain untuk dieksekusi engineer.

> **Guardrail §4 (tetap berlaku, tidak ada yang dilonggarkan).** ADR ini **tidak** menambah permukaan API IG apa pun. Semua tetap `graph.instagram.com` (§4.0). Semua outbound tetap lewat **satu pintu Gate** (§10) — termasuk pada **setiap dequeue retry**: retry TIDAK mem-bypass Gate, wajib re-cek window 24h / 7-hari-private-reply / dedupe / kuota / kill-switch saat dieksekusi. Pesan yang **window-nya sudah lewat saat dequeue di-drop, bukan dikirim telat**. Tidak ada item §4b (blast/follower-trigger/IG-Live/scraping). Retry = mengirim ulang **pesan yang sama** yang sudah sah, bukan pesan baru.

---

## 0. Ringkasan Keputusan

Tiga isu berbagi satu akar: **jaminan "tidak hilang, tidak ganda" (at-least-once + idempotensi) belum utuh di sepanjang jalur `webhook → queue → worker → Gate → sender`.** Setiap batas antar-sistem (Postgres ledger ↔ Redis/asynq ↔ Gate Redis ↔ igapi) punya lubang. ADR ini menutup ketiganya dengan **reuse mekanisme yang sudah ada** (asynq `TaskID`/`Retention`/`MaxRetry`, Gate dedupe, ledger `processed_*`) alih-alih membangun subsystem baru (§12a-4).

**Keputusan per isu (ringkas):**

| # | Masalah | Keputusan terpilih | Alasan 1-kalimat |
|---|---------|--------------------|-------------------|
| #3 | `Queue` tanpa retry generik | **Satu task `outbound:send` generik & Kind-aware** yang menyerap jalur seller; node netral meng-enqueue via `EnqueueDeferredFunc` yang diinjeksi (pola sama seperti seller `EnqueueOutboundFunc`), dengan `Deadline`/TTL per pesan; handler re-cek Gate tiap dequeue. | Meng-generalkan retry yang **sudah terbukti** di seller ke semua kind tanpa menambah task/tabel baru, sekaligus melebur duplikasi seller. |
| #4 | Ledger di-insert sebelum enqueue | **Enqueue-first + `asynq.TaskID(event_id)` + `Retention`**; ledger `processed_*` di-insert **setelah** enqueue sukses (konfirmasi/dedupe jangka-panjang), plus read-check di awal. | "Processed" tak pernah tercatat tanpa task durabel; idempotensi enqueue diambil dari fitur asynq, bukan outbox/relay baru. |
| #6 | Re-run task → outbound/reserve ganda | **(a) Tambahkan `Kind` ke dedupe key Gate** (memperbaiki collision multi-aksi + menjamin idempotensi per-kind), **(b) `reserve` seller idempoten** via `UNIQUE(account_id, ig_comment_id)` ON CONFLICT return-existing, **(c) invariant: handler ingest tak boleh return error retryable setelah outbound**. | Gate dedupe sudah membuat outbound idempoten per (kind,trigger); yang bocor adalah collision antar-kind + efek-samping non-outbound (reserve) — keduanya ditutup lokal tanpa menyentuh engine core. |

### Temuan arsitektur kunci (hasil audit kode)

- **[#6 — BUG collision dedupe] Dedupe key Gate TIDAK menyertakan `Kind`.** `libs/safety/dedupe.go` `dedupeKey = "safety:dedupe:{account}:{user}:{trigger}"`. Pada workflow netral `[reply-comment → send-whatsapp-link]` untuk komentar yang sama: `reply-comment` (Kind=`comment-reply`, TriggerKey=commentID) meng-`Allow` dan menandai key; lalu `send-whatsapp-link` (Kind=`private-reply`, TriggerKey=commentID **sama**) melihat key itu → **Reject "dedupe"**. Private reply hilang karena public reply-nya. Ini bukan sekadar isu retry — ini **salah pada eksekusi pertama**. Menambah `Kind` ke key sekaligus memperbaiki collision ini DAN membuat re-run task idempoten per-kind (dasar solusi #6).
- **[#6 — sudah aman sebagian] Engine menelan error aksi; task ingest umumnya TIDAK retry setelah outbound.** `libs/workflow/engine.go` `runWorkflow` menaruh error `Action.Execute` ke `result.Err` dan `return (result, true, nil)` — bukan error non-nil. `Engine.Run` hanya mengembalikan error untuk kesalahan **struktural** (node not found/kind salah). Di `comment_ingest.go`/`dm_ingest.go`, `RunStore.Insert` yang gagal **di-log, return nil** (tak retry). Konsekuensi: satu-satunya error retryable yang bisa muncul **setelah** outbound adalah error struktural pasca-Compile — praktis mustahil (Compile sudah memvalidasi registry). Jadi #6 lebih ke **mengunci invariant ini** + menutup efek-samping non-outbound (reserve), bukan menambal double-send yang sudah dicegah dedupe.
- **[#6 — reserve tidak idempoten] `seller.reserve` bukan outbound → tak tersentuh Gate dedupe.** `reserveAction` memanggil `svc.Reserve` = DecrementStock + CreateReservation (insert baris UUID baru). Re-run task = reservasi kedua + stok ke-decrement dua kali. Harus di-idempotenkan di layer seller (bukan engine).
- **[#3 — window sudah re-dicek di retry] Gate `checkWindow` sudah menegakkan 7-hari (private-reply) & 24h (dm) dari `CommentAt`.** Retry yang telat otomatis di-`Reject` saat dequeue (fail-safe drop). `Deadline` eksplisit di payload dipakai sebagai **TTL seragam lintas kind** (termasuk `comment-reply` yang tak punya window di `window.go`) dan penghenti keras retry — belt-and-suspenders dengan window re-check.
- **[#3 — retry seller sudah ada, tinggal digeneralkan] `outbound:send` + `OutboundSendPayload` + `OutboundSendHandler` sudah interface-based** (`outboundStore`, `waitingPayMarker`, `SenderFactory`). Handler tinggal digeneralkan jadi Kind-aware + Deadline + kirim via 3 metode Sender; kopling reservasi tetap lewat interface yang diinjeksi (opsional saat `ReservationID==""`). **Tidak perlu task/tabel baru.**
- **[#4] `asynq` mendukung `TaskID` (idempotent enqueue) + `Retention` (simpan task selesai agar konflik TaskID terdeteksi pasca-selesai).** Pola `asynq.TaskID(...)` sudah dipakai runner untuk `reservation:expire` & `outbound:send`. Reuse untuk ingest.
- **[catatan] Tabel `outbound_log` (00005) ADA tapi TIDAK ditulis di mana pun** (hanya di models.go generated). Bukan bagian jalur reliability sekarang; audit-write opsional dicatat sebagai non-scope.

### Acceptance Criteria (DoD §14)

1. **[#3]** Node netral `send-dm`, `reply-comment`, `send-whatsapp-link` yang menerima `DecisionQueue` **meng-enqueue `outbound:send`** (bukan sekadar "ditunda") dengan `Deadline` per kind; verifikasi lewat test bahwa pesan terkirim setelah kuota pulih dan **di-drop** bila `Deadline` lewat saat dequeue.
2. **[#3]** `outbound:send` **satu handler generik Kind-aware** menggantikan handler khusus-seller: `Allow`→kirim via metode Sender sesuai Kind (`ReplyToComment`/`SendPrivateReply`/`SendDM`), `Queue`→return error (asynq retry, bounded `MaxRetry` + `Deadline`), `Reject`→drop. Setiap dequeue re-`Gate.Allow` **sebelum** igapi (§10 one-door). Kopling reservasi seller aktif hanya bila `ReservationID != ""`.
3. **[#4]** `processComment`/`processMessaging` **enqueue-first**: enqueue dengan `asynq.TaskID(comment_id|mid)` + `Retention`, perlakukan `ErrDuplicateTask` sebagai sukses, **baru** insert ledger `processed_*`. Test: enqueue-gagal → ledger **tidak** tertulis → retry Meta memproses ulang; enqueue-sukses + ledger-gagal → retry Meta tidak menghasilkan task/proses ganda.
4. **[#6]** Dedupe key Gate menyertakan `Kind`. Test: workflow `[reply-comment → send-whatsapp-link]` mengirim **kedua** outbound (tak saling-dedupe); re-run task yang identik **tidak** mengirim outbound kedua per kind.
5. **[#6]** `seller.Reserve` idempoten per `(account_id, ig_comment_id)`: re-run = reservasi & decrement stok **tidak** berganda; mengembalikan reservasi yang sudah ada.
6. **[semua]** Engine core (`libs/workflow/{engine,node,context,gate,event}.go`) **tidak diubah** (grep diff kosong). `libs/workflow/nodes/*` tetap **tidak** mengimpor `libs/kits/*` maupun `libs/safety`. Solusi generik tidak tahu soal Kit (§8).
7. Semua tetap `graph.instagram.com` (§4.0); tidak ada item §4b; copy default Bahasa Indonesia; token desain §11 (Safety Center menampilkan antrean/deferred bila relevan).

### Non-Scope (ditunda)

- **Two-phase dedupe (claim→commit).** Dedupe ditandai **saat `Allow`** (sebelum kirim). Bila proses crash di jendela sempit **antara `Allow` dan `send` sukses**, retry di-blok dedupe → pesan **hilang** (tidak ganda). Untuk MVP ini diterima: mencegah **double-send/spam** (risiko IG lebih buruk) diprioritaskan; overflow `Queue` (kasus hilang yang **sering**) sudah ditutup #3 karena `Queue` **tidak** menandai dedupe. Pengerasan claim→commit dicatat Fase-2 (§6 R-lanjutan).
- **Outbox table + relay poller** untuk ingest (ditolak, lihat §2.2).
- **Audit-write `outbound_log`** per outbound sukses (tabel ada, belum dipakai) — enhancement observability terpisah.
- **Retry backoff adaptif berbasis sisa kuota** (mis. jadwalkan dequeue tepat saat bucket jam reset). MVP pakai `MaxRetry` + backoff asynq + `Deadline`; peningkatan dicatat Fase-2.
- **Dead-letter UI / replay manual** di Safety Center. Task yang habis retry masuk asynq archive (bawaan); UI-nya menyusul.

---

## 1. Konteks & Masalah

Jalur inti (§5): `webhook ingest → asynq → worker (workflow engine) → safety Gate → senders`. Tiga batas antar-sistem punya lubang reliability yang saling terpisah tapi ditangani bersama karena semuanya menyentuh **safety + ingest**.

### 1.1 Isu #3 — `DecisionQueue` tanpa retry generik

`libs/safety` Gate mengembalikan `Queue` untuk: DM overflow ≥200/jam, auto-pause ≥80% (`AutoPauseThreshold`), DM/hari ≥1000, comment-reply ≥750/jam, komentar/post/5mnt ≥30 (`quota.go`). `decision.go` menjanjikan: *"The message is NOT lost — it will be retried when quota recovers."*

Realita: hanya `seller.privateReplyAction` (`libs/kits/seller/privatereply.go`) yang menepati janji itu — ia memanggil `enqueueOutbound` (`EnqueueOutboundFunc`, diinjeksi runner) yang mengenqueue task `outbound:send`, ditangani `apps/worker/internal/tasks/outbound_send.go` yang re-cek Gate. **Node aksi generik** (`action_send_dm.go`, `action_reply_comment.go`, `action_wa_link.go`) pada `DecisionQueue` hanya:

```go
case workflow.DecisionQueue:
    return workflow.ActionResult{Detail: "...ditunda (belum ada retry generik)"}, nil
```

→ pesan **efektif hilang**. Melanggar §10 ("overflow → antre, bukan ditolak") dan §4c. `outbound:send` yang ada bersifat **khusus seller**: payload membawa `ReservationID`, handler menjaga status reservasi & memanggil `MarkWaitingPay` — tak bisa langsung dipakai node netral.

### 1.2 Isu #4 — ordering dedupe→enqueue di webhook ingest

`apps/api/internal/webhook/handler.go` (`processComment` ~b.178–246; `processMessaging` ~b.298–349) urutannya:

```
resolve account → InsertProcessedComment (ledger, ON CONFLICT DO NOTHING) → filter → EnqueueCommentIngest
```

Ledger `processed_comment`/`processed_message` (00004/00016) dipakai sebagai dedupe: `rows==0` berarti sudah diproses → skip. **Bila insert ledger sukses tapi enqueue gagal** (Redis down/timeout), event tercatat "processed" tetapi **tak pernah masuk queue**. Meta akan retry webhook (karena... sebenarnya kita balas 200 selalu — lihat catatan), namun bila retry datang, `InsertProcessedComment` mengembalikan `rows==0` → **di-drop dedupe** → hilang permanen.

Catatan penting hasil audit: `Receive` selalu membalas **200** dan hanya mem-`log.Error` kegagalan `processComment`. Jadi Meta **tidak** tahu ada kegagalan dan **tidak** menjamin retry untuk event itu. Ini memperparah #4: satu kegagalan enqueue = kehilangan senyap. Perbaikan harus memastikan **"processed" hanya tercatat bila hand-off ke queue sudah durabel**.

### 1.3 Isu #6 — idempotensi retry worker ingest

asynq me-retry task yang me-`return` error non-nil (bounded `MaxRetry`, default 25 + backoff eksponensial). `comment_ingest.go`/`dm_ingest.go` menjalankan workflow live → node aksi mengirim outbound. Kekhawatiran review: bila task gagal **setelah** sebagian outbound terkirim, re-run mengulang seluruh run → outbound ganda.

Audit (lihat §0 temuan) menunjukkan gambaran lebih spesifik:
- Error aksi **tidak** memicu retry (ditelan engine). Error retryable praktis hanya infra **sebelum** outbound (GetAccountByID, LoadLive) → aman.
- **Yang benar-benar bocor:** (a) **collision dedupe antar-kind** membuat workflow multi-aksi salah bahkan pada run pertama; (b) **efek-samping non-outbound `seller.reserve`** yang bukan objek Gate → re-run = reservasi/stok ganda; (c) tidak ada **invariant tertulis** yang mencegah engineer masa depan mengubah `RunStore.Insert`/aksi jadi fatal → membuka pintu double-send.

---

## 2. Keputusan per Isu (+ alternatif yang ditolak)

### 2.1 Isu #3 — Satu task `outbound:send` generik Kind-aware yang menyerap jalur seller

**Keputusan.** Generalkan mekanisme retry seller yang sudah terbukti menjadi **satu jalur retry deferred-outbound netral**:

1. **Kontrak netral baru di `libs/workflow/nodes`** (bukan engine core — paket `nodes` memang lapisan node netral yang sudah diperluas ADR-006):
   ```go
   // libs/workflow/nodes/deferred.go (BARU)
   type DeferredOutbound struct {
       AccountID    string
       Kind         string    // "private-reply" | "dm" | "comment-reply"
       IgUserID     string    // IGSID akun bisnis (identitas pengirim)
       TargetUserID string    // IGSID penerima
       ObjectID     string    // comment_id (reply/private-reply) atau message id (dm) — anchor+dedupe
       Text         string    // teks final yang sudah dirender
       CommentAt    time.Time // untuk re-cek window Gate saat dequeue
       PostID       string    // opsional (counter per-post/5mnt)
       TriggerKey   string
       Deadline     time.Time // TTL: di-drop bila now > Deadline saat dequeue (§4c)
   }
   type EnqueueDeferredFunc func(ctx context.Context, d DeferredOutbound, delay time.Duration) error
   ```
   Diinjeksi ke node lewat `nodes.RegisterFactories(fmap, enqueueDeferred)` (tanda-tangan diperluas; `nil` → perilaku lama "deferred only", menjaga backcompat test).

2. **Node netral pada `DecisionQueue`** membangun `DeferredOutbound` (Kind sesuai node, `Deadline = CommentAt + window`), memanggil `enqueueDeferred`. Contoh `Deadline` per kind:
   - `private-reply` (wa-link/seller): `CommentAt + 7*24h` (`PrivateReplyWindowDays`).
   - `dm` (send-dm): `last_interaction_at + 24h` (`MessagingWindowHours`).
   - `comment-reply`: `CommentAt + DeferredCommentReplyTTL` (konstanta baru, mis. 6 jam — kuota comment-reply pulih dalam menit/jam; window keras tak ada di `window.go`).

3. **Handler `outbound:send` digeneralkan** (bukan diganti — reuse struktur interface yang sudah ada): Kind-aware, enforce `Deadline`, kirim via metode Sender yang benar. Kopling reservasi seller **tetap** lewat interface injeksi (`waitingPayMarker` + guard status via `outboundStore`) tetapi **aktif hanya bila `ReservationID != ""`**. Handler tetap **struktural-netral** (bergantung pada interface kecil, bukan `import seller`); seller-awareness terkurung di wiring `main.go`/`runner.go` (komposisi root yang memang boleh tahu seller).

4. **Migrasi jalur seller → generik.** `seller.privateReplyAction` tetap meng-enqueue pada `Queue`, tapi kini menghasilkan `ptasks.OutboundSendPayload` yang **diperluas** (dengan `Kind="private-reply"` + `ReservationID` terisi + `Deadline`). Setelah handler generik siap: `seller.OutboundRetry` + `seller.EnqueueOutboundFunc` **dihapus**, seller memakai `nodes.EnqueueDeferredFunc` yang sama (payload membawa `ReservationID` sebagai field opsional). Satu task, satu handler, nol duplikasi (§12a-1). Tahap migrasi rinci di §4.

**Alternatif yang ditolak:**
- *Dua task terpisah (netral + seller).* Ditolak: duplikasi handler re-gate-and-send (§12a-1); dua tempat yang harus dijaga sinkron.
- *Gate yang mengenqueue sendiri saat `Queue`.* Ditolak: `libs/safety` tak punya teks pesan yang sudah dirender maupun `Sender`; melanggar SoC (§12a-3) dan menyeret asynq ke lib safety.
- *Menyimpan deferred di tabel Postgres + poller.* Ditolak: asynq **sudah** queue durabel dengan delay/retry/archive; tabel+poller = reinvent (§12a-4). `Deadline` + `MaxRetry` cukup.
- *Tanpa `Deadline`, andalkan window re-check Gate saja.* Ditolak sebagian: `comment-reply` tak punya window di `window.go` → bisa retry sampai `MaxRetry` sia-sia; `Deadline` memberi drop-stale seragam.

### 2.2 Isu #4 — Enqueue-first + `asynq.TaskID` + `Retention`; ledger jadi konfirmasi

**Keputusan.** Balik urutan agar **"processed" tak pernah tercatat tanpa task durabel**, dan pakai idempotensi enqueue bawaan asynq:

Urutan baru `processComment` (analog `processMessaging`):
```
1. resolve account (skip unknown)                              [tak berubah]
2. ExistsProcessedComment(comment_id)? → skip bila true         [read-check; tangkap retry Meta yang telat > Retention]
3. filter catalog/live                                          [tak berubah]
4. EnqueueCommentIngest dengan asynq.TaskID(comment_id) + Retention(24h)
      - ErrDuplicateTask / TaskIDConflict → perlakukan SUKSES (sudah ter-enqueue)
      - error lain (Redis down) → return error (log Error); ledger TIDAK ditulis → retry Meta ulang
5. InsertProcessedComment (ON CONFLICT DO NOTHING)              [durabel record + dedupe jangka panjang]
```

Invariant: langkah 5 hanya tercapai bila 4 sukses. Kehilangan senyap butuh **ketiga**-nya gagal bersamaan (read-check miss **dan** TaskID miss **dan** enqueue gagal tak terdeteksi) — dan bila enqueue gagal kita **tidak** menulis ledger, jadi retry Meta/berikutnya memproses ulang. `Retention(24h)` menjaga deteksi konflik `TaskID` pasca-task-selesai (menutup retry Meta dalam 24 jam); `ExistsProcessed*` menutup retry yang lebih telat.

`asynq.TaskID` mencegah **double-enqueue** untuk `comment_id`/`mid` yang sama → tak ada dua task ingest untuk satu event. Ini juga memperkuat #6 (satu event = satu task).

**Alternatif yang ditolak:**
- *Outbox pattern* (insert ledger + outbox dalam satu tx Postgres, relay poller mengenqueue). Ditolak: menambah tabel + goroutine relay + tuning poll-interval — berat untuk MVP saat asynq `TaskID`+`Retention` sudah memberi idempotensi enqueue (§12a-4). Dicatat sebagai opsi Fase-2 bila butuh jaminan exactly-once lintas-region.
- *Delete ledger saat enqueue gagal* (insert dulu, hapus bila enqueue error). Ditolak: `DELETE` sendiri bisa gagal → tetap bocor; menambah operasi tulis di jalur panas; kalah bersih dibanding enqueue-first.
- *Biarkan urutan, andalkan retry Meta.* Ditolak: `Receive` selalu balas 200; Meta tak dijamin retry event yang gagal di-`processComment`.

### 2.3 Isu #6 — `Kind` di dedupe key + `reserve` idempoten + invariant no-retry-after-outbound

**Keputusan (tiga bagian):**

**(a) Tambahkan `Kind` ke dedupe key Gate.** `libs/safety/dedupe.go`:
```
dedupeKey = "safety:dedupe:{kind}:{account}:{user}:{trigger}"
```
Memperbaiki collision (§0 temuan) DAN menjadikan re-run task idempoten **per-kind**: tiap outbound punya key sendiri, ditandai saat `Allow`, sehingga re-run yang identik di-`Reject` per kind. Perubahan lokal di `libs/safety` (bukan engine core); TTL per-kind (`dedupeTTLFor`) tak berubah.

**(b) `seller.Reserve` idempoten per `(account_id, ig_comment_id)`.** Tambah `UNIQUE(account_id, ig_comment_id)` di `reservation` + ubah `CreateReservation` jadi `ON CONFLICT (account_id, ig_comment_id) DO NOTHING RETURNING ...`; bila konflik, `Reserve` **mengambil reservasi yang ada** (query `GetReservationByComment`) alih-alih membuat baru, dan **tidak** men-decrement stok lagi. Decrement + create dibungkus tx yang sudah ada (`NewPgxTxRunner`, MAJOR-3a) — kini tx menjadi no-op idempoten saat konflik. Ini seller-lokal (reserve = konsep seller), engine tak tersentuh.

**(c) Kunci invariant "no retryable error after outbound".** Dokumentasikan + tegakkan di `comment_ingest.go`/`dm_ingest.go`: setelah suatu workflow `Triggered`, handler **hanya boleh** `return nil` (kegagalan `RunStore.Insert`/aksi = log, bukan error). Error retryable **hanya** untuk kegagalan infra **sebelum** engine run (parse/account/loader). Audit menunjukkan ini sudah nyaris berlaku; tambahkan komentar-guardrail + test regresi agar tak diregres. Efek-samping non-outbound yang mungkin ter-replay (reserve) sudah ditutup (b).

**Alternatif yang ditolak:**
- *Run-id deterministik + `node_execution` ledger (checkpoint per-node) di engine.* Ditolak untuk MVP: mengharuskan `engine.runWorkflow` berkonsultasi ke checkpoint sebelum tiap aksi → **mengubah engine core** (dilarang dibekukan). Kombinasi (a)+(b)+(c) memberi idempotensi yang cukup tanpa menyentuh engine. Checkpoint per-node dicatat sebagai opsi bila kelak ada aksi non-outbound non-idempoten lain (rule-of-three, §12a-4).
- *Two-phase dedupe (claim saat Allow, commit saat send sukses, release saat gagal).* Ditolak untuk MVP (lihat Non-Scope §0): menutup jendela crash-mid-send yang **jarang**, dengan biaya kompleksitas signifikan; MVP memprioritaskan cegah double-send.
- *Pindahkan mark-dedupe ke setelah send.* Ditolak: membuka **double-send** pada race/crash (dua run lolos Gate sebelum salah satu mengirim) — lebih buruk untuk IG (spam) daripada kehilangan langka.

---

## 3. Desain Detail

### 3.1 Perubahan kontrak (runner ↔ nodes ↔ gate)

```
                 apps/worker/internal/runner.New (composition root — boleh tahu seller)
                          │  membangun enqueueDeferred (asynq TaskID + Deadline)
        ┌─────────────────┼──────────────────────────────┐
        ▼                 ▼                               ▼
 nodes.RegisterFactories(fmap, enqueueDeferred)   seller.RegisterFactories(fmap, svc, waPhone, enqueueDeferred)
        │  (netral, Kind ∈ dm/comment-reply/private-reply)   │  (private-reply + ReservationID)
        ▼                                                    ▼
   node.Execute → Gate.Allow                          node.Execute → Gate.Allow
        │  Queue → enqueueDeferred(DeferredOutbound{Deadline,...})
        ▼
   asynq: outbound:send  (TaskID = "outbound:{account}:{kind}:{trigger}", MaxRetry, ProcessIn)
        ▼
   OutboundSendHandler.ProcessTask  (GENERIK, Kind-aware)
        │  1. load account (token) — drop bila !connected
        │  2. Deadline lewat? → drop (log "TTL §4c lewat")
        │  3. ReservationID != "" ? → guard status reserved (seller); else skip
        │  4. Gate.Allow(Kind, ...)  ← SATU PINTU, re-cek window/dedupe/kuota/kill-switch
        │        Allow  → Sender.<ReplyToComment|SendPrivateReply|SendDM> ; bila resv → MarkWaitingPay
        │        Queue  → return error (asynq retry; berhenti saat Deadline/MaxRetry)
        │        Reject → drop (window tutup / dedupe / kill-switch)
        ▼
   igapi (graph.instagram.com §4.0)
```

**`workflow.Gater` / `workflow.OutboundReq` / engine core: TIDAK berubah.** Node tetap memanggil `rc.Gate.Allow` seperti sekarang; yang berubah hanya cabang `DecisionQueue` (memanggil `enqueueDeferred`).

### 3.2 Payload asynq (diubah) — `libs/platform/tasks/types.go`

`OutboundSendPayload` diperluas jadi generik (field lama dipertahankan; `ReservationID` jadi opsional):
```go
type OutboundSendPayload struct {
    AccountID     string `json:"account_id"`
    Kind          string `json:"kind"`             // BARU: private-reply|dm|comment-reply
    IgUserID      string `json:"ig_user_id"`
    ObjectID      string `json:"object_id"`        // BARU (generalisasi CommentID): comment_id | message id
    TargetUserID  string `json:"target_user_id"`
    Text          string `json:"text"`             // (dari ReplyText — teks final)
    PostID        string `json:"post_id"`
    TriggerKey    string `json:"trigger_key"`
    CommentAt     string `json:"comment_at"`       // RFC3339 — re-cek window Gate
    Deadline      string `json:"deadline"`         // BARU: RFC3339 — TTL §4c
    ReservationID string `json:"reservation_id,omitempty"` // OPSIONAL — hanya jalur seller
}
```
Catatan kompat: `CommentID`→`ObjectID` dan `ReplyText`→`Text` adalah rename; karena payload hanya dikonsumsi worker (bukan API publik), aman diubah dalam satu commit. Task lama in-flight saat deploy: `outbound:send` kosong-antrean singkat — jika perlu, pertahankan sementara alias JSON `reply_text`.

### 3.3 Skema / migrasi baru

**`db/migrations/00017_reservation_comment_unique.sql`** (untuk #6b):
```sql
-- +goose Up
-- Idempotensi reserve (ADR-007 #6b): satu reservasi per (akun, komentar).
-- Re-run comment:ingest tidak boleh membuat reservasi/decrement stok ganda.
-- Data lama duplikat (jika ada) harus dibersihkan sebelum migrasi ini.
ALTER TABLE reservation
    ADD CONSTRAINT reservation_account_comment_uq UNIQUE (account_id, ig_comment_id);
-- +goose Down
ALTER TABLE reservation DROP CONSTRAINT reservation_account_comment_uq;
```

**Tidak ada tabel baru untuk #3 & #4** (reuse asynq + ledger `processed_*` yang ada). Ini disengaja (§12a-4).

### 3.4 Query sqlc baru/berubah — `db/query/`

- **`processed_comment.sql`**: `ExistsProcessedComment(ig_comment_id) bool` (read-check langkah 2). `InsertProcessedComment` tetap (dipanggil di langkah 5).
- **`processed_message.sql`**: `ExistsProcessedMessage(ig_message_id) bool`.
- **`reservation.sql`**: `CreateReservation` → `... ON CONFLICT (account_id, ig_comment_id) DO NOTHING RETURNING *`; tambah `GetReservationByComment(account_id, ig_comment_id)` untuk ambil-existing saat konflik. Regenerate `libs/platform/dbgen`.

### 3.5 Enqueue idempoten — `apps/api/internal/enqueue/enqueue.go`

`EnqueueCommentIngest`/`EnqueueDMIngest` menambah opsi:
```go
asynq.NewTask(tasks.TaskCommentIngest, b),
    asynq.TaskID("ingest:comment:"+p.CommentID),   // idempotent enqueue
    asynq.Retention(24*time.Hour),                  // konflik TaskID terdeteksi pasca-selesai
```
dan menerjemahkan `errors.Is(err, asynq.ErrDuplicateTask)` → return `nil` (sukses). Handler webhook lalu tetap menulis ledger.

### 3.6 Handler `outbound:send` generik — `apps/worker/internal/tasks/outbound_send.go`

Perubahan inti:
- `PrivateReplySender` → `outboundSender` dengan 3 metode (`ReplyToComment`, `SendPrivateReply`, `SendDM`) — `igapi.Client` sudah memenuhinya (ia `workflow.Sender`). `SenderFactory` mengembalikan `outboundSender`.
- Tambah cek `Deadline` (drop bila lewat) sebelum Gate.
- Cabang kirim by `p.Kind`.
- Guard reservasi + `MarkWaitingPay` **hanya** bila `p.ReservationID != ""`.
- `waitingPayMarker`/`outboundStore` tetap interface (di-inject; untuk deployment netral bisa nil-guarded).

### 3.7 Node netral — `libs/workflow/nodes/`

- **`deferred.go` (baru)**: `DeferredOutbound`, `EnqueueDeferredFunc`, konstanta `DeferredCommentReplyTTL`.
- **`factories.go`**: `RegisterFactories(fmap, enqueueDeferred EnqueueDeferredFunc)` — factory menutup (closure) enqueue ke dalam instance node.
- **`action_send_dm.go` / `action_reply_comment.go` / `action_wa_link.go`**: simpan `enqueueDeferred`; ganti cabang `DecisionQueue` dari report-only menjadi enqueue (fallback report-only bila `enqueueDeferred==nil`). Hitung `Deadline` per kind.

### 3.8 Wiring — `apps/worker/internal/runner/runner.go` + `cmd/worker/main.go`

- `runner.New` membangun `enqueueDeferred` (satu closure asynq: `TaskID("outbound:"+account+":"+kind+":"+trigger)`, `ProcessIn(delay)`, `MaxRetry`). Diteruskan ke `nodes.RegisterFactories` **dan** `seller.RegisterFactories` (seller kirim payload ber-`ReservationID`).
- Hapus `seller.EnqueueOutboundFunc`/`OutboundRetry` setelah migrasi (§4 tahap C).
- `main.go`: `outboundHandler` `SenderFactory` mengembalikan `outboundSender` (3 metode) — `func(token string) tasks.outboundSender { return igapi.New(token) }`.

### 3.9 Urutan operasi ingest baru (ringkas)

`processComment` & `processMessaging` (§2.2): `resolve → Exists?-skip → filter → enqueue(TaskID+Retention, dup=ok) → InsertProcessed`. Balasan 200 tetap; namun kegagalan **enqueue** kini menghentikan penulisan ledger (event akan diproses ulang pada retry berikutnya). Karena `Receive` tak menyampaikan non-200 ke Meta, ledger-after-enqueue memastikan tidak ada "processed tanpa task".

---

## 4. Rencana Implementasi Bertahap (per agen + file)

Urutan dipilih agar tiap tahap bisa di-merge & diuji independen; engine core tak tersentuh di seluruh tahap.

### Tahap A — Safety dedupe key (isu #6a) → **safety-ratelimit-engineer**
Paling terisolasi, memperbaiki bug korek­tif yang berdiri sendiri.
- `libs/safety/dedupe.go`: sisipkan `Kind` ke `dedupeKey`.
- `libs/safety/gate_test.go`, `libs/safety/compliance_test.go`: update ekspektasi + tambah test collision multi-kind.
- **DoD**: workflow `[reply-comment, send-whatsapp-link]` mengirim dua outbound; dua kind pada trigger sama tak saling-dedupe.

### Tahap B — Skema idempotensi + query (isu #4 & #6b) → **db-schema-engineer**
- `db/migrations/00017_reservation_comment_unique.sql` (+down).
- `db/query/reservation.sql`: `CreateReservation` ON CONFLICT DO NOTHING RETURNING; `GetReservationByComment`.
- `db/query/processed_comment.sql`: `ExistsProcessedComment`.
- `db/query/processed_message.sql`: `ExistsProcessedMessage`.
- Regenerate `libs/platform/dbgen` (sqlc).
- **DoD**: migrasi up/down bersih; sqlc build hijau; query baru ada di dbgen.

### Tahap C — Retry generik + payload + handler + nodes + wiring (isu #3, migrasi seller) → **go-backend-engineer**
- `libs/platform/tasks/types.go`: perluas `OutboundSendPayload` (Kind/ObjectID/Text/Deadline; ReservationID opsional).
- `libs/workflow/nodes/deferred.go` (baru): `DeferredOutbound`, `EnqueueDeferredFunc`, `DeferredCommentReplyTTL`.
- `libs/workflow/nodes/factories.go`: signature `RegisterFactories(fmap, enqueueDeferred)`.
- `libs/workflow/nodes/action_send_dm.go` / `action_reply_comment.go` / `action_wa_link.go`: cabang `DecisionQueue` → enqueue (+ hitung Deadline).
- `apps/worker/internal/tasks/outbound_send.go`: generalkan (Kind-aware, Deadline, 3-metode Sender, reservasi opsional).
- `apps/worker/internal/runner/runner.go`: bangun `enqueueDeferred`; teruskan ke `nodes.RegisterFactories` + `seller.RegisterFactories`; hapus `EnqueueOutboundFunc`/`OutboundRetry` seller setelah alih.
- `libs/kits/seller/privatereply.go` + `kit.go` + `factories.go`: alihkan enqueue ke `nodes.EnqueueDeferredFunc` (payload ber-ReservationID); `reservation.go`: `Reserve` idempoten (pakai query Tahap B).
- `apps/worker/cmd/worker/main.go`: `SenderFactory` → `outboundSender` 3-metode.
- **DoD**: node netral Queue → task terenqueue; handler kirim setelah kuota pulih; drop bila Deadline lewat; seller tetap jalan lewat task yang sama; grep `libs/workflow/nodes` tak mengimpor `libs/kits`/`libs/safety`; engine core diff kosong.

### Tahap D — Ingest ordering (isu #4) → **go-backend-engineer**
- `apps/api/internal/enqueue/enqueue.go`: `TaskID`+`Retention`; tangani `ErrDuplicateTask`.
- `apps/api/internal/webhook/handler.go`: reorder `processComment`/`processMessaging` (Exists-check → filter → enqueue → InsertProcessed).
- `comment_ingest.go`/`dm_ingest.go`: tambah komentar-guardrail invariant #6c (tak return error retryable pasca-outbound) + rapikan bila ada jalur yang melanggar.
- **DoD**: enqueue-gagal → ledger tak tertulis (test); enqueue-sukses lalu re-deliver Meta → tak ada task/proses ganda (TaskID).

### Tahap E — Test lintas lapis → **qa-test-engineer**
Lihat §5. Fokus: skenario reliability end-to-end (Queue→retry→send, TTL-drop, enqueue-fail, re-run idempoten, collision-fix).

---

## 5. Skenario Tes Kunci

**#3 retry generik**
1. `send-dm` kena `Queue` (kuota DM/jam penuh) → `outbound:send` terenqueue dengan Kind=dm; setelah counter bucket reset, dequeue → `Allow` → `SendDM` terpanggil sekali.
2. `reply-comment` kena `Queue` (750/jam) → task terenqueue Kind=comment-reply; dequeue re-`Gate.Allow`.
3. **TTL drop**: `send-whatsapp-link` Queue, `Deadline` (CommentAt+7h dev-shim) lewat saat dequeue → handler **drop**, `SendPrivateReply` **tidak** dipanggil, tak ada retry lanjutan.
4. Dequeue saat masih `Queue` → return error → asynq retry; setelah `MaxRetry`/`Deadline` → berhenti (archive), tak ada kirim.
5. Re-`Gate.Allow` pada dequeue benar-benar dipanggil **sebelum** igapi (assert urutan; §10 one-door).

**#4 ordering ingest**
6. Enqueue dibuat gagal (fake enqueuer error) → `ExistsProcessedComment` tetap false setelahnya (ledger tak ditulis) → panggilan `processComment` kedua (retry) berhasil enqueue + tulis ledger.
7. Enqueue sukses, lalu `InsertProcessedComment` gagal (fake) → re-deliver event yang sama → enqueue kedua mengembalikan `ErrDuplicateTask` (TaskID) → diperlakukan sukses → **tidak** ada task ingest kedua.
8. Event sudah diproses penuh + ledger tertulis → re-deliver → `Exists`-check true → skip tanpa enqueue.

**#6 idempotensi**
9. **Collision fix**: workflow `[reply-comment → send-whatsapp-link]` satu komentar → **dua** outbound terkirim (bukan satu ter-dedupe).
10. Re-run `comment:ingest` identik (simulasi asynq retry) untuk workflow single-action → outbound **tidak** terkirim kedua (Gate dedupe per-kind Reject).
11. **Reserve idempoten**: re-run `comment:ingest` keep-code → hanya **satu** reservasi; stok ter-decrement **sekali**; `Reserve` kedua mengembalikan reservasi yang sama.
12. Regresi invariant: `RunStore.Insert` gagal → task `comment:ingest` tetap `return nil` (tak retry).

**Compliance (regresi §4c/§10)**
13. Retry tak pernah mem-bypass Gate: kill-switch di-engage antara enqueue dan dequeue → dequeue → `Reject` → drop.
14. Window: private-reply deferred yang `CommentAt`-nya jadi >7 hari saat dequeue → Gate `Reject` → drop (bukan kirim telat).

---

## 6. Non-Scope (diulang untuk kejelasan) & Catatan Fase-2

- **Two-phase dedupe (claim→commit)** untuk menutup jendela crash-mid-send (R-lanjutan). Saat itu, mark-dedupe dipecah: pending-claim TTL pendek saat `Allow`, promote saat send sukses, release saat gagal — memungkinkan retry send-failure tanpa risiko double-send.
- **Checkpoint per-node (`node_execution` ledger + run-id deterministik)** — hanya bila muncul aksi non-outbound non-idempoten kedua (rule-of-three). Untuk sekarang cukup reserve-idempotent (b).
- **Outbox + relay** untuk ingest — bila butuh exactly-once lintas-region.
- **Audit-write `outbound_log`** per outbound sukses (observability Safety Center).
- **Dead-letter/replay UI** untuk task ter-archive.
- **Backoff adaptif berbasis reset bucket kuota** (jadwalkan dequeue tepat saat jam bucket berganti) — MVP pakai `MaxRetry`+backoff asynq+`Deadline`.

---

## 7. Definition of Done (checklist §14)

- [x] Node netral `Queue` → enqueue `outbound:send`; terkirim setelah kuota pulih; drop saat `Deadline` lewat (#3).
- [x] Satu handler `outbound:send` Kind-aware; seller memakai task yang sama; `EnqueueOutboundFunc`/`OutboundRetry` seller dihapus (#3 migrasi).
- [x] Ingest enqueue-first + `TaskID`+`Retention`; ledger setelah enqueue; `ErrDuplicateTask`=sukses (#4).
- [x] Dedupe key Gate memuat `Kind`; `reserve` idempoten `(account, comment)`; invariant no-retry-after-outbound terkunci (#6).
- [x] Engine core (`libs/workflow/{engine,node,context,gate,event}.go`) diff kosong; `libs/workflow/nodes/*` tak impor `libs/kits`/`libs/safety`; solusi generik netral segmen (§8).
- [x] Setiap dequeue retry re-`Gate.Allow` sebelum igapi (§10); window/dedupe/kuota/kill-switch ditegakkan ulang (§4c).
- [x] Hanya `graph.instagram.com` (§4.0); tak ada §4b; copy BI; token §11.
- [x] Skenario §5 hijau (unit + integrasi).
