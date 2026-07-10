'use client';

import { useEffect, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { I, Logo, Meter } from '@zosmed/ui';
import type { Account } from '@zosmed/types';
import { logout } from '@/lib/auth';
import { SYSTEM_NAV, WORKSPACE_NAV } from './nav';
import { NavItem } from './NavItem';

const COLLAPSE_KEY = 'zz-sidebar-collapsed';

/**
 * True on the workflow builder route (/workflows/{id}) — but NOT its sibling
 * static pages (list, runs, comment-to-order, inspector). The builder needs
 * the extra horizontal room, so the rail auto-collapses there.
 */
function isBuilderRoutePath(pathname: string): boolean {
  return /^\/workflows\/(?!runs$|comment-to-order$|inspector$)[^/]+$/.test(pathname);
}

export function Sidebar({ account }: { account: Account }) {
  const pathname = usePathname();
  const router = useRouter();
  const isBuilder = isBuilderRoutePath(pathname);

  // userPref = the persisted, user-chosen state for normal pages.
  // collapsed = the actual rendered state (forced collapsed on the builder).
  const [userPref, setUserPref] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  // Restore persisted preference after hydration (avoids SSR mismatch).
  // Reading localStorage must happen post-mount, so this intentionally
  // sets state from an effect rather than a lazy initializer.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setUserPref(localStorage.getItem(COLLAPSE_KEY) === '1');
  }, []);

  // Route-driven: force-collapse on the builder, otherwise follow user pref.
  // Deps are booleans, so this only re-runs when entering/leaving the builder
  // or when the stored pref loads — a manual toggle while on the builder is
  // therefore preserved until the route changes. Kept as its own state (not a
  // computed value) so `toggle()` can flip `collapsed` independently while on
  // the builder route without touching the persisted `userPref`.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setCollapsed(isBuilder ? true : userPref);
  }, [isBuilder, userPref]);

  function toggle() {
    setCollapsed((c) => {
      const next = !c;
      // Only persist as the global preference off the builder — the builder's
      // collapse is route-driven, not a lasting user choice.
      if (!isBuilder) {
        setUserPref(next);
        localStorage.setItem(COLLAPSE_KEY, next ? '1' : '0');
      }
      return next;
    });
  }

  const isActive = (href: string) => pathname === href || pathname.startsWith(`${href}/`);
  const initial = account.displayName.charAt(0).toUpperCase();

  async function handleLogout() {
    await logout();
    router.push('/login');
    router.refresh();
  }

  return (
    <aside
      className="bg-bg border-bg-3 relative flex flex-shrink-0 flex-col border-r py-5 transition-[width] duration-200 ease-out"
      style={{ width: collapsed ? 64 : 232, paddingLeft: collapsed ? 8 : 14, paddingRight: collapsed ? 8 : 14 }}
    >
      {/* Collapse toggle */}
      <button
        type="button"
        onClick={toggle}
        title={collapsed ? 'Perlebar sidebar' : 'Perkecil sidebar'}
        className="bg-bg-2 border-line text-text-3 hover:text-text absolute -right-3 top-6 z-10 flex h-6 w-6 items-center justify-center rounded-full border transition-colors"
      >
        <span className="text-xs" style={{ transform: collapsed ? 'none' : 'rotate(180deg)' }}>
          ›
        </span>
      </button>

      <div className={collapsed ? 'flex justify-center pb-[18px]' : 'px-1.5 pb-[18px]'}>
        <Logo size={22} showWord={!collapsed} />
      </div>

      {/* Account switcher */}
      {collapsed ? (
        <div className="mb-4 flex justify-center">
          <span
            className="flex h-7 w-7 items-center justify-center rounded-md text-xs font-semibold"
            style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}
            title={account.handle}
          >
            {initial}
          </span>
        </div>
      ) : (
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
      )}

      {!collapsed ? <span className="mono tracked text-text-3 mb-1.5 px-2 text-[9.5px]">WORKSPACE</span> : null}
      {WORKSPACE_NAV.map((it) => (
        <NavItem key={it.key} item={it} active={isActive(it.href)} collapsed={collapsed} />
      ))}

      {!collapsed ? <span className="mono tracked text-text-3 mb-1.5 mt-3.5 px-2 text-[9.5px]">SYSTEM</span> : <div className="my-2" />}
      {SYSTEM_NAV.map((it) => (
        <NavItem key={it.key} item={it} active={isActive(it.href)} collapsed={collapsed} />
      ))}

      {/* Usage card — hidden when collapsed to keep the rail slim */}
      {collapsed ? (
        <div className="mt-auto" />
      ) : (
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
      )}

      <button
        type="button"
        onClick={handleLogout}
        title={collapsed ? 'Keluar' : undefined}
        className={`text-text-3 hover:text-text mt-3 flex w-full cursor-pointer items-center rounded-md py-2 text-left text-xs transition-colors ${collapsed ? 'justify-center px-0' : 'gap-2 px-2'}`}
      >
        <I.arrow style={{ transform: 'rotate(180deg)' }} />
        {collapsed ? null : 'Keluar'}
      </button>
    </aside>
  );
}
