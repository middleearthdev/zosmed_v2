import type { ReactNode } from 'react';
import { Avatar, cn } from '@zosmed/ui';

export interface ConvBubbleProps {
  side: 'them' | 'us';
  ai?: boolean;
  name?: string;
  time: string;
  text: ReactNode;
  source?: string;
  tag?: string;
  suggested?: boolean;
}

/** Conversation message bubble — outgoing (lime / dashed-suggested) or incoming. */
export function ConvBubble({ side, ai, name, time, text, source, tag, suggested }: ConvBubbleProps) {
  const us = side === 'us';
  const solidLime = us && !suggested;
  return (
    <div className={cn('flex flex-col gap-1', us ? 'items-end' : 'items-start')}>
      {source ? <span className="mono text-text-3 text-[10px]">via {source}</span> : null}
      <div className="flex max-w-[70%] items-end gap-2">
        {!us ? <Avatar name={(name ?? '').slice(0, 2).toUpperCase()} color="var(--zz-pink)" size={24} /> : null}
        <div
          className="rounded-xl px-3.5 py-2.5 text-[13.5px] leading-normal"
          style={{
            background: solidLime ? 'var(--zz-lime)' : 'var(--zz-bg-3)',
            color: solidLime ? 'var(--zz-bg)' : 'var(--zz-text)',
            border: suggested ? '1px dashed var(--zz-lime)' : 'none',
          }}
        >
          {ai ? (
            <span
              className="mono mb-1 block text-[9px]"
              style={{ opacity: solidLime ? 0.6 : 1, color: solidLime ? 'var(--zz-bg)' : 'var(--zz-blue)' }}
            >
              ● AI · zosmed{tag ? ` · ${tag}` : ''}
              {suggested ? ' · suggested' : ''}
            </span>
          ) : null}
          <div>{text}</div>
        </div>
      </div>
      <div className="text-text-3 flex items-center gap-2 text-[10px]">
        <span className="mono">{time}</span>
        {suggested ? (
          <>
            <span className="text-lime">Approve &amp; send</span>
            <span>·</span>
            <span>Edit</span>
            <span>·</span>
            <span>Reject</span>
          </>
        ) : null}
      </div>
    </div>
  );
}
