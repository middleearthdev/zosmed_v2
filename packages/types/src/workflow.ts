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
}

/**
 * Katalog node feasible §7, subset runnable = ADR-004 §5 iterasi 1:
 * `comment-received`, `comment-to-order` (trigger); `keyword-match` (filter);
 * `send-whatsapp-link`, `reserve-stock` (action). Sisanya tampil tapi
 * non-runnable ("segera") — R6.
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
    runnable: false,
  },
  {
    category: 'filter',
    nodeType: 'time-window',
    label: 'Jendela waktu',
    description: 'Lanjut hanya pada rentang waktu tertentu.',
    runnable: false,
  },
  // Actions
  {
    category: 'action',
    nodeType: 'reply-comment',
    label: 'Balas komentar publik',
    description: 'Kirim balasan publik ke komentar (rate-limited).',
    runnable: false,
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
    description: 'Private reply berisi link wa.me terisi otomatis (nama/produk/post).',
    runnable: true,
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
    runnable: false,
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

/** Config `keyword-match` filter. */
export interface KeywordMatchConfig {
  keywords: string[];
  caseInsensitive: boolean;
}

/** Config `send-whatsapp-link` action. */
export interface SendWhatsappLinkConfig {
  /** Template teks; mendukung variabel {{nama}}, {{produk}}, {{post}}. */
  template: string;
  /** Nomor WhatsApp tujuan format internasional tanpa "+" (mis. "6281234567890"). */
  waPhone: string;
}
