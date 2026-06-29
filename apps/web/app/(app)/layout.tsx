import type { ReactNode } from 'react';
import { getAccount } from '@/lib/mock/api';
import { KitProvider } from '@/lib/kit-context';
import { AppShell } from './_components/AppShell';

export default async function AppLayout({ children }: { children: ReactNode }) {
  const account = await getAccount();
  return (
    <KitProvider segment={account.kit}>
      <AppShell account={account}>{children}</AppShell>
    </KitProvider>
  );
}
