# ADR-006 — Messaging / Story Ingest Pipeline

Status: Proposed (Conditional PASS ig-platform-guardian, koreksi B0 diterapkan 2026-07-08)
Tanggal: 2026-07-08
Penulis: System Architect (Zosmed)
Scope: Membangun **jalur ingest kedua** (setelah comment ingest ADR-001/ADR-005) untuk event **DM, balasan Story, mention Story, dan ad-referral (click-to-DM)**, plus **window/last-interaction store** (§4c 24 jam). Ini membuka 6 node yang `Runnable:false` di katalog: trigger `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad`; filter `conversation-state`; action `send-dm`. **Tidak** menulis ulang engine (`libs/workflow` core), **tidak** menyentuh `libs/safety` core (window 24h sudah ada di `window.go`).
Referensi: CLAUDE.md §4.0 (Instagram Login only), §4a/§4b/§4c, §7 (katalog node), §8 (engine netral + Kit), §10 (safety one-door), §12a (prinsip coding), §14 (DoD). Penerus langsung ADR-005 (§1 klasifikasi B) — bangun di atas ADR-001..ADR-005.

> **Koreksi B0 (ig-platform-guardian, Conditional PASS).** Semua 6 node feasible & §4b-clean, dengan koreksi payload-shape berikut sudah dibakukan ke ADR ini: (1) **story-mention adalah event messaging** (`entry[].messaging[].message.attachments[].type=="story_mention"`), **bukan** `changes[].mentions`; (2) field webhook `changes[].mentions` adalah kapabilitas berbeda (@mention di komentar/caption, §4a) dan **di-drop** dari ADR ini; (3) langganan cukup field **`messages`** (produk Instagram) untuk DM + story-reply + story-mention; (4) semua event messaging **`Source=dm`** — `SourceStory` **tidak** diproduksi pipeline ini; (5) story-mention **membuka window 24h** seperti story-reply (R3 diperbaiki); (6) ad-referral diparse dari **`message.referral`** (thread baru) **atau** **top-level `referral`** (`messaging_referral`, thread lama).

---

## 0. Ringkasan Keputusan

ADR-005 sudah men-decouple comment ingest sehingga workflow generik comment-triggered reachable, dan menaikkan 4 node netral (`post-selection`, `time-window`, `reply-comment`, `outbound-webhook`) jadi runnable. Yang masih terkunci: **semua trigger non-comment** dan **`send-dm`** — karena **belum ada jalur ingest untuk webhook `messages`** dan **belum ada window/last-interaction store** yang jadi syarat DM 24 jam (§4c). ADR ini menutup keduanya.

Enam node yang dibuka (flip `Runnable:false → true`):

```
  TRIGGERS (semua Source=dm)          FILTER                    ACTION
  dm-received      subtype=dm          conversation-state        send-dm  (GATED, Kind=dm)
  story-reply      subtype=story-reply (window 24h)
  story-mention    subtype=story-mention
  click-to-dm-ad   subtype=ad-referral
```

Empat keputusan inti:

1. **Jalur ingest kedua, bukan cabang di comment ingest.** Task baru `dm:ingest` (mirror `comment_ingest.go`) memproduksi `workflow.Event` **tanpa ketergantungan `catalog_post`** (comment ingest coupled ke catalog untuk seller enrichment; DM/story tidak). Webhook `payload.go` diperluas memparse **satu** bentuk baru: `entry[].messaging[]` (field **`messages`**) — mencakup DM, story reply, **story mention**, dan ad-referral. **Identitas akun & kontak = IGSID** (`entry.id` untuk akun, `sender.id` untuk kontak), bukan Page id (§4.0). Field `changes[].mentions` **tidak** diparse di ADR ini (kapabilitas berbeda, lihat koreksi B0 & §3).

2. **Subtype event di `Event.Raw`, bukan mengubah struct `Event`.** Engine core dibekukan (guardrail ADR-004/005). Empat trigger dibedakan lewat satu key `Event.Raw[event_subtype]` ∈ {`dm`, `story-reply`, `story-mention`, `ad-referral`}. **`Event.Source` = `SourceDM` untuk SEMUA event messaging** (DM, story-reply, story-mention, ad-referral semua tiba lewat `entry[].messaging[]`) — `SourceStory` tidak diproduksi pipeline ini; subtype-lah yang membedakan. Pemetaan subtype→trigger bersifat 1:1 (tidak tumpang tindih).

3. **Window store = tabel `conversation` di Postgres (bukan Redis).** Setiap event messaging masuk meng-**upsert** `conversation.last_interaction_at` (window 24h dibuka/di-refresh oleh interaksi user — termasuk story-mention, R3). `send-dm` mengisi `OutboundReq.CommentAt` dari timestamp ini; `libs/safety/window.go` **yang sudah ada** menegakkan 24h (`Kind=dm`). Filter `conversation-state` membaca window yang sama. Rasionalisasi Redis-vs-Postgres di §4.2.

4. **`send-dm` bermakna hanya di flow DM/story.** Pada flow **comment** tidak ada window messaging terbuka (komentar **tidak** membuka window 24h, §4c) → `dm:ingest` tak dijalankan, `comment_ingest` tak mengisi `Raw[last_interaction_at]` → `send-dm` **skip lokal** ("tidak ada window 24 jam terbuka") sebelum menyentuh Sender. Menampilkan `send-dm` di flow comment = node yang selalu skip; ini didokumentasikan, bukan bug (R4).

### Temuan arsitektur kunci

