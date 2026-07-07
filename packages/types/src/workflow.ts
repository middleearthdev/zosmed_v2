/**
 * DTO contracts for the Workflow REST endpoints (ADR-004 §3).
 * Selaras dengan `apps/api/internal/workflow/dto.go` — ubah keduanya dalam
 * satu commit jika ada perubahan shape (§5a, §12a-1).
 *
 * Reuse `Workflow` / `WorkflowNode` / `WorkflowEdge` / `WorkflowStatus` /
 * `NodeKind` / `RunStatus` dari `domain.ts` — TIDAK didefinisikan ulang di
 * sini (hindari duplikasi, CLAUDE.md §12a-1).
 *
 * Referensi: docs/specs/workflow.md §3, §5.
 */
import type { Id, ISODateTime, Segment } from './common';
import type {
  ActionKind,
  FilterKind,
  RunStatus,
  TriggerKind,
  WorkflowEdge,
  WorkflowNode,
  WorkflowStatus,
} from './domain';

// ── CRUD DTOs ─────────────────────────────────────────────────────────────

/** Baris ringkas untuk layar daftar workflow (`GET /api/v1/workflows`). */
export interface WorkflowSummary {
  id: Id;
  name: string;
  status: WorkflowStatus;
  segment: Segment;
  nodeCount: number;
  updatedAt: ISODateTime;
}

/** Body `POST /api/v1/workflows`. */
export interface CreateWorkflowRequest {
  name: string;
  segment: Segment;
}

/**
 * Body `PUT /api/v1/workflows/{id}`. Save = replace penuh (ADR-004 §8 R4):
 * seluruh node/edge canvas dikirim ulang; server hapus+insert transaksional.
 * `nodes[].id` boleh id lama (dipertahankan) atau kosong/baru (server assign).
 */
