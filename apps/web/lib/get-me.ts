/**
 * Server-only session lookup (ADR-003 §8.1, §14). Split out from `lib/auth.ts`
 * because Next.js forbids importing `next/headers` from any module a Client
 * Component can reach — `lib/auth.ts` backs client forms (`LoginForm` etc.),
 * so the one piece that needs `cookies()` lives here instead.
 */
import { cookies } from 'next/headers';
import type { ApiEnvelope, MeResponse } from '@zosmed/types';
import { API_BASE } from './env';

/**
 * Current session user + linked IG account (`GET /api/v1/auth/me`). Returns
 * `null` on a missing/invalid session — never throws; callers (layouts,
 * pages) decide whether to redirect (§12a-3 SoC).
 *
 * Calls the API directly rather than the relative `/api/...` path: a Server
 * Component fetch has no implicit browser origin to resolve the Next rewrite
 * against, so we hit `API_BASE` and forward the incoming cookie by hand.
 */
export async function getMe(): Promise<MeResponse | null> {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  if (!cookieHeader) return null;

  try {
    const res = await fetch(new URL('/api/v1/auth/me', API_BASE), {
      headers: { cookie: cookieHeader },
      cache: 'no-store',
    });
    if (!res.ok) return null;
    const envelope: ApiEnvelope<MeResponse> = await res.json();
    return envelope.data ?? null;
  } catch (err) {
    console.warn('[getMe] fetch gagal:', err);
    return null;
  }
}
