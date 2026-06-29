import type { CSSProperties } from 'react';

export interface PlaceholderProps {
  label?: string;
  height?: number;
  theme?: 'dark' | 'light';
  style?: CSSProperties;
}

/** Striped image placeholder used by artboards before real media is wired. */
export function Placeholder({ label = 'image', height = 120, theme = 'dark', style }: PlaceholderProps) {
  const dark = theme === 'dark';
  return (
    <div
      style={{
        height,
        width: '100%',
        backgroundImage: `repeating-linear-gradient(135deg, transparent 0 10px, ${
          dark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.05)'
        } 10px 11px)`,
        border: `1px dashed ${dark ? 'var(--zz-line-2)' : 'var(--zb-line-2)'}`,
        color: dark ? 'var(--zz-text-3)' : 'var(--zb-text-3)',
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        letterSpacing: '0.04em',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: 6,
        textTransform: 'uppercase',
        ...style,
      }}
    >
      {label}
    </div>
  );
}
