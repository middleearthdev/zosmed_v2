/**
 * DTO contracts for the Comment-to-Order REST endpoints (CLAUDE.md §8.1.4).
 * Selaras dengan apps/api/internal/commentorder/dto.go — ubah keduanya dalam
 * satu commit jika ada perubahan shape (§5a, §12a-1).
 *
 * Referensi endpoint: docs/specs/comment-to-order.md §4.2 – §4.4.
 */
import type { Id, ISODateTime } from './common';
import type { ReservationStatus } from './domain';

// ── Incoming comments ────────────────────────────────────────────────────────

/** Satu komentar masuk dari post/Reel terdaftar (hasil agregat endpoint). */
export interface IncomingCommentDTO {
  id: Id;
  /** Handle IG tanpa '@'. Mapping FE: u */
  user: string;
  /** Teks komentar asli. Mapping FE: t */
  text: string;
  /** Relative time server-rendered (misal "2m", "14m"). Mapping FE: tm */
  ago: string;
  /** Kode keep/C yang terdeteksi, atau null. Mapping FE: match */
  matchedCode: string | null;
  /** true = kode valid & reservasi dibuat. Mapping FE: ok */
  reserved: boolean;
  /** true = komentar duplikat, dilewati. Mapping FE: dupe */
  duplicate: boolean;
}

// ── Reservations ─────────────────────────────────────────────────────────────

/** Satu baris reservasi dalam antrian stok (queue). */
export interface ReservationDTO {
  id: Id;
  /** Kode keep/C (misal "C3"). */
  code: string;
  /** Handle pembeli tanpa '@'. Mapping FE: u */
  buyerHandle: string;
  /** Nama produk + varian (misal "Sage Tote · M"). Mapping FE: p */
  product: string;
  /** Label harga terformat (misal "Rp 189rb"). Mapping FE: price */
  priceLabel: string;
  /**
   * Status canonical reservasi (CLAUDE.md §6 / §8.1).
   * FE memetakan 'expired-released' → 'expired' untuk label pendek.
   */
  status: ReservationStatus;
  /**
   * Label countdown terformat server (misal "4:52", "✓ closed", "— released").
   * Mapping FE: cd. FE juga bisa hitung dari expiresAt untuk countdown live.
   */
  countdownLabel: string;
  /** ISO timestamp; sumber countdown live di FE untuk status reserved/waiting-pay. */
  expiresAt: ISODateTime;
}

// ── Product catalog ───────────────────────────────────────────────────────────

/** Produk dalam catalog post beserta stok realtime. */
export interface CatalogProductDTO {
  code: string;
  name: string;
  /** Stok tersisa. Mapping FE: left */
  stockLeft: number;
  /** Stok total awal. Mapping FE: total */
  stockTotal: number;
}

// ── Stats ─────────────────────────────────────────────────────────────────────

/**
 * Satu KPI kartu statistik di header layar Comment-to-Order.
 * Warna (color) TIDAK dikirim backend — FE menderivasi dari key menggunakan
 * token §11 (presentational concern, SoC §12a-3).
 */
export interface CommentOrderStatDTO {
  /** Kunci mesin untuk derivasi warna di FE. */
  key: 'code-detected' | 'reserved-now' | 'closed-wa' | 'expired';
  /** Label tampilan (Bahasa Indonesia, misal "CODE DETECTED"). */
  label: string;
  /** Nilai string terformat (misal "147"). */
  value: string;
}

// ── Agregat response ──────────────────────────────────────────────────────────

/**
 * Respons endpoint `GET /api/v1/comment-order?accountId=&postId=`.
 * Dibungkus envelope `{ data: CommentOrderResponse, error: null }`.
 */
export interface CommentOrderResponse {
  /** Label komentar post (misal "💬 412 komentar"). Mapping FE: postComments */
  postCommentsLabel: string;
  comments: IncomingCommentDTO[];
  stats: CommentOrderStatDTO[];
  reservations: ReservationDTO[];
  products: CatalogProductDTO[];
}

// ── Settings ──────────────────────────────────────────────────────────────────

/**
 * Shape untuk `GET/PUT /api/v1/comment-order/settings?accountId=`.
 * Keywords default dari KIT_KEYWORDS.seller (constants.ts) — satu sumber (§12a-1).
 */
export interface SettingsDTO {
  /** Kata kunci yang memicu deteksi kode (contoh: ["keep","c","c1","c3"]). */
  keywords: string[];
  /** Durasi hold stok dalam detik (default 300 = 5 menit). */
  holdSeconds: number;
  /** Template teks private reply; mendukung variabel {{nama}}, {{kode}}, {{produk}}. */
  replyTemplate: string;
}
