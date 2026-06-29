'use client';

import { usePathname } from 'next/navigation';
import { Logo, Meter } from '@zosmed/ui';
import type { Account } from '@zosmed/types';
import { SYSTEM_NAV, WORKSPACE_NAV } from './nav';
import { NavItem } from './NavItem';

export function Sidebar({ account }: { account: Account }) {
  const pathname = usePathname();
  const isActive = (href: string) => pathname === href || pathname.startsWith(`${href}/`);
  const initial = account.displayName.charAt(0).toUpperCase();

  return (
    <aside className="bg-bg border-bg-3 flex w-[232px] flex-shrink-0 flex-col border-r px-3.5 py-5">
      <div className="px-1.5 pb-[18px]">
        <Logo size={22} />
      </div>

      {/* Account switcher */}
      <div className="bg-bg-2 border-line mb-4 flex items-center gap-2.5 rounded-lg border px-2.5 py-2">
        <span
          className="flex h-6 w-6 items-center justify-center rounded-md text-xs font-semibold"
          style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}
        >
          {initial}
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-[12.5px] font-medium">{account.handle}</div>
          <div className="mono text-text-3 text-[10px]">3 IG · 1 user</div>
        </div>
        <span className="text-text-3">⌄</span>
      </div>

      <span className="mono tracked text-text-3 mb-1.5 px-2 text-[9.5px]">WORKSPACE</span>
      {WORKSPACE_NAV.map((it) => (
        <NavItem key={it.key} item={it} active={isActive(it.href)} />
      ))}

      <span className="mono tracked text-text-3 mb-1.5 mt-3.5 px-2 text-[9.5px]">SYSTEM</span>
      {SYSTEM_NAV.map((it) => (
        <NavItem key={it.key} item={it} active={isActive(it.href)} />
      ))}

      {/* Usage card */}
      <div className="bg-bg-2 border-line mt-auto rounded-[10px] border p-3">
        <span className="mono tracked text-text-3 text-[9.5px]">USAGE THIS MONTH</span>
        <div className="mb-1.5 mt-2 flex justify-between text-xs">
          <span className="text-text-2">Auto-DMs</span>
          <span className="mono">
            <span>892</span>
            <span className="text-text-3">/2,500</span>
          </span>
        </div>
        <Meter value={0.36} trackClassName="mb-3" />
        <button className="bg-bg-3 text-text border-line-2 w-full cursor-pointer rounded-md border px-2.5 py-[7px] text-xs">
          Upgrade plan
        </button>
      </div>
    </aside>
  );
}
