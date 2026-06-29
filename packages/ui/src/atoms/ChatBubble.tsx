import type { ReactNode } from 'react';
import { cn } from '../lib/cn';

export interface ChatBubbleProps {
  side: 'them' | 'us';
  text: ReactNode;
  ai?: boolean;
  /** Outgoing bubble background (default lime). Kits override with their accent. */
  accent?: string;
  className?: string;
}

/** Simple chat bubble for previews/playgrounds (AI Studio, Kits, Landing, Onboarding). */
export function ChatBubble({ side, text, ai, accent = 'var(--zz-lime)', className }: ChatBubbleProps) {
  const us = side === 'us';
  return (
    <div className={cn('flex', us ? 'justify-end' : 'justify-start', className)}>
      <div
        className="rounded-[10px] px-3 py-2 text-[13px] leading-normal"
        style={{ maxWidth: '80%', background: us ? accent : 'var(--zz-bg-3)', color: us ? 'var(--zz-bg)' : 'var(--zz-text)' }}
      >
        {ai ? (
          <span className="mono mb-[3px] block text-[9px]" style={{ opacity: 0.7 }}>
            ● AI · zosmed
          </span>
        ) : null}
        {text}
      </div>
    </div>
  );
}
