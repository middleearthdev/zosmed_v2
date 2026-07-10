import type { ReactNode } from 'react';
import { redirect } from 'next/navigation';
import { getMe } from '@/lib/get-me';
import { adaptMeToAccount } from '@/lib/mock/api';
import { KitProvider } from '@/lib/kit-context';
import { AppShell } from './_components/AppShell';

/**
 * Auth guard for every screen under `(app)` (ADR-003 §5.3, AC-10). Coarse
 * cookie-presence check already happened in `proxy.ts`; here we verify
 * the session is actually valid and onboarding is complete before rendering.
 */
export default async function AppLayout({ children }: { children: ReactNode }) {
  const me = await getMe();
  if (!me) redirect('/login');
  if (!me.user.onboardingCompleted) redirect('/onboarding');

  const account = adaptMeToAccount(me);

  return (
    <KitProvider segment={account.kit}>
      <AppShell account={account}>{children}</AppShell>
    </KitProvider>
  );
}
