/**
 * Domain constants defined ONCE and imported everywhere (DRY — CLAUDE.md §12a).
 * Rate-limit caps mirror CLAUDE.md §4c; treat as the system defaults.
 */
import type { ReservationStatus, Segment } from './index';

/** Practical rate-limit caps (CLAUDE.md §4c). */
export const RATE_LIMITS = {
  /** Comment replies per hour (Meta technical cap). */
  commentRepliesPerHour: 750,
  /** DMs per hour (safe; overflow is queued, not rejected). */
  dmPerHour: 200,
  /** DMs per day (behaviour-based soft limit). */
  dmPerDay: 1000,
  /** Comments per post per 5 minutes (human-paced). */
  commentsPerPostPer5min: 30,
  /** AI tokens per day (cost guard, soft). */
  aiTokensPerDay: 1_000_000,
} as const;

/** Auto-pause when usage reaches this fraction of a quota (CLAUDE.md §10). */
export const AUTO_PAUSE_THRESHOLD = 0.8;

/** Private reply must be sent within this many days of the comment (§4c). */
export const PRIVATE_REPLY_WINDOW_DAYS = 7;

/** Standard DM window in hours (§4c). */
export const MESSAGING_WINDOW_HOURS = 24;

/** Ordered reservation lifecycle (CLAUDE.md §6 / §8.1). */
export const RESERVATION_STATUSES: readonly ReservationStatus[] = [
  'reserved',
  'waiting-pay',
  'closed-wa',
  'expired-released',
] as const;

/** Default trigger keywords per Kit (CLAUDE.md §8). */
export const KIT_KEYWORDS: Record<Segment, readonly string[]> = {
  seller: ['keep', 'c', 'c1', 'c3', 'order'],
  creator: ['mau', 'info'],
  booking: ['booking', 'jadwal'],
} as const;
