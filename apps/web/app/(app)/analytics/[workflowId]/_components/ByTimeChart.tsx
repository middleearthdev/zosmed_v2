/**
 * 24-bar "by time of day" chart for the analytics drilldown.
 * Screen-local: only used by analytics/[workflowId]/page.tsx.
 * Values precomputed in mock data — no Math.random() in render.
 */
import type { ByTimeBar } from '@/lib/mock/analytics';

interface ByTimeChartProps {
  bars: ByTimeBar[];
}

export function ByTimeChart({ bars }: ByTimeChartProps) {
  return (
    <div className="grid gap-[2px]" style={{ gridTemplateColumns: 'repeat(24, 1fr)' }}>
      {bars.map((b) => {
        const pct = Math.round(Math.min(1, Math.max(0, b.v)) * 100);
        return (
          <div
            key={b.hour}
            className="rounded-[2px]"
            style={{
              height: 80,
              background: `color-mix(in oklch, var(--zz-lime) ${pct}%, #0a0a0a)`,
            }}
          />
        );
      })}
    </div>
  );
}
