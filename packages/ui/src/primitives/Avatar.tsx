export interface AvatarProps {
  name?: string;
  color?: string;
  size?: number;
  theme?: 'dark' | 'light';
}

export function Avatar({ name = 'AB', color = '#2a2a2e', size = 28, theme = 'dark' }: AvatarProps) {
  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: 999,
        background: color,
        color: theme === 'dark' ? 'var(--zz-text)' : 'var(--zb-text)',
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontFamily: 'var(--font-mono)',
        fontSize: size * 0.38,
        fontWeight: 600,
        flexShrink: 0,
      }}
    >
      {name}
    </div>
  );
}
