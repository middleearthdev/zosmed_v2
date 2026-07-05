/** Shared scalar/brand types used across domain contracts. */

/** ISO-8601 timestamp string, e.g. "2026-06-28T09:30:00Z". */
export type ISODateTime = string;

/** Opaque entity id (UUID/string). */
export type Id = string;

/** Product segment — selects which Kit is loaded (CLAUDE.md §8). */
export type Segment = 'seller' | 'creator' | 'booking';

/** 24-hour messaging window state (CLAUDE.md §4c). */
export type WindowState = 'open' | 'closed';

// ── API envelope (mirrors backend `httpx.respond`, ADR-002/ADR-003 §4) ──────

/** Error shape nested inside `ApiEnvelope` — one shape, reused everywhere. */
export interface ApiErrorShape {
  code: string;
  message: string;
  /** Extra machine-readable detail, e.g. onboarding_incomplete's reason. */
  reason?: string;
}

/** Generic `{data,error}` envelope every backend endpoint responds with. */
export interface ApiEnvelope<T> {
  data: T | null;
  error: ApiErrorShape | null;
}
