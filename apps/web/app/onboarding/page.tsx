import { redirect } from 'next/navigation';
import { getMe } from '@/lib/get-me';
import { getInstagramConnectUrl } from '@/lib/mock/api';
import { OnboardingClient } from './OnboardingClient';

interface OnboardingPageProps {
  searchParams: Promise<{ connected?: string }>;
}

/**
 * Server component: auth guard + fetch real state (ADR-003 §5.2/§5.3, AC-11).
 * `OnboardingClient` is purely presentational — it receives `user`/`account`
 * as props and never fetches on its own (§12a-3 SoC).
 */
export default async function OnboardingPage({ searchParams }: OnboardingPageProps) {
  const [me, params] = await Promise.all([getMe(), searchParams]);

  if (!me) redirect('/login');
  if (me.user.onboardingCompleted) redirect('/dashboard');

  return (
    <OnboardingClient
      user={me.user}
      account={me.account}
      connectUrl={getInstagramConnectUrl()}
      justConnected={params.connected === '1'}
    />
  );
}