- **Window 24h SUDAH terpasang di safety, tinggal diberi data.** `libs/safety/window.go` `checkWindow` untuk `Kind=dm` sudah: `CommentAt.IsZero() → allow (caller bertanggung jawab tracking); else deadline = CommentAt + 24h`. Artinya **tidak perlu mengubah `libs/safety`** — cukup pastikan `send-dm` **selalu** menyuplai `CommentAt` non-zero dari conversation store. Konsekuensi penting: karena `IsZero() → allow`, `send-dm` **tidak boleh** mengandalkan gate untuk menolak "tidak ada window" — ia harus **guard presence** last-interaction sendiri (skip bila absen), lalu gate menegakkan **freshness** 24h. Split bersih: node = "ada window?", gate = "window masih < 24h?".
- **`workflow.Sender.SendDM` & `igapi.SendDM` SUDAH ada.** Tidak ada permukaan API IG baru; `POST /{ig-user-id}/messages` recipient `{id}` sudah dipetakan ke `graph.instagram.com` (§4.0). Node `send-dm` cukup memanggil `rc.Gate.Allow(Kind=dm)` lalu `rc.Sender.SendDM`.
- **`HasLiveWorkflow(accountID)` sudah ada** (ADR-005 B1) — dipakai ulang sebagai syarat enqueue `dm:ingest`, tanpa query baru.

### Acceptance Criteria (DoD §14)

1. Webhook `POST /webhooks/meta` memparse field **`messages`** (`entry[].messaging[]`) — DM, story-reply, story-mention (via `message.attachments[].type=="story_mention"`), ad-referral. **Tidak** memparse `changes[].mentions` (di-drop, koreksi B0). Meresolusi akun via `GetAccountByIgUserID(entry.id)` (IGSID §4.0), dedupe via `processed_message` (PK = `message.mid`), dan **enqueue `dm:ingest`** bila akun punya ≥1 workflow `live` (`HasLiveWorkflow`). Comment ingest **tidak berubah**.
2. Task `dm:ingest` memproduksi `workflow.Event{Source: dm, Raw[event_subtype], Raw[last_interaction_at], ...}` **tanpa** lookup `catalog_post`, lalu load→compile→`NewEngine`→`Run`→`RunStore.Insert` (pola identik `comment_ingest.go`, first-triggered-wins).
3. Enam node `Runnable=true` di `libs/workflow/nodes/catalog.go` dan tereksekusi engine. Mirror `packages/types` `NODE_CATALOG.runnable` disinkronkan dalam commit yang sama.
4. Setiap event messaging masuk meng-upsert `conversation.last_interaction_at` (window 24h — termasuk story-mention). `conversation-state` membaca window; `send-dm` memakainya sebagai `CommentAt`.
5. `send-dm` memanggil `rc.Gate.Allow` dengan `Kind=dm` (cap DM 200/jam→Queue, 1000/hari; window 24h; dedupe per message id) **sebelum** `rc.Sender.SendDM` (§10 one-door). Bila `Raw[last_interaction_at]` absen/zero → **skip** tanpa menyentuh Sender.
6. `send-dm` pada workflow **comment-triggered** selalu skip ("tidak ada window 24 jam terbuka") — didokumentasikan di help copy node.
7. Tidak ada item §4b: `send-dm` = 1:1 window-gated, **bukan** blast (§4b.6); `click-to-dm-ad` = ad-referral entry (percakapan sah), bukan auto-DM ke stranger; tidak ada follower-trigger/auto-follow/IG-Live/scraping.
8. Hanya `graph.instagram.com` (§4.0). Engine core (`libs/workflow/{engine,node,context,gate,event}.go`) & `libs/safety` core **tidak diubah** (grep diff kosong). `libs/workflow/nodes/*` tak mengimpor `libs/kits/*`.
9. Copy default Bahasa Indonesia gaya olshop; token desain §11; pill "● LIVE" = workflow aktif, bukan IG Live (§9).

### Non-Scope (ditunda)

- **`intent` (ragu/trust) AI**, `ai-reply`, `send-trust-kit`, `notify-optin`, `handoff-human`, `tag-contact` — tetap ditunda (ADR-005 §1B; butuh AI service / asset store / opt-in scheduler / inbox / contacts). ADR ini **hanya** 6 node messaging.
- **@mention di komentar/caption (`changes[].mentions`, §4a "@mention via webhook").** Kapabilitas berbeda dari story-mention (permukaan comment-level, payload `{media_id, comment_id}`). Di-drop dari ADR ini; bila kelak dibangun, ingest & dedupe-nya berbasis `comment_id` (bukan `mid`) dan lebih dekat ke comment pipeline. Dicatat sebagai kandidat ADR terpisah (R8).
- **Cross-source window pada flow comment** (commenter yang juga punya window DM terbuka): `comment_ingest` **tidak** diubah untuk membaca conversation store. `conversation-state`/`send-dm` di flow comment melihat "tertutup" (perilaku benar untuk MVP; enhancement dicatat R5).
- **Broadcast / one-time-notification** (untuk re-engage di luar 24h) = subsystem opt-in, ADR terpisah.
- **Inbox multi-agen** yang memakai `conversation` sebagai thread store — tabel `conversation` di sini sengaja minimal (window tracking); kolom thread/inbox menyusul Fase 2.

---

## 1. Feasibility Audit §4b + Klasifikasi Scope

Audit §4b dijalankan lebih dulu — **tidak ada** dari 6 node yang menyentuh DO-NOT list. Konfirmasi per node (diverifikasi ig-platform-guardian, Conditional PASS):

```
  node              kelas   §4a dukungan                                       §4b?
  ──────────────────────────────────────────────────────────────────────────────────
  dm-received        A*     webhook `messages` (user memulai)                  AMAN
  story-reply        A*     webhook `messages` + message.reply_to.story        AMAN
  story-mention      A*     webhook `messages` + attachments[].story_mention   AMAN
  click-to-dm-ad     A*     ad-referral di `messages` (message/top-level ref)  AMAN (bukan blast/auto-DM)
  conversation-state A*     logika server (baca window store)                  AMAN
  send-dm            A*     POST /{ig-id}/messages recipient {id}              AMAN (1:1 window-gated, BUKAN §4b.6)
  ──────────────────────────────────────────────────────────────────────────────────
  (C) tidak feasible / §4b: TIDAK ADA.
```

`A*` = feasible di level node **setelah** enabler ADR ini (ingest messaging + window store) ada. Verifikasi §4b eksplisit:

