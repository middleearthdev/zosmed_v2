import Link from 'next/link';
import { cn } from '@zosmed/ui';
import type { NavItemDef } from './nav';

export function NavItem({ item, active, collapsed }: { item: NavItemDef; active: boolean; collapsed?: boolean }) {
  return (
    <Link
      href={item.href}
      title={collapsed ? item.label : undefined}
      className={cn(
        'mb-0.5 flex items-center rounded-md py-1.5 text-[13px]',
        collapsed ? 'justify-center pr-0' : 'gap-2.5 pr-2.5',
        active ? 'bg-bg-3 text-text' : 'text-text-2',
      )}
      style={{
        borderLeft: active ? '2px solid var(--zz-lime)' : '2px solid transparent',
        paddingLeft: collapsed ? 0 : 8,
      }}
    >
      <span className="relative" style={{ color: active ? 'var(--zz-lime)' : 'var(--zz-text-3)' }}>
        {item.icon}
        {collapsed && item.badge ? (
          <span
            className="absolute -right-1.5 -top-1 h-1.5 w-1.5 rounded-full"
            style={{ background: 'var(--zz-lime)' }}
          />
        ) : null}
      </span>
      {collapsed ? null : (
        <>
          <span className="flex-1">{item.label}</span>
          {item.badge ? (
            <span className="mono bg-bg-3 border-line text-text-2 rounded-full border px-1.5 text-[10px]">{item.badge}</span>
          ) : null}
        </>
      )}
    </Link>
  );
}
