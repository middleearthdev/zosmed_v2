import { cn } from '../lib/cn';

export interface MeterProps {
  /** 0..1 fill ratio (clamped). */
  value: number;
  color?: string;
  height?: number;
  /** Track classes — defaults to bg-3; pass e.g. 'bg-line' to match darker tracks. */
  trackClassName?: string;
  className?: string;
}

/** Single progress/quota bar. Reused by Gauge, workflow progress, keyword bars. */
export function Meter({ value, color = 'var(--zz-lime)', height = 4, trackClassName, className }: MeterProps) {
  const pct = Math.max(0, Math.min(1, value)) * 100;
  return (
    <div className={cn('bg-bg-3 overflow-hidden rounded-full', trackClassName)} style={{ height }}>
      <div className={cn('rounded-full', className)} style={{ width: `${pct}%`, height, background: color }} />
    </div>
  );
}
