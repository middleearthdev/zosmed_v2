import type { RunRateBar } from '@/lib/mock/workflows';

/** Stacked run-rate bars over 24h (success / review / failed). SSR-safe — data is precomputed. */
export function RunRateChart({ bars }: { bars: RunRateBar[] }) {
  const baseline = 130;
  return (
    <svg viewBox="0 0 1240 140" width="100%" height="140">
      {bars.map((b, i) => {
        const x = 8 + i * 25.6;
        return (
          <g key={i}>
            <rect x={x} y={baseline - b.success} width={18} height={b.success} fill="var(--zz-lime)" rx={1} />
            {b.review > 0 ? (
              <rect x={x} y={baseline - b.success - b.review} width={18} height={b.review} fill="var(--zz-warn)" rx={1} />
            ) : null}
            {b.failed > 0 ? (
              <rect x={x} y={baseline - b.success - b.review - b.failed} width={18} height={b.failed} fill="var(--zz-pink)" rx={1} />
            ) : null}
          </g>
        );
      })}
      {[0, 6, 12, 18, 24].map((h, i) => (
        <text
          key={h}
          x={8 + (h / 24) * 1228}
          y={138}
          fontFamily="var(--font-mono)"
          fontSize="10"
          fill="var(--zz-text-3)"
          textAnchor={i === 0 ? 'start' : i === 4 ? 'end' : 'middle'}
        >
          {h.toString().padStart(2, '0')}:00
        </text>
      ))}
    </svg>
  );
}
