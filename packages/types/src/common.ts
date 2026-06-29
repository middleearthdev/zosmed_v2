/** Shared scalar/brand types used across domain contracts. */

/** ISO-8601 timestamp string, e.g. "2026-06-28T09:30:00Z". */
export type ISODateTime = string;

/** Opaque entity id (UUID/string). */
export type Id = string;

/** Product segment — selects which Kit is loaded (CLAUDE.md §8). */
export type Segment = 'seller' | 'creator' | 'booking';

/** 24-hour messaging window state (CLAUDE.md §4c). */
export type WindowState = 'open' | 'closed';
