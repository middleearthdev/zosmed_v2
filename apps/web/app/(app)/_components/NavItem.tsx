import Link from 'next/link';
import { cn } from '@zosmed/ui';
import type { NavItemDef } from './nav';

export function NavItem({ item, active }: { item: NavItemDef; active: boolean }) {
  return (
    <Link
      href={item.href}
      className={cn(
        'mb-0.5 flex items-center gap-2.5 rounded-md py-1.5 pr-2.5 text-[13px]',
        active ? 'bg-bg-3 text-text' : 'text-text-2',
      )}
      style={{
        borderLeft: active ? '2px solid var(--zz-lime)' : '2px solid transparent',
        paddingLeft: 8,
      }}
    >
      <span style={{ color: active ? 'var(--zz-lime)' : 'var(--zz-text-3)' }}>{item.icon}</span>
      <span className="flex-1">{item.label}</span>
      {item.badge ? (
        <span className="mono bg-bg-3 border-line text-text-2 rounded-full border px-1.5 text-[10px]">
          {item.badge}
        </span>
      ) : null}
    </Link>
  );
}
