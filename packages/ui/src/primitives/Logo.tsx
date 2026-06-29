import type { CSSProperties } from 'react';

export interface LogoProps {
  size?: number;
  theme?: 'dark' | 'light';
  showWord?: boolean;
}

const LIME = 'var(--zz-lime)';

export function Logo({ size = 22, theme = 'dark', showWord = true }: LogoProps) {
  const fg = theme === 'dark' ? 'var(--zz-text)' : 'var(--zb-text)';
  const bubbleBg = theme === 'dark' ? 'var(--zz-bg)' : 'var(--zb-bg)';
  const wordStyle: CSSProperties = {
    fontFamily: 'var(--font-sans)',
    fontWeight: 600,
    fontSize: size * 0.78,
    letterSpacing: '-0.02em',
    color: fg,
  };
  return (
    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
      <svg width={size} height={size} viewBox="0 0 48 48" fill="none" aria-hidden>
        <circle cx="24" cy="6" r="4" fill={LIME} />
        <rect x="22.4" y="9" width="3.2" height="6" rx="1.6" fill={LIME} />
        <path
          d="M14 13 H34 a8 8 0 0 1 8 8 v10 a8 8 0 0 1 -8 8 H20 l-8 7 v-7 a2 2 0 0 1 0 0 V21 a8 8 0 0 1 8 -8 Z"
          fill={LIME}
        />
        <rect x="16" y="20" width="20" height="12" rx="6" fill={bubbleBg} />
        <circle cx="21.5" cy="26" r="2.6" fill={LIME} />
        <circle cx="30.5" cy="26" r="2.6" fill={LIME} />
      </svg>
      {showWord && <span style={wordStyle}>zosmed</span>}
    </div>
  );
}
