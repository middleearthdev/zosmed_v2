import { Avatar, I } from '@zosmed/ui';
import type { Account } from '@zosmed/types';

export function Topbar({ account }: { account: Account }) {
  return (
    <header className="border-bg-3 flex h-14 flex-shrink-0 items-center justify-between border-b px-6">
      <div className="text-text-2 flex items-center gap-3 text-[13px]">
        <span>Workspace</span>
        <span className="text-line-2">/</span>
        <span className="text-text">{account.handle}</span>
        <span
          className="mono ml-2 rounded text-[10px]"
          style={{
            padding: '2px 6px',
            background: 'var(--zz-lime-soft)',
            color: 'var(--zz-lime)',
            letterSpacing: '0.06em',
          }}
        >
          PRO
        </span>
      </div>
      <div className="flex items-center gap-3">
        <div className="bg-bg-2 border-line text-text-3 flex w-[280px] items-center gap-2 rounded-lg border px-2.5 py-1.5">
          <I.search />
          <span className="text-[13px]">Search workflows, posts, contacts…</span>
          <span className="mono bg-bg-3 border-line ml-auto rounded-sm border px-[5px] py-px text-[10px]">
            ⌘K
          </span>
        </div>
        <span className="bg-bg-2 border-line text-text-2 inline-flex h-8 w-8 items-center justify-center rounded-lg border">
          <I.cog />
        </span>
        <Avatar name="MR" color="#3a3a40" />
      </div>
    </header>
  );
}
