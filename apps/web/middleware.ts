import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/**
 * Coarse route guard (ADR-003 §5.3, AC-10). Only checks whether the `zsid`
 * session cookie is present — it does NOT validate the session against the
 * backend (that would mean a network round-trip on every navigation). Fine-
 * grained checks (session actually valid, onboarding completed) happen in
 * `(app)/layout.tsx` and `app/onboarding/page.tsx` via `getMe()`.
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

/** Auth pages that should bounce an already-logged-in user to `/dashboard`. */
const AUTH_PATHS = ['/login', '/register'];

export function middleware(request: NextRequest) {
  const hasSession = request.cookies.has(SESSION_COOKIE);
  const { pathname } = request.nextUrl;

  const isAuthPath = AUTH_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`));
  if (isAuthPath) {
    if (hasSession) return NextResponse.redirect(new URL('/dashboard', request.url));
    return NextResponse.next();
  }

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
    '/login',
    '/login/:path*',
    '/register',
    '/register/:path*',
  ],
};
