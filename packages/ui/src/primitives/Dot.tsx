export interface DotProps {
  color?: string;
  size?: number;
}

export function Dot({ color = 'var(--zz-lime)', size = 6 }: DotProps) {
  return (
    <span
      style={{ width: size, height: size, background: color, borderRadius: 999, display: 'inline-block' }}
    />
  );
}
