/**
 * 7-day revenue line chart with gradient fill + dot-per-day + day labels.
 * Screen-local: only used by analytics/[workflowId]/page.tsx.
 * No Math.random() — data is precomputed in mock fixtures.
 */

interface RevenuePoint {
  day: string;
  value: number;
}

interface RevenueChartProps {
  points: RevenuePoint[];
  /** Highlighted index (tooltip day). */
  highlightIdx?: number;
}

const SVG_W = 920;
const SVG_H = 240;
const PAD_LEFT = 40;
const PAD_BOTTOM = 20;
const INNER_H = SVG_H - PAD_BOTTOM;
const MAX_VAL = 40; // jt
const Y_TICKS = [0, 10, 20, 30, 40];

function toXY(points: RevenuePoint[]) {
  return points.map((p, i) => {
    const x = PAD_LEFT + 20 + i * 130;
    const y = INNER_H - (p.value / MAX_VAL) * (INNER_H - 20);
    return { x, y, day: p.day, value: p.value };
  });
}

export function RevenueChart({ points, highlightIdx = 3 }: RevenueChartProps) {
  const mapped = toXY(points);
  const linePath = mapped.map((p, i) => `${i ? 'L' : 'M'}${p.x} ${p.y}`).join(' ');
  const first = mapped[0];
  const last = mapped[mapped.length - 1];
  if (!first || !last) return null;
  const areaPath = `${linePath} L${last.x} ${INNER_H} L${first.x} ${INNER_H} Z`;

  return (
    <svg viewBox={`0 0 ${SVG_W} ${SVG_H}`} width="100%" height={SVG_H} style={{ marginTop: 18 }}>
      <defs>
        <linearGradient id="rev-fill" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor="var(--zz-lime)" stopOpacity="0.32" />
          <stop offset="100%" stopColor="var(--zz-lime)" stopOpacity="0" />
        </linearGradient>
      </defs>

      {/* Horizontal grid lines */}
      {[0, 60, 120, 180].map((y) => (
        <line key={y} x1={PAD_LEFT} x2={SVG_W - 10} y1={y + 20} y2={y + 20} stroke="#1a1a1d" />
      ))}

      {/* Y-axis labels */}
      {Y_TICKS.map((v, i) => (
        <text
          key={v}
          x={PAD_LEFT - 8}
          y={INNER_H - i * 45 + 4}
          fontFamily="var(--font-mono)"
          fontSize={10}
          fill="var(--zz-text-3)"
          textAnchor="end"
        >
          {v}jt
        </text>
      ))}

      {/* Area fill */}
      <path d={areaPath} fill="url(#rev-fill)" />

      {/* Line */}
      <path d={linePath} fill="none" stroke="var(--zz-lime)" strokeWidth={2} />

      {/* Dots + x-labels + optional tooltip */}
      {mapped.map((p, i) => (
        <g key={i}>
          <circle
            cx={p.x}
            cy={p.y}
            r={4}
            fill="var(--zz-bg)"
            stroke="var(--zz-lime)"
            strokeWidth={2}
          />
          <text
            x={p.x}
            y={SVG_H - 2}
            fontFamily="var(--font-mono)"
            fontSize={10}
            fill="var(--zz-text-3)"
            textAnchor="middle"
          >
            {p.day}
          </text>
          {i === highlightIdx && (
            <g>
              <rect
                x={p.x - 32}
                y={p.y - 30}
                width={64}
                height={20}
                fill="var(--zz-bg)"
                stroke="var(--zz-lime)"
                rx={3}
              />
              <text
                x={p.x}
                y={p.y - 16}
                fontFamily="var(--font-mono)"
                fontSize={11}
                fill="var(--zz-lime)"
                textAnchor="middle"
              >
                Rp {p.value}jt
              </text>
            </g>
          )}
        </g>
      ))}
    </svg>
  );
}