- **§4b.1 (new-follower trigger):** tidak dibangun. Tidak ada node "follower baru".
- **§4b.6 (blast DM massal):** `send-dm` **satu pesan ke satu user** yang punya window 24h terbuka, di-dedupe per message id, di-cap oleh gate (200/jam, 1000/hari). Tidak ada mekanisme fan-out ke daftar follower. Re-engage di luar window = opt-in (di luar scope, §4c).
- **`click-to-dm-ad`:** entry point iklan Click-to-DM — **user yang klik iklan lalu mengirim DM**. Ini permulaan percakapan yang sah (§7 "Click-to-DM ad"), bukan sistem yang memulai DM ke orang asing. Feasible.
- **Story reply/mention:** event messaging resmi (§4a "Story reply & Story mention"). Tidak menyentuh IG Live (§4b.4–5).

### Boundary engine vs Kit (§8) — keputusan penempatan

Keenam node **netral segmen** → semua di **`libs/workflow/nodes`** (bukan Kit). Alasan: DM/story/mention adalah primitif percakapan yang dipakai **semua** segmen (Seller handoff, Creator lead-magnet DM, Booking reminder). Tidak ada istilah keep/produk/booking di sini. Kit nanti hanya mengonfigurasi node ini (template DM, keyword) — tanpa menambah node baru. Menambah segmen = tambah Kit di atas node netral ini, engine tak berubah (§8).

---

## 2. Kontrak Node (6 node) — config, Build, guardrail

Semua netral di `libs/workflow/nodes`; registrasi via `nodes.RegisterFactories(fmap)` yang sudah ada (cukup tambah entri). Konvensi `Event.Raw` key (literal, duplikasi wire-key yang diterima §12a-4, sama pola `rawKeyIgUserID`/`rawKeyCommentAt` di `action_wa_link.go`):

```
event_subtype        string   "dm" | "story-reply" | "story-mention" | "ad-referral"
last_interaction_at  time.Time  waktu interaksi user terakhir (window 24h); absen di flow comment
ig_user_id           string   IGSID akun pengirim (sudah ada, dipakai SendDM sebagai sender)
ad_ref               string   (opsional) payload referral iklan untuk click-to-dm-ad
```

| node_type | kategori | config (field → schema FE) | Factory.Build validasi | guardrail kunci |
| --- | --- | --- | --- | --- |
| `dm-received` | trigger | `{}` (tanpa config) | selalu OK | Match: `Source==dm && Raw[event_subtype]=="dm"` |
| `story-reply` | trigger | `{}` | selalu OK | Match: `Source==dm && Raw[event_subtype]=="story-reply"` |
| `story-mention` | trigger | `{}` | selalu OK | Match: `Source==dm && Raw[event_subtype]=="story-mention"` |
| `click-to-dm-ad` | trigger | `{ adRef?: string }` (opsional filter ref iklan) | trim; kosong = match semua ad-referral | Match: `Source==dm && Raw[event_subtype]=="ad-referral"` (dan `adRef` cocok bila diisi) |
| `conversation-state` | filter | `{ requireOpen: bool }` default `true` | boolean | logika server murni; baca `Raw[last_interaction_at]`; **tanpa outbound** |
| `send-dm` | action | `{ template: string }` placeholder `{nama}` | template opsional (default BI) | **GATED** `Kind=dm`; guard presence window; §4.0 |

Catatan: keempat trigger sekarang berbagi `Source==dm` (semua tiba lewat `entry[].messaging[]`); pembeda tunggal adalah `Raw[event_subtype]`. Ini menjaga `Match` tetap tipis dan seragam.

### 2.1 Trigger `dm-received` / `story-reply` / `story-mention` / `click-to-dm-ad`

Empat trigger tipis, seluruh perbedaan = pengecekan `Event.Raw[event_subtype]` (semua `Source==dm`). Contoh (pola sama `commentReceivedTrigger`):

```go
func (t *dmReceivedTrigger) Match(_ context.Context, e workflow.Event) bool {
    return e.Source == workflow.SourceDM && rawString(e.Raw, rawKeyEventSubtype) == subtypeDM
}
// story-reply / story-mention identik, hanya beda konstanta subtype:
//   subtypeStoryReply   = "story-reply"
//   subtypeStoryMention = "story-mention"
//   subtypeAdReferral   = "ad-referral"
```

`click-to-dm-ad` menambah filter opsional `adRef` (bila diisi, bandingkan dengan `Raw[ad_ref]`). Tidak ada outbound di trigger. **§4b:** tidak satu pun trigger ini menyentuh follower/auto-follow/IG-Live — semua dari webhook `messages` resmi.

### 2.2 Filter `conversation-state` (netral, tanpa outbound)

```
config: { requireOpen: bool }   // default true
Allow(rc):
  last := rawTime(rc.Event.Raw, rawKeyLastInteractionAt)
  open := !last.IsZero() && time.Since(last) < 24h    // MessagingWindowHours (mirror konstanta safety)
  return open == requireOpen, nil
```

Konstanta 24h di-mirror sebagai literal lokal (pola sama `replyCommentKind`) — jangan impor `libs/safety` dari node netral. Filter **hanya bermakna** saat `Raw[last_interaction_at]` terisi (flow DM/story). Pada flow comment key absen → `open=false` → filter fail bila `requireOpen=true` (benar). Tanpa outbound → tanpa gate.

### 2.3 Action `send-dm` (netral) — GATED, window-guarded

```
config: { template: string }         // placeholder {nama}; default BI olshop
Execute(rc):
  1. last := rawTime(rc.Event.Raw, rawKeyLastInteractionAt)
     if last.IsZero() → return ActionResult{Detail:"skip: tidak ada window 24 jam terbuka"}, nil  // GUARD sebelum Sender
  2. req := OutboundReq{
        AccountID: rc.Event.AccountID, Kind: "dm",           // == safety.KindDM
        TargetUserID: rc.Event.FromID, TriggerKey: rc.Event.ObjectID,
        CommentAt: last,                                       // ← gate menegakkan freshness 24h dari sini
        PostID: "" }
  3. d := rc.Gate.Allow(ctx, req)                              // ← ONE-DOOR sebelum Sender
  4. Allow  → rc.Sender.SendDM(ctx, Raw[ig_user_id], rc.Event.FromID, text)
     Queue  → dilaporkan ditunda (belum ada retry generik — sama seperti send-whatsapp-link/reply-comment)
     Reject → skipped, dilaporkan (mis. "messaging window 24j lewat", dedupe, kill-switch)
```