export interface SaveWorkflowRequest {
  name: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

/** Respons `DELETE /api/v1/workflows/{id}`. */
export interface DeleteWorkflowResponse {
  deleted: true;
}

// ── Activate validation (422 `validation_failed`) ───────────────────────────

/** Alasan gagal saat `POST /api/v1/workflows/{id}/activate` (ADR-004 §3). */
export type ValidationFailureReason =
  | 'no_trigger'
  | 'no_action'
  | 'unknown_node_type'
  | 'trigger_not_runnable'
  | 'cycle';

/** Copy Bahasa Indonesia per alasan gagal — satu sumber (§12a-1 DRY). */
export const VALIDATION_FAILURE_MESSAGES: Record<ValidationFailureReason, string> = {
  no_trigger: 'Workflow butuh minimal 1 trigger sebelum bisa diaktifkan.',
  no_action: 'Workflow butuh minimal 1 action sebelum bisa diaktifkan.',
  unknown_node_type: 'Ada node dengan tipe yang belum dikenali sistem — hapus atau ganti node tersebut.',
  trigger_not_runnable: 'Trigger yang dipakai belum didukung untuk dijalankan otomatis (masih "segera hadir").',
  cycle: 'Alur node membentuk lingkaran (cycle) — perbaiki koneksi antar node dulu.',
};

// ── Runs ─────────────────────────────────────────────────────────────────

/** Satu langkah eksekusi dalam `RunSummary.steps` (serialisasi `workflow.StepLog`). */
export interface RunStepDTO {
  nodeKey: string;
  kind: 'trigger' | 'filter' | 'action';
  status: string;
  detail: string;
}

/**
 * Baris ringkas run (`GET /api/v1/workflows/{id}/runs` atau `GET /api/v1/runs`).
 * `status` selalu salah satu dari `success`/`failed`/`skipped` di sini —
 * subset dari `RunStatus` domain.ts (`queued`/`running` dipakai konteks lain).
 */
export interface RunSummary {
  id: Id;
  workflowId: Id | null;
  workflowName: string;
  triggerSummary: string;
  status: RunStatus;
  durationMs: number;
  steps: RunStepDTO[];
  at: ISODateTime;
}

// ── Node catalog (feasible-only, CLAUDE.md §7) ──────────────────────────────

/** Union semua `node_type` feasible di seluruh kategori. */
export type AnyNodeType = TriggerKind | FilterKind | ActionKind;

// ── Config-schema (inspector schema-driven, ADR-005 §R5/§F4) ────────────────

/**
 * Jenis kontrol yang dirender inspector (`SchemaForm`). Ini MURNI concern
 * render frontend (ADR-005 opsi-1): Go tidak lagi menyimpan schema. Beberapa
 * jenis sengaja "ramah pengguna" dan menyembunyikan format teknis yang
 * dibutuhkan backend — konversi terjadi di dalam kontrol, bukan di kepala user:
 *
 * - `time`     → input jam:menit, DISIMPAN sebagai menit sejak 00:00 (number)
 * - `weekdays` → toggle hari, DISIMPAN sebagai nomor hari string ("0"=Min … "6"=Sab)
 * - `phone`    → input nomor, DINORMALISASI ke digit E.164 (awalan 0 → 62)
 * - `list`     → input chip (ketik lalu Enter), DISIMPAN sebagai string[]
 *
 * Nilai yang dihasilkan tetap cocok dengan yang divalidasi node Go di
 * `Factory.Build` (mis. filter_time_window menerima menit + nomor hari).
 */
export type FieldKind = 'text' | 'textarea' | 'list' | 'select' | 'boolean' | 'time' | 'weekdays' | 'phone';

/** Satu pilihan untuk field bertipe `select`. */
export interface FieldOption {
  value: string;
  label: string;
}

/** Deskripsi satu field config sebuah node untuk inspector builder (FE-only). */
export interface FieldSchema {
  key: string;
  type: FieldKind;
  label: string;
  required?: boolean;
  /** Kalimat pendukung singkat & ramah (hindari jargon/angka teknis). */
  help?: string;
  placeholder?: string;
  /** Hanya bermakna saat `type === 'select'`. */
  options?: FieldOption[];
  /** Nilai awal saat node baru ditambahkan dari palette (seed FE saja). */
  default?: unknown;
}

/**
 * Satu entri katalog node. Statik di FE — mirror `libs/workflow/nodes/catalog.go`
 * (ADR-004 R7, pola sama `KIT_KEYWORDS`). TIDAK ada endpoint `/node-catalog`
 * dipanggil di iterasi ini; `GET /api/v1/node-catalog` opsional di backend.
 *
 * Catatan: tidak menyertakan `iconKey`/warna — itu murni presentational dan
 * dipetakan di `apps/web` (SoC §12a-3), supaya `packages/types` tidak perlu
 * bergantung ke `@zosmed/ui`.
 */
export interface NodeCatalogEntry {
  category: 'trigger' | 'filter' | 'action';
  nodeType: AnyNodeType;
  /** Label tampilan Bahasa Indonesia gaya olshop. */
  label: string;
  description: string;
  /** false = tampil di palette dengan badge "segera", disabled untuk drag/aktivasi. */
  runnable: boolean;
  /**
   * Schema config yang dirender inspector (ADR-005 §R5). Ada untuk node
   * runnable yang punya config user-facing; `undefined` untuk node tanpa
   * config atau yang belum pindah ke form schema-driven.
   */
  configSchema?: FieldSchema[];
}

/**
 * Katalog node feasible §7. Subset runnable (ADR-004 §5 + ADR-005 §1):
 * trigger `comment-received`, `comment-to-order`; filter `keyword-match`,
 * `post-selection`, `time-window`; action `send-whatsapp-link`,
 * `reserve-stock`, `reply-comment`, `outbound-webhook`. Sisanya tampil tapi
 * non-runnable ("segera") sampai subsystem-nya ada (ADR-005 §1 klasifikasi B).
 */
export const NODE_CATALOG: readonly NodeCatalogEntry[] = [
  // Triggers
  {
    category: 'trigger',
    nodeType: 'comment-received',
    label: 'Komentar IG masuk',
    description: 'Trigger saat ada komentar baru di post/Reel (webhook comments).',
    runnable: true,
  },
  {
    category: 'trigger',
    nodeType: 'comment-to-order',
    label: 'Comment-to-Order (keep/C)',
    description: 'Deteksi kode keep/C di komentar post/Reel → reserve stok otomatis.',
    runnable: true,
  },
  {
    category: 'trigger',
    nodeType: 'dm-received',
    label: 'DM IG masuk',
    description: 'Trigger saat user memulai DM duluan.',
    runnable: false,
  },
  {
    category: 'trigger',
    nodeType: 'story-reply',
    label: 'Balasan Story',
    description: 'Trigger saat user membalas Story (membuka window 24 jam).',
    runnable: false,
  },
  {
    category: 'trigger',
    nodeType: 'story-mention',
    label: 'Mention di Story',
    description: 'Trigger saat akun di-mention di Story orang lain.',
    runnable: false,
  },
  {
    category: 'trigger',
    nodeType: 'click-to-dm-ad',
    label: 'Klik iklan Click-to-DM',
    description: 'Trigger dari entry point iklan yang membuka percakapan.',
    runnable: false,
  },
  // Filters
  {
    category: 'filter',
    nodeType: 'keyword-match',
    label: 'Cocokkan kata kunci',
    description: 'Lanjut hanya jika teks mengandung salah satu kata kunci.',
    runnable: true,
    configSchema: [
      {
        key: 'keywords',
        type: 'list',
        label: 'Kata yang dipantau',
        help: 'Workflow lanjut kalau komentar memuat salah satu kata ini. Kosongkan untuk semua komentar.',
        placeholder: 'ketik kata lalu Enter — mis. keep, order, mau',
        default: [],
      },
    ],
  },
  {
    category: 'filter',
    nodeType: 'conversation-state',
    label: 'Status percakapan',
    description: 'Cek apakah percakapan masih dalam window 24 jam.',
    runnable: false,
  },
  {
    category: 'filter',
    nodeType: 'intent',
    label: 'Intent: ragu / trust',
    description: 'Deteksi keraguan pembeli ("real kak?", "ga tipu2 kan?").',
    runnable: false,
  },
  {
    category: 'filter',
    nodeType: 'post-selection',
    label: 'Pilih post tertentu',
    description: 'Filter berdasarkan post/Reel asal komentar.',
    runnable: true,
    configSchema: [
      {
        key: 'mediaIds',
        type: 'list',
        label: 'Batasi ke post/Reel tertentu',
        help: 'Opsional. Biarkan kosong supaya berlaku di semua post. Isi ID post kalau mau membatasi.',
        placeholder: 'tempel ID post lalu Enter',
        default: [],
      },
    ],
  },
  {
    category: 'filter',
    nodeType: 'time-window',
    label: 'Jendela waktu',
    description: 'Lanjut hanya pada rentang waktu tertentu.',
    runnable: true,
    configSchema: [
      {
        key: 'days',
        type: 'weekdays',
        label: 'Hari aktif',
        help: 'Pilih hari workflow ini boleh jalan. Tidak dipilih = setiap hari.',
        default: [],
      },
      { key: 'startMinute', type: 'time', label: 'Mulai jam' },
      { key: 'endMinute', type: 'time', label: 'Sampai jam' },
      {
        key: 'timezone',
        type: 'select',
        label: 'Zona waktu',
        default: 'Asia/Jakarta',
        options: [
          { value: 'Asia/Jakarta', label: 'WIB (Jakarta)' },
          { value: 'Asia/Makassar', label: 'WITA (Makassar)' },
          { value: 'Asia/Jayapura', label: 'WIT (Jayapura)' },
        ],
      },
    ],
  },
  // Actions
  {
    category: 'action',
    nodeType: 'reply-comment',
    label: 'Balas komentar publik',
    description: 'Kirim balasan publik ke komentar (rate-limited).',
    runnable: true,
    configSchema: [
      {
        key: 'template',
        type: 'textarea',
        label: 'Balasan otomatis',
        help: 'Ketik {nama} untuk menyapa nama pengomennya. Kosongkan untuk pakai balasan bawaan.',
        placeholder: 'Halo kak {nama}, cek DM ya 🙏',
      },
    ],
  },
  {
    category: 'action',
    nodeType: 'send-dm',
    label: 'Kirim DM',
    description: 'Kirim direct message (window 24 jam).',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'ai-reply',
    label: 'Balasan AI (olshop)',
    description: 'Balasan otomatis pakai persona AI olshop.',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'send-whatsapp-link',
    label: 'Kirim link WhatsApp',
    description: 'Private reply berisi link wa.me terisi otomatis.',
    runnable: true,
    configSchema: [
      {
        key: 'phone',
        type: 'phone',
        label: 'Nomor WhatsApp tujuan',
        required: true,
        help: 'Nomor admin untuk closing. Boleh tulis 0812… atau 62812…',
        placeholder: '0812 3456 7890',
      },
      {
        key: 'template',
        type: 'textarea',
        label: 'Pesan pembuka',
        help: 'Ketik {nama} untuk nama pengomen, {wa_link} untuk link WhatsApp-nya.',
        default: 'Halo kak {nama}! Yuk lanjut ngobrol di WhatsApp ya: {wa_link}',
      },
    ],
  },
  {
    category: 'action',
    nodeType: 'send-trust-kit',
    label: 'Kirim trust-kit',
    description: 'Kirim testimoni/real-pict/resi saat terdeteksi keraguan.',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'reserve-stock',
    label: 'Reserve stok',
    description: 'Tahan stok + countdown untuk comment-to-order.',
    runnable: true,
  },
  {
    category: 'action',
    nodeType: 'notify-optin',
    label: 'Notifikasi opt-in',
    description: 'Kirim one-time notification ke user yang sudah opt-in.',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'handoff-human',
    label: 'Hand-off ke admin',
    description: 'Alihkan percakapan ke admin manusia.',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'tag-contact',
    label: 'Tag kontak',
    description: 'Tambahkan tag ke kontak (CRM internal).',
    runnable: false,
  },
  {
    category: 'action',
    nodeType: 'outbound-webhook',
    label: 'Webhook keluar',
    description: 'Kirim event ke backend eksternal milik pengguna.',
    runnable: true,
    configSchema: [
      { key: 'url', type: 'text', label: 'URL tujuan', required: true, help: 'Alamat https server kamu yang menerima data event.', placeholder: 'https://…' },
      { key: 'includeSignature', type: 'boolean', label: 'Tanda tangani request (HMAC)' },
      { key: 'secret', type: 'text', label: 'Secret', help: 'Kunci untuk verifikasi tanda tangan. Isi kalau tanda tangan diaktifkan.' },
    ],
  },
] as const;

/** Cek apakah sebuah node_type ada di katalog & runnable (dipakai validasi FE, gate publish tombol). */
export function isNodeTypeRunnable(nodeType: string): boolean {
  return NODE_CATALOG.some((n) => n.nodeType === nodeType && n.runnable);
}

/** Cari entri katalog by node_type. */
export function findCatalogEntry(nodeType: string): NodeCatalogEntry | undefined {
  return NODE_CATALOG.find((n) => n.nodeType === nodeType);
}

// ── Node config shapes (per node kind, ADR-004 catatan integrasi) ──────────

/** Config `keyword-match` filter. `caseInsensitive` disembunyikan dari UI (default true di Go). */
export interface KeywordMatchConfig {
  keywords: string[];
  caseInsensitive?: boolean;
}

/**
 * Config `send-whatsapp-link` action. Nama field selaras struct Go
 * `sendWhatsAppLinkConfig` (`libs/workflow/nodes/action_wa_link.go`): key
 * `phone` (bukan `waPhone`) dan placeholder template `{nama}`/`{wa_link}`.
 */
export interface SendWhatsappLinkConfig {
  /** Nomor WhatsApp tujuan, format internasional tanpa "+" (mis. "6281234567890"). */
  phone: string;
  /** Template teks; placeholder {nama}, {wa_link}. */
  template?: string;
}

/** Config `post-selection` filter (ADR-005 §2.1). Kosong = izinkan semua post/Reel. */
export interface PostSelectionConfig {
  /** ID media Instagram yang diizinkan. */
  mediaIds: string[];
}

/** Config `time-window` filter (ADR-005 §2.2). Kosong = setiap hari, setiap waktu. */
export interface TimeWindowConfig {
  /** Nomor hari time.Weekday sebagai string ("0"=Minggu … "6"=Sabtu). */
  days?: string[];
  /** Menit sejak tengah malam lokal [0,1439], inklusif. */
  startMinute?: number;
  endMinute?: number;
  /** IANA tz; default "Asia/Jakarta" (WIB). */
  timezone?: string;
}

/** Config `reply-comment` action (ADR-005 §2.3). Placeholder: {nama}. */
export interface ReplyCommentConfig {
  template: string;
}

/** Config `outbound-webhook` action (ADR-005 §2.4). URL non-IG, ber-guard SSRF di backend. */
export interface OutboundWebhookConfig {
  url: string;
  includeSignature?: boolean;
  secret?: string;
}
