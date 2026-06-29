import type { CSSProperties, ReactNode } from 'react';

export type PillTone = 'lime' | 'neutral' | 'warn' | 'pink' | 'blue';

export interface PillProps {
  children: ReactNode;
  tone?: PillTone;
  style?: CSSProperties;
}

const tones: Record<PillTone, { bg: string; fg: string; border: string }> = {
  lime: { bg: 'var(--zz-lime-soft)', fg: 'var(--zz-lime)', border: 'oklch(0.9 0.2 130 / 0.3)' },
  neutral: { bg: '#1a1a1d', fg: 'var(--zz-text-2)', border: '#2a2a2e' },
  warn: { bg: 'oklch(0.85 0.16 75 / 0.12)', fg: 'var(--zz-warn)', border: 'oklch(0.85 0.16 75 / 0.3)' },
  pink: { bg: 'oklch(0.78 0.2 0 / 0.12)', fg: 'var(--zz-pink)', border: 'oklch(0.78 0.2 0 / 0.3)' },
  blue: { bg: 'oklch(0.78 0.16 240 / 0.12)', fg: 'var(--zz-blue)', border: 'oklch(0.78 0.16 240 / 0.3)' },
};

/** Status pill. Note: "● LIVE" here means workflow-running, never IG Live (§9). */
export function Pill({ children, tone = 'lime', style }: PillProps) {
  const t = tones[tone];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        background: t.bg,
        color: t.fg,
        border: `1px solid ${t.border}`,
        padding: '3px 8px',
        borderRadius: 999,
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        letterSpacing: '0.04em',
        ...style,
      }}
    >
      {children}
    </span>
  );
}