Guardrail (reviewer menolak PR yang melanggar):
- **Guard presence** (`last.IsZero()`) **wajib** sebelum gate — karena `safety.checkWindow` meng-`allow` `CommentAt` zero untuk `Kind=dm` (caller bertanggung jawab tracking). Tanpa guard ini, `send-dm` di flow comment akan lolos gate secara keliru.
- **§10 one-door:** `rc.Gate.Allow(Kind="dm")` **sebelum** `rc.Sender.SendDM`. `Kind="dm"` = cap DM (`quota.go` `KindDM`), **bukan** comment-reply.
- **Dedupe** `(account, user, message-id)` ditangani gate (`dedupeTTLDM=24h`) → tidak ada DM ganda untuk pemicu yang sama (§4c). Bukan blast (§4b.6).
- `send-dm` **tidak** set `PostID` (bukan comment-reply; per-post/5min tak relevan untuk DM).

`send-dm` tidak menambah permukaan API baru — `rc.Sender.SendDM` sudah dipetakan ke `POST /{ig-user-id}/messages` di `libs/igapi` (§4.0).

---

## 3. Webhook Parsing (`apps/api/internal/webhook`)

> **Verifikasi ig-platform-guardian: PASS bersyarat.** Yang dikonfirmasi CORRECT (biarkan apa adanya): DM standar (`sender.id`/`recipient.id`/`message.mid`/`message.text`), story reply (`message.reply_to.story{id,url}`), ad-referral thread-baru (`message.referral{ref,ad_id,...}`), identitas IGSID via `entry.id`/`sender.id`, `graph.instagram.com` only. Yang DIPERBAIKI (sudah dibakukan di bawah): story-mention = attachment messaging (bukan `changes[].mentions`); ad-referral thread-lama = top-level `referral` (`messaging_referral`); langganan cukup field **`messages`**. `postback.referral` adalah konstruk Messenger/Facebook — **bukan** Instagram Login; parse hanya sebagai opsional/toleran, jangan diandalkan.

### 3.1 Bentuk payload: `entry[].messaging[]` (field `messages`)

Semua event yang dibutuhkan ADR ini tiba di **`entry[].messaging[]`**. Field `changes[]` **tetap** hanya diparse untuk `comments` (existing, tak diubah). `payload.go` diperluas:

```
MetaEntry:
  ID        string          // IGSID akun (sudah ada)
  Time      int64           // (sudah ada)
  Changes   []MetaChange    // comments (existing, TAK diubah)
  Messaging []MetaMessaging // BARU — field `messages`: DM / story-reply / story-mention / ad-referral

MetaMessaging:
  Sender    { ID string }   // IGSID kontak (from)
  Recipient { ID string }   // IGSID akun (== entry.id)
  Timestamp int64           // ms epoch — waktu interaksi (untuk last_interaction_at)
  Message   *struct {
      Mid         string
      Text        string
      ReplyTo     *struct { Story *struct { ID, URL string } }             // story-reply
      Attachments []struct { Type string; Payload struct { URL string } }  // story-mention: Type=="story_mention"
      Referral    *struct { Ref, AdID, Source, Type string }               // ad-referral (thread BARU: pesan pertama)
  }
  Referral  *struct { Ref, AdID, Source, Type string }                     // ad-referral TOP-LEVEL (messaging_referral, thread LAMA)
  Postback  *struct { Referral *struct { Ref, AdID string } }              // opsional/toleran — konstruk FB, JANGAN diandalkan
```

Tidak ada penambahan pada `changes[]` (field `mentions` di-drop, lihat §0 Non-Scope / R8).

### 3.2 `ExtractMessagingEvents` → `IngestMessaging`

Fungsi baru (mirror `ExtractComments`), menghasilkan slice `IngestMessaging` ternormalisasi dengan **subtype sudah diklasifikasi**. Urutan klasifikasi penting (story-mention & story-reply diperiksa sebelum plain DM):

```
Untuk tiap entry.messaging[] (m):
  source  := "dm"          // SELALU dm untuk permukaan messaging (koreksi B0)
  subtype := "dm"
  adRef   := ""

  if m.Message punya attachment dgn Type=="story_mention"   → subtype = "story-mention"
  else if m.Message.ReplyTo.Story != nil                    → subtype = "story-reply"
  else if m.Message.Referral != nil || m.Referral != nil    → subtype = "ad-referral"
        adRef = firstNonEmpty(m.Message.Referral.Ref, m.Referral.Ref)   // + AdID bila ada
  // (m.Postback.Referral hanya fallback toleran, JANGAN jadi jalur utama)

  emit IngestMessaging{
     EntryID: entry.ID, EntryTime: entry.Time,
     ContactID: m.Sender.ID, MessageID: m.Message.Mid,   // Mid → dedupe processed_message
     Text: m.Message.Text, Subtype: subtype, Source: source,
     AdRef: adRef, EventAt: m.Timestamp,                 // ms epoch → RFC3339 di payload
  }
```

`changes[]` **tidak** menghasilkan `IngestMessaging` (tak ada loop `mentions`). Story-mention **membawa `mid`**, jadi dedupe `processed_message.ig_message_id` PK tetap valid untuk semua subtype. Story-mention **tidak** punya `media_id` di permukaan messaging → `MediaID` kosong (URL story ada di attachment payload bila kelak dibutuhkan; MVP tak memakainya).

### 3.3 `Receive` — cabang messaging (tambahan, comment path tak berubah)

Di `handler.go` `Receive`, setelah loop comment tambahkan loop messaging. `processMessaging(ctx, im)`:

