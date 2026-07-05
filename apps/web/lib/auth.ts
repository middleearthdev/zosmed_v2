/**
 * Client-safe auth actions (ADR-003 §8.1) — imported by `LoginForm`,
 * `RegisterForm`, `OnboardingClient`, and `Sidebar` (logout). Every action
 * hits a same-origin `/api/v1/...` path proxied by the Next rewrite
 * (`next.config.ts`), so the `zsid` session cookie rides along automatically.
 *
 * `getMe()` (server-only, needs `next/headers`) lives in `lib/get-me.ts`
 * instead — Next.js forbids importing `next/headers` from anything a Client
 * Component can reach, and this file backs client forms.
 *
 * SoC (§12a-3): this file owns fetching; presentational components only call
 * these functions and render the result.
 */
import type { AccountStatus, ApiEnvelope, ApiErrorShape, AppUser, Segment } from '@zosmed/types';

export type ApiError = ApiErrorShape;

export interface ActionResult<T> {
  ok: boolean;
  data?: T;
  error?: ApiError;
}

const NETWORK_ERROR: ApiError = {
  code: 'network_error',
  message: 'Tidak bisa terhubung ke server. Coba lagi.',
};

const UNKNOWN_ERROR: ApiError = {
  code: 'unknown_error',
  message: 'Terjadi kesalahan. Coba lagi.',
};

/** Friendlier Bahasa Indonesia copy per error code (falls back to server message). */
const ERROR_MESSAGES: Record<string, string> = {
  invalid_credentials: 'Email atau password salah.',
  email_taken: 'Email ini sudah terdaftar — coba masuk atau pakai email lain.',
  invalid_request: 'Data belum lengkap atau formatnya belum sesuai.',
  unauthorized: 'Sesi kamu berakhir, silakan masuk lagi.',
};

/** Single place to turn a backend error into copy shown to the user (§12a-1 DRY). */
export function authErrorMessage(error: ApiError | undefined): string {
  if (!error) return UNKNOWN_ERROR.message;
  return ERROR_MESSAGES[error.code] ?? error.message;
}

/**
 * POST/PUT JSON to a same-origin `/api/v1/...` path. One helper for every
 * auth/onboarding mutation below (§12a-1 DRY).
 */
async function request<T>(path: string, method: 'POST' | 'PUT', body?: unknown): Promise<ActionResult<T>> {
  try {
    const res = await fetch(path, {
      method,
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    });
    const envelope: ApiEnvelope<T> = await res.json();
    if (!res.ok || envelope.error) {
      return { ok: false, error: envelope.error ?? UNKNOWN_ERROR };
    }
    return envelope.data !== null ? { ok: true, data: envelope.data } : { ok: true };
  } catch {
    return { ok: false, error: NETWORK_ERROR };
  }
}

export interface LoginResult {
  user: AppUser;
  account: AccountStatus | null;
}

/** `POST /api/v1/auth/login`. */
export function login(email: string, password: string): Promise<ActionResult<LoginResult>> {
  return request<LoginResult>('/api/v1/auth/login', 'POST', { email, password });
}

/** `POST /api/v1/auth/register` — register aktif publik (ADR-003 §13.1). */
export function register(email: string, password: string, segment?: Segment): Promise<ActionResult<{ user: AppUser }>> {
  return request<{ user: AppUser }>('/api/v1/auth/register', 'POST', segment ? { email, password, segment } : { email, password });
}

/** `POST /api/v1/auth/logout` — no-op aman bila cookie sudah tidak ada. */
export function logout(): Promise<ActionResult<{ ok: boolean }>> {
  return request<{ ok: boolean }>('/api/v1/auth/logout', 'POST');
}

/** `PUT /api/v1/onboarding/segment`. */
export function setSegment(segment: Segment): Promise<ActionResult<{ user: AppUser }>> {
  return request<{ user: AppUser }>('/api/v1/onboarding/segment', 'PUT', { segment });
}

/** `POST /api/v1/onboarding/complete` — 409 bila segmen/koneksi IG belum lengkap. */
export function completeOnboarding(): Promise<ActionResult<{ user: AppUser }>> {
  return request<{ user: AppUser }>('/api/v1/onboarding/complete', 'POST');
}
