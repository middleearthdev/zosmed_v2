export interface SparklineProps {
  data: number[];
  color?: string;
  height?: number;
}

/** Minimal area sparkline (ported from the design artboard). */
export function Sparkline({ data, color = 'var(--zz-lime)', height = 36 }: SparklineProps) {
  const w = 200;
  const h = height;
  const max = Math.max(...data);
  const min = Math.min(...data);
  const span = max - min || 1;
  const step = w / Math.max(1, data.length - 1);
  const pts = data.map((d, i) => [i * step, h - ((d - min) / span) * h] as const);
  const path = pts.map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
  const area = `${path} L${w},${h} L0,${h} Z`;
  return (
    <svg
      width="100%"
      height={h}
      viewBox={`0 0 ${w} ${h}`}
      preserveAspectRatio="none"
      style={{ display: 'block', marginTop: 8 }}
    >
      <path d={area} fill={color} opacity="0.12" />
      <path d={path} stroke={color} strokeWidth="1.5" fill="none" />
    </svg>
  );
}