```
1. account := GetAccountByIgUserID(im.EntryID)   // IGSID → akun (§4.0); unknown → skip (never 500)
2. dedupe: InsertProcessedMessage(im.MessageID, accountID, ...) ON CONFLICT DO NOTHING; 0 rows → skip
3. enqueue-gate: HasLiveWorkflow(accountID) == false → skip (tak ada workflow yang mungkin fire)
   (catatan: TIDAK ada pengecekan catalog di sini — DM/story tidak coupled ke catalog)
4. EnqueueDMIngest(DMIngestPayload{ AccountID, Source, Subtype, MessageID, MediaID,
                                    FromID: im.ContactID, FromUsername (bila ada), Text,
                                    AdRef, EventAt: RFC3339 })
5. respon 200 ASAP (identik comment path)
```

`FromUsername` sering **tidak** tersedia di payload messaging (hanya `sender.id`). Keputusan: kirim `FromUsername` kosong; template `{nama}` fallback ke handle kosong / "kak" (copy default olshop menoleransi). Resolusi username via `GET /{igsid}` = enhancement (butuh scope/izin; verifikasi guardian) — **di luar scope**, dicatat R6.

**Guardrail:** webhook tetap hanya verifikasi→dedupe→enqueue (SoC §12a-3). Tidak memanggil igapi, tidak menyentuh conversation store (itu tugas worker — konsisten dengan comment path yang menaruh enrichment di worker).

---

## 4. `dm_ingest` Task + Window Store

### 4.1 Task `dm:ingest` (`apps/worker/internal/tasks/dm_ingest.go`)

Mirror struktur `comment_ingest.go`, **tanpa** blok seller enrichment/catalog:

```
ProcessTask(ctx, t):
  1. unmarshal DMIngestPayload
  2. accountID := uuidx.Parse(p.AccountID)
  3. account := DB.GetAccountByID(accountID); skip bila != "connected"   // pola sama comment_ingest
  4. WINDOW: p.Source SELALU "dm" untuk permukaan messaging (DM/story-reply/story-mention/ad-referral),
     dan semua membuka/refresh window 24h (R3) →
        eventAt := parse(p.EventAt) (fallback now)
        DB.UpsertConversationInteraction(accountID, p.FromID, eventAt, last_source="dm")   // §4.2
        lastInteraction = eventAt
  5. raw := {
        rawKeyIgUserID:          account.IgUserID,   // sender untuk SendDM
        rawKeyEventSubtype:      p.Subtype,
        rawKeyLastInteractionAt: lastInteraction,    // ← window; selalu terisi untuk messaging
        rawKeyAdRef:             p.AdRef (bila ada),
     }
  6. event := workflow.Event{ Source: workflow.SourceDM, AccountID: p.AccountID, ObjectID: p.MessageID,
                              MediaID: p.MediaID, FromID: p.FromID, FromUsername: p.FromUsername,
                              Text: p.Text, Raw: raw }
  7. sender := igapi.New(account.AccessToken)
  8. loaded := Loader.LoadLive(accountID)
     for each: Compiler.Compile → NewEngine([def]) → eng.Run(event, sender, Gate)
               first Triggered → RunStore.Insert(trigger_source="dm", trigger_summary="DM by @…") → return
     (tanpa fallback built-in seller — fallback itu khusus comment-to-order; DM path tak punya legacy)
```

Registrasi handler di `apps/worker/cmd/worker/main.go`: `mux.HandleFunc(ptasks.TaskDMIngest, dmHandler.ProcessTask)`. `libs/platform/tasks/types.go` tambah `TaskDMIngest = "dm:ingest"` + `DMIngestPayload`. `apps/api/internal/enqueue` tambah `EnqueueDMIngest`.

Karena semua event messaging kini `Source=dm`, cabang window (langkah 4) berlaku seragam — upsert dilakukan untuk setiap event ingest. Ini menutup story-mention dengan benar (R3) tanpa cabang khusus.

### 4.2 Window store — keputusan: tabel `conversation` (Postgres), bukan Redis

**Keputusan (R2): Postgres.** Rasionalisasi:

| Kriteria | Postgres `conversation` | Redis |
| --- | --- | --- |
| Durabilitas | window 24h = state percakapan yang **tidak boleh hilang** saat restart/flush; salah = DM ditolak/diluluskan keliru (compliance §4c) | ephemeral; cocok untuk counter yang memang reset per window (quota), bukan untuk state |
| Domain entity | `Conversation` sudah entitas §6, dipakai Inbox Fase 2 — tabel ini jadi fondasinya | bukan model domain |
| Preseden | `processed_comment` = ledger DB yang ditulis di ingest (pola identik) | quota/dedupe counter (beda tujuan) |
| Biaya | 1 upsert + 1 read per event DM, ter-index `(account_id, contact_ig_user_id)` — murah untuk volume MVP | lebih cepat, tapi tak sepadan dengan risiko kehilangan state |

Redis tetap dipakai untuk quota/dedupe (`libs/safety`) — itu benar karena counter memang bucketed & fana. Window last-interaction adalah **state**, jadi ke Postgres. Bila kelak throughput tinggi, cache read di Redis bisa ditambah sebagai optimasi (tiket menyusul) — write tetap ke Postgres sebagai source of truth.

### 4.3 Aliran window → gate

```
DM/story masuk (user) ──► webhook ──► dm:ingest
                                   │ UpsertConversationInteraction(now)   [buka/refresh window]
                                   │ Raw[last_interaction_at] = now
                                   ▼
                          Engine.Run ── send-dm.Execute
                                   │ guard: last_interaction ada? (ya)
                                   │ OutboundReq.CommentAt = last_interaction
                                   ▼
                          rc.Gate.Allow(Kind=dm)
                                   │ safety.checkWindow: now - CommentAt < 24h?  ✔
                                   │ quota DM 200/jam, 1000/hari; dedupe (msg id)
                                   ▼
                          rc.Sender.SendDM  →  POST /{ig-id}/messages (graph.instagram.com)

Comment masuk ──► comment:ingest (TAK diubah) ── Raw[last_interaction_at] ABSEN
                                   ▼
                          send-dm.Execute ── guard: last_interaction absen → SKIP  (benar, R4)
```

---

## 5. Perubahan DB (goose + sqlc)

