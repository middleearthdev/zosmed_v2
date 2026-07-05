/**
 * Single source for the API origin (CLAUDE.md §12a-1 DRY). Consumed by
 * `next.config.ts` (rewrites), `lib/get-me.ts` (server-side direct fetch),
 * and `lib/mock/api.ts` (comment-order fetch).
 */
export const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
