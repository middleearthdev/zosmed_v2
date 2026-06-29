/**
 * 7×24 intensity heatmap for Analytics overview (hourly comments by day).
 * Screen-local: only used by analytics/page.tsx.
 * Values pre-computed deterministically in mock data — no Math.random() in render.
 */
import type { HeatCell } from '@/lib/mock/analytics';

interface HeatmapProps {
  cells: HeatCell[];
}

export function Heatmap({ cells }: HeatmapProps) {
  const days = 7;
  const hours = 24;

  return (
    <div className="flex flex-col gap-[2px]">
      {Array.from({ length: days }, (_, d) => (
        <div key={d} className="grid gap-[2px]" style={{ gridTemplateColumns: `repeat(${hours}, 1fr)` }}>
          {Array.from({ length: hours }, (_, h) => {
            const cell = cells[d * hours + h];
            const opacity = cell ? (cell.v * 0.85).toFixed(2) : '0';
            return (
              <span
                key={h}
                className="block rounded-[2px]"
                style={{
                  height: 12,
                  background: `oklch(0.9 0.2 130 / ${opacity})`,
                }}
              />
            );
          })}
        </div>
      ))}
    </div>
  );
}