Nomor migrasi berikutnya: **`00015`**, **`00016`** (existing sampai `00014`). Gaya mengikuti `00003`/`00004` (enum eksplisit bila perlu, index hot-path, catatan sinkron `packages/types`).

### 5.1 `00015_conversation.sql`

```sql
-- +goose Up
-- Window/last-interaction store (§4c 24h). Satu baris per (akun, kontak).
-- last_interaction_at di-refresh setiap event MESSAGING masuk (DM/story-reply/
-- story-mention/ad-referral) — semuanya membuka window messaging (§4c, ADR-006 R3).
-- Komentar TIDAK membuka window messaging → tidak menyentuh tabel ini.
CREATE TABLE conversation (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          uuid        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    contact_ig_user_id  text        NOT NULL,                 -- IGSID kontak (§4.0)
    last_interaction_at timestamptz NOT NULL,                 -- sumber window 24h
    last_source         text        NOT NULL DEFAULT 'dm',    -- 'dm' (semua permukaan messaging)
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, contact_ig_user_id)
);
CREATE INDEX conversation_account_contact_idx ON conversation(account_id, contact_ig_user_id);

-- +goose Down
DROP TABLE conversation;
```

### 5.2 `00016_processed_message.sql`

```sql
-- +goose Up
-- Dedupe ledger event messaging (mirror processed_comment 00004): Meta retry
-- non-200 → cegah enqueue ganda. PK = ig message id (message.mid); setiap subtype
-- (DM/story-reply/story-mention/ad-referral) membawa mid, jadi satu PK cukup.
CREATE TABLE processed_message (
    ig_message_id      text        PRIMARY KEY,
    account_id         uuid        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    subtype            text        NOT NULL DEFAULT 'dm',
    contact_ig_user_id text        NOT NULL DEFAULT '',
    received_at        timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE processed_message;
```

### 5.3 `db/query/conversation.sql`

```sql
-- name: UpsertConversationInteraction :one
-- Buka/refresh window. GREATEST menjaga monoton bila event tiba out-of-order.
INSERT INTO conversation (account_id, contact_ig_user_id, last_interaction_at, last_source)
VALUES (@account_id, @contact_ig_user_id, @last_interaction_at, @last_source)
ON CONFLICT (account_id, contact_ig_user_id) DO UPDATE SET
    last_interaction_at = GREATEST(conversation.last_interaction_at, EXCLUDED.last_interaction_at),
    last_source         = EXCLUDED.last_source,
    updated_at          = now()
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversation WHERE account_id = @account_id AND contact_ig_user_id = @contact_ig_user_id;
```

### 5.4 `db/query/message.sql`

```sql
-- name: InsertProcessedMessage :execrows
-- 0 rows → sudah pernah diproses (dedupe). Pola sama InsertProcessedComment.
INSERT INTO processed_message (ig_message_id, account_id, subtype, contact_ig_user_id)
VALUES (@ig_message_id, @account_id, @subtype, @contact_ig_user_id)
ON CONFLICT (ig_message_id) DO NOTHING;
```

`sqlc generate` setelah migrasi. Tidak ada perubahan pada tabel `workflow*` — jalur runtime memakai loader/compiler/runstore yang sudah ada.

---

## 6. Pembagian Kerja

Path absolut dari root `/Users/fahminurcahya/Documents/Project/zosmed/zosmed_v2`. **Kontrak lintas-agen** (sepakati di awal, ubah dua sisi dalam satu commit §5a): `packages/types/src/workflow.ts` `NODE_CATALOG` ↔ `libs/workflow/nodes/catalog.go` — `runnable=true` untuk 6 node + nama field config (`send-dm.template`, `conversation-state.requireOpen`, `click-to-dm-ad.adRef`) harus **identik** dengan struct Go.

### 6.1 go-backend-engineer

- **B0. Verifikasi ig-platform-guardian — SELESAI (Conditional PASS).** Koreksi payload sudah dibakukan ke ADR ini: langganan **hanya field `messages`** (produk Instagram) untuk DM + story-reply + story-mention; story-mention = `message.attachments[].type=="story_mention"` (bukan `changes[].mentions`); ad-referral dari `message.referral` **atau** top-level `referral`; `postback.referral` opsional-toleran saja. Implementasi WAJIB mengikuti §3.1/§3.2 hasil koreksi ini.
- **B1. Migrasi DB.** `db/migrations/00015_conversation.sql`, `00016_processed_message.sql` (§5.1/§5.2). `goose up`.
- **B2. Query sqlc.** `db/query/conversation.sql` (`UpsertConversationInteraction`, `GetConversation`), `db/query/message.sql` (`InsertProcessedMessage`). `sqlc generate`.
- **B3. Task types + enqueue.** `libs/platform/tasks/types.go`: `TaskDMIngest` + `DMIngestPayload{AccountID,Source,Subtype,MessageID,MediaID,FromID,FromUsername,Text,AdRef,EventAt}`. `apps/api/internal/enqueue/enqueue.go`: `EnqueueDMIngest`.
- **B4. Webhook parsing.** `apps/api/internal/webhook/payload.go`: `MetaEntry.Messaging []MetaMessaging` dengan `Message.{Mid,Text,ReplyTo.Story,Attachments[],Referral}` + `MetaMessaging.Referral` top-level (§3.1); `ExtractMessagingEvents(p) []IngestMessaging` dgn klasifikasi subtype dari attachment/reply_to/referral (§3.2). **JANGAN** parse `changes[].mentions`. `handler.go`: `processMessaging` (resolve akun IGSID → `InsertProcessedMessage` dedupe → `HasLiveWorkflow` gate → `EnqueueDMIngest`); panggil dari `Receive` (comment path tak diubah). Toleran terhadap field absen.
- **B5. Node netral (6 node).** `libs/workflow/nodes/`: `trigger_dm.go` (dm-received), `trigger_story.go` (story-reply, story-mention — keduanya `Source==dm`, beda subtype), `trigger_click_to_dm_ad.go`, `filter_conversation_state.go`, `action_send_dm.go`. Tambah Raw-key konstanta (`rawKeyEventSubtype`, `rawKeyLastInteractionAt`, `rawKeyAdRef`) + subtype konstanta. `action_send_dm.go` **wajib** guard presence + pola gate `Kind="dm"` meniru `sendWhatsAppLinkAction` (Allow/Queue/Reject). `catalog.go`: `Runnable=true` untuk 6 node. `factories.go`: daftarkan 6 factory.
- **B6. dm_ingest handler.** `apps/worker/internal/tasks/dm_ingest.go` (§4.1) — mirror comment_ingest **tanpa** catalog/seller enrichment; `Source=workflow.SourceDM`; upsert conversation untuk setiap event; isi `Raw`; load→compile→run→RunStore.Insert. `cmd/worker/main.go`: register `TaskDMIngest`. (Loader/Compiler/RunStore/Gate sudah tersedia di `runner.Runner` — tidak perlu wiring baru.)
- **B7. Tests.** Unit: 4 trigger match/no-match per subtype (semua `Source=dm`); `conversation-state` open/closed/absent; `send-dm` guard-skip (window absen), gate Allow→SendDM, Queue/Reject reported, dedupe. Parser: `ExtractMessagingEvents` untuk dm/story-reply/story-mention(attachment)/ad-referral(message & top-level referral) + payload malformed di-skip. Handler `processMessaging` (fakedb pola existing). Grep-guard: engine core & `libs/safety` core diff kosong; `libs/workflow/nodes/*` tak impor `libs/kits/*` maupun `libs/safety`; tak ada `graph.facebook.com`; tak ada parsing `changes[].mentions`.

