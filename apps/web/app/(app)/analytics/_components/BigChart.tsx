/**
 * Three-series area+line chart for the Analytics overview.
 * Uses precomputed deterministic points from mock data — no Math.random().
 * Screen-local: only used by analytics/page.tsx.
 */

interface ChartSeries {
  color: string;
  points: number[];
}

interface BigChartProps {
  series: ChartSeries[];
}

const W = 1300;
const H = 220;
const MAX = 200;
const DAYS = 28;

export function BigChart({ series }: BigChartProps) {
  return (
    <svg
      width="100%"
      height={H}
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="none"
      style={{ display: 'block' }}
    >
      {/* Grid lines */}
      {[0.25, 0.5, 0.75].map((p) => (
        <line key={p} x1={0} x2={W} y1={H * p} y2={H * p} stroke="#1a1a1d" />
      ))}

      {/* Series */}
      {series.map((s, idx) => {
        const pts = s.points.map((d, i) => [
          i * (W / (DAYS - 1)),
          H - (d / MAX) * H,
        ] as const);
        const path = pts
          .map(([x, y], k) => `${k ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`)
          .join(' ');
        const area = `${path} L${W},${H} L0,${H} Z`;
        return (
          <g key={idx}>
            <path d={area} fill={s.color} opacity={0.08} />
            <path d={path} stroke={s.color} strokeWidth="1.6" fill="none" />
          </g>
        );
      })}

      {/* Today marker (near right edge) */}
      <line
        x1={W - 30}
        x2={W - 30}
        y1={0}
        y2={H}
        stroke="var(--zz-lime)"
        strokeWidth="1"
        strokeDasharray="3 3"
        opacity="0.5"
      />
      <circle cx={W - 30} cy={48} r={4} fill="var(--zz-lime)" />
    </svg>
  );
}
