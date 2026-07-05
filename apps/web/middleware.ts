import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/**
 * Coarse route guard (ADR-003 §5.3, AC-10). Runs on the edge WITHOUT backend
 * access, so it can only make the "negative" claim: *no cookie → definitely not
 * logged in → send to /login*. It must NOT make the "positive" claim (*cookie
 * present → logged in*), because a present-but-invalid cookie (tampered, expired,
 * or revoked) can't be detected here — doing so would fight the validated guard
 * in `(app)/layout.tsx` and cause an infinite redirect loop (ERR_TOO_MANY_REDIRECTS).
 *
 * The authoritative check (session actually valid, onboarding complete, and
 * bouncing an already-logged-in user away from /login) lives server-side in
 * `(app)/layout.tsx`, `app/onboarding/page.tsx`, and `app/{login,register}/page.tsx`
 * via `getMe()`, which validates against the backend.
 */
const SESSION_COOKIE = 'zsid';

/** Screens that require a logged-in user (CLAUDE.md §9 sidebar + onboarding). */
const PROTECTED_PATHS = [
  '/dashboard',
  '/workflows',
  '/inbox',
  '/ai',
  '/contacts',
  '/analytics',
  '/safety',
  '/templates',
  '/settings',
  '/team',
  '/notifications',
  '/kits',
  '/states',
  '/onboarding',
];

export function middleware(request: NextRequest) {
  const hasSession = request.cookies.has(SESSION_COOKIE);
  const { pathname } = request.nextUrl;

  // Only the negative claim: no cookie on a protected path → /login. Bouncing an
  // already-logged-in user off /login is done in app/{login,register}/page.tsx
  // (validated via getMe), never here (see file header — avoids redirect loops).
  const isProtected = PROTECTED_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`));
  if (isProtected && !hasSession) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  return NextResponse.next();
}

// Keep the matcher a literal array (Next statically analyzes it at build time —
// dynamically computed matchers are ignored, so this intentionally is NOT
// derived from PROTECTED_PATHS/AUTH_PATHS above).
export const config = {
  matcher: [
    '/dashboard/:path*',
    '/workflows/:path*',
    '/inbox/:path*',
    '/ai/:path*',
    '/contacts/:path*',
    '/analytics/:path*',
    '/safety/:path*',
    '/templates/:path*',
    '/settings/:path*',
    '/team/:path*',
    '/notifications/:path*',
    '/kits/:path*',
    '/states/:path*',
    '/onboarding',
    '/onboarding/:path*',
  ],
};