### 6.2 frontend-ui-engineer

- **F1. Katalog & tipe (KONTRAK — sepakati dulu dgn B5).** `packages/types/src/workflow.ts`: `runnable=true` untuk `dm-received`, `story-reply`, `story-mention`, `click-to-dm-ad`, `conversation-state`, `send-dm` (mirror `catalog.go`). Tambah `configSchema`: `send-dm` (`template` textarea, help "{nama}"), `conversation-state` (`requireOpen` boolean, help ramah "hanya lanjut kalau chat masih aktif < 24 jam"), `click-to-dm-ad` (`adRef` text opsional). Tambah interface config: `SendDmConfig{template?:string}`, `ConversationStateConfig{requireOpen?:boolean}`, `ClickToDmAdConfig{adRef?:string}`.
- **F2. Copy help window-24h.** Untuk `send-dm`, help copy WAJIB menyebut "hanya bekerja saat percakapan DM/Story masih dalam 24 jam — pada workflow dari komentar tidak akan mengirim" (AC-6). Bahasa olshop, jelas, tanpa jargon.
- **F3. Palette & badge.** 6 node pindah dari badge "segera" → aktif (drag-able, boleh di-`activate`). Tidak menambah node/label yang menyiratkan IG Live atau blast (§4b/§9). Pill "● LIVE" tetap = workflow aktif.
- **F4. Inspector (nol komponen baru).** `SchemaForm` schema-driven ADR-005 sudah merender `textarea`/`boolean`/`text` — 6 node cukup lewat `configSchema`, **tanpa** komponen inspector baru (§12a). Validasi inline (banner/highlight) reuse `VALIDATION_FAILURE_MESSAGES`.
- **F5. Guard desain.** Token §11 (dark/lime, mono untuk label teknis); copy default BI. Tidak ada elemen yang menyiratkan data IG Live / follower / broadcast.

> Tidak ada layar baru. `send-dm`/story/DM trigger muncul di builder yang sudah ada (React Flow, ADR-005). Runs screen menampilkan `trigger_source` = `dm` otomatis dari `workflow_run` (kolom sudah ada).

---

## 7. Urutan & Dependency

```
BACKEND (B0 sudah PASS — tinggal ikuti §3 hasil koreksi)
  B1 migrasi ─► B2 sqlc ─┬─► B6 dm_ingest ─────────────┐
                         └─► B4 webhook parsing ────────┼─► B7 tests
  B3 task types+enqueue ─────────────────────────────► B4/B6
  B5 nodes (6) ─────────────────────────────────────► B6 (runtime) ; B7
  (B5 paralel dgn B1–B4; hanya butuh kesepakatan katalog F1)

FRONTEND
  F1 katalog/tipe (KONTRAK) ─► F2 copy ─► F3 palette ─► F4 inspector ; F5 paralel
```

Blocking utama:
- **B1→B2** memblokir B6 (window store) & B4 (dedupe).
- **B5 (nodes)** memblokir makna runtime; bisa paralel dengan B1–B4 setelah kesepakatan katalog **F1**.
- **F1** = kontrak lintas agen; sepakati sebelum B5 & F-lain agar tak revisi ganda.
- Backend bisa menyelesaikan B1–B7 (runtime hidup) tanpa FE; FE bisa mulai F1 di atas katalog yang disepakati.

---

## 8. Risiko / Keputusan Terbuka (rekomendasi arsitek)

