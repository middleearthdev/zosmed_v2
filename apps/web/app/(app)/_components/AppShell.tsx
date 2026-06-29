import type { ReactNode } from 'react';
import type { Account } from '@zosmed/types';
import { Sidebar } from './Sidebar';

/**
 * Authenticated app frame: fixed sidebar + main column (§9). Each page owns its
 * own 56px header bar — the dashboard uses the breadcrumb `Topbar`, most other
 * screens use the title-style `PageHeader` (matches the design artboards).
 */
export function AppShell({ account, children }: { account: Account; children: ReactNode }) {
  return (
    <div className="bg-bg text-text flex h-screen">
      <Sidebar account={account} />
      <main className="flex flex-1 flex-col overflow-hidden">{children}</main>
    </div>
  );
}