- **R1 — Enqueue gate = `HasLiveWorkflow` (bukan per-trigger-type).** Enqueue `dm:ingest` bila akun punya ≥1 workflow live, tanpa mengecek apakah ada trigger DM/story. Event tanpa trigger cocok → run tak fire (skipped, tak ditulis, R2 ADR-004). **Rekomendasi: terima** (murah, reuse query; optimasi `HasLiveWorkflowWithMessagingTrigger` = premature, §12a-4).
- **R2 — Window store di Postgres.** Lihat §4.2. **Rekomendasi: terima** (durabilitas + domain entity + preseden `processed_comment`).
- **R3 — Story-mention MEMBUKA window 24h (diperbaiki, koreksi B0).** Premis ADR draft awal ("mention bukan pesan → tak membuka window") **salah**: story-mention adalah event messaging (tiba di `entry[].messaging[]`), jadi di bawah messaging window ia **membuka/refresh window 24h seperti story-reply**. Karena `Source` kini `dm` untuk semua permukaan messaging, cabang window §4.1 (upsert setiap event) otomatis mencakupnya — tanpa cabang khusus. **Rekomendasi: terima** (perlakukan sama dengan story-reply). Alternatif konservatif (tidak membuka window untuk mention) tetap compliant — hanya membuat `send-dm` lebih sering skip — tetapi tidak dipilih karena Meta memang membuka window untuk mention.
- **R4 — `send-dm` di flow comment selalu skip.** Sesuai instruksi scope. Guard presence `last_interaction_at` (absen di flow comment) → skip lokal sebelum Sender. **Rekomendasi: terima + copy help node menjelaskannya** (AC-6). Tidak menyembunyikan node dari flow comment (builder netral); cukup jelas via copy.
- **R5 — Cross-source window (comment ingest tidak baca conversation).** `comment_ingest` tak diubah untuk mengisi `Raw[last_interaction_at]`. Commenter yang kebetulan punya window DM terbuka tak terdeteksi di flow comment. **Rekomendasi: tunda** — enhancement kecil (satu `GetConversation` di comment_ingest) yang bisa ditambah tanpa perubahan kontrak. Bukan blocker MVP.
- **R6 — `FromUsername` kosong pada event messaging.** Payload `messages` sering hanya `sender.id`. `{nama}` fallback ke kosong/"kak". Resolusi `GET /{igsid}` butuh verifikasi izin guardian → **tunda**. Copy default olshop menoleransi sapaan tanpa nama.
- **R7 — `Kind` literal "dm" di node netral.** `action_send_dm.go` memakai literal `"dm"` (== `safety.KindDM`), tak mengimpor `libs/safety` (pola sama `replyCommentKind`). **Rekomendasi: terima** (wire-key convention, §12a-4).
- **R8 — `changes[].mentions` (@mention komentar/caption) di-drop.** Kapabilitas §4a berbeda dari story-mention (comment-level, dedupe by `comment_id`). Tidak dibutuhkan untuk 6 node ADR ini. **Rekomendasi: tunda** ke ADR terpisah bila ada permintaan fitur; jangan campur ke jalur messaging.

---

## 9. Guardrail Ringkas (verifikasi PR)

- Engine core `libs/workflow/{engine,node,context,gate,event}.go` **tidak diubah**; `libs/safety` core **tidak diubah** (window 24h sudah ada) — grep diff kosong untuk file-file ini.
- `libs/workflow/nodes/*` (termasuk 6 node baru) **tidak** mengimpor `libs/kits/*` maupun `libs/safety` (§8/§9 boundary). Konstanta `Kind="dm"`/24h = literal lokal.
- `send-dm` memanggil `rc.Gate.Allow(Kind="dm")` **sebelum** `rc.Sender.SendDM` (§10 one-door), dan **guard presence** window sebelum gate. Tidak ada outbound IG yang mem-bypass safety.
- `nodes.Catalog` tetap tidak memuat item §4b (follower/blast/auto-follow/IG-Live). `send-dm` = 1:1 window-gated + dedupe (bukan §4b.6). `click-to-dm-ad` = ad-referral entry (percakapan sah).
- Hanya `graph.instagram.com` (§4.0) — tidak ada `graph.facebook.com`; `postback.referral` (konstruk FB) tidak diandalkan. Akun & kontak diresolusi via **IGSID** (`entry.id`/`sender.id`), bukan Page id.
- Story-mention diparse dari `message.attachments[].type=="story_mention"` (permukaan messaging, `Source=dm`) — **bukan** `changes[].mentions`. Langganan webhook cukup field **`messages`** (produk Instagram).
- `packages/types` `NODE_CATALOG.runnable` + config schema selaras `libs/workflow/nodes/catalog.go` (kontrak lintas bahasa §12a-1) — commit yang sama.
- Copy default Bahasa Indonesia gaya olshop; token §11; pill "● LIVE" = workflow aktif, bukan IG Live (§9).

---

## 10. Alternatif yang Ditolak

- **Window store di Redis.** Ditolak sebagai source of truth: window 24h = state percakapan compliance-critical yang tak boleh hilang saat restart/flush (§4.2). Redis tetap untuk quota/dedupe (counter fana). Cache read Redis boleh menyusul sebagai optimasi, write tetap Postgres.
- **Story-mention lewat `changes[].mentions`.** Ditolak (koreksi B0): shape itu adalah @mention komentar/caption (comment-level, §4a), bukan story mention. Story-mention adalah event messaging (`message.attachments[].story_mention`). Memakai `changes[].mentions` untuk story = salah kapabilitas & salah dedupe key.
- **Menambah cabang DM di dalam `comment_ingest.go`.** Ditolak: comment ingest coupled ke `catalog_post`/seller enrichment; mencampur DM (yang tak punya catalog) melanggar SoC (§12a-3) dan memaksa cabang kondisional. Task terpisah `dm:ingest` lebih bersih (mirror, bukan merge).
- **Mengubah `libs/safety/window.go` agar `Kind=dm` menolak `CommentAt` zero.** Ditolak: melanggar guardrail "safety core tak diubah" dan akan menggeser tanggung jawab tracking. `send-dm` guard presence sendiri lebih eksplisit & lokal.
- **Menambah field `Subtype`/`WindowAt` ke struct `workflow.Event`.** Ditolak: engine core dibekukan (ADR-004/005). `Event.Raw` sudah kanal yang tepat untuk konteks runtime (preseden `RawKeyKode`/`comment_at`).
- **`send-dm` sebagai node Kit (seller).** Ditolak: DM adalah primitif percakapan netral (dipakai Creator/Booking juga, §8). Menaruhnya di Kit = duplikasi lintas Kit dan bocornya konsep ke tiap segmen.
- **Mengandalkan `postback.referral` untuk ad-referral.** Ditolak sebagai jalur utama: itu konstruk Messenger/Facebook, bukan Instagram Login (§4.0). Ad-referral Instagram = `message.referral` (thread baru) / top-level `referral` (thread lama). `postback` hanya fallback toleran.
- **Resolusi username via scraping/lookup non-OAuth.** Ditolak (§4b.7). `{nama}` hanya dari payload webhook; kosong bila IG tak menyediakan.
