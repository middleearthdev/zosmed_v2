'use client';

/**
 * Runs screen (ADR-004 F6) — client for selection/interaction over the
 * server-fetched `RunSummary[]` (no re-fetch here; KPIs/rate chart are
 * derived client-side from real data, never fabricated, CLAUDE.md §4b).
 */
import { useMemo, useState } from 'react';
import { Card, Dot, Pill } from '@zosmed/ui';
import type { RunStatus, RunSummary } from '@zosmed/types';
import { RunRateChart, type RunRateBar } from '../../_components/RunRateChart';

const STATUS_TONE: Record<RunStatus, 'lime' | 'pink' | 'warn' | 'neutral'> = {
  success: 'lime',
  failed: 'pink',
  skipped: 'warn',
  queued: 'neutral',
  running: 'neutral',
};
const RUN_FILTERS: { key: 'all' | RunStatus; label: string }[] = [
  { key: 'all', label: 'Semua' },
  { key: 'success', label: 'Sukses' },
  { key: 'failed', label: 'Gagal' },
  { key: 'skipped', label: 'Dilewati' },
];
const GRID = 'grid-cols-[90px_1.2fr_0.7fr_0.6fr_90px]';

function formatDuration(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms}ms`;
}

/** Bucket runs into 24 hourly buckets (local time) from the fetched sample — best-effort, not a guaranteed full-24h dataset. */
function buildRunRateBars(runs: RunSummary[]): RunRateBar[] {
  const bars: RunRateBar[] = Array.from({ length: 24 }, () => ({ success: 0, failed: 0, skipped: 0 }));
  for (const r of runs) {
    const hour = new Date(r.at).getHours();
    const bucket = bars[hour];
    if (!bucket) continue;
    if (r.status === 'success') bucket.success += 1;
    else if (r.status === 'failed') bucket.failed += 1;
    else if (r.status === 'skipped') bucket.skipped += 1;
  }
  return bars;
}

export function RunsClient({ runs }: { runs: RunSummary[] }) {
  const [filter, setFilter] = useState<'all' | RunStatus>('all');
  const [selectedId, setSelectedId] = useState<string | null>(runs[0]?.id ?? null);

  const filtered = useMemo(() => (filter === 'all' ? runs : runs.filter((r) => r.status === filter)), [runs, filter]);
  const selected = useMemo(() => runs.find((r) => r.id === selectedId) ?? filtered[0] ?? null, [runs, selectedId, filtered]);
  const rateBars = useMemo(() => buildRunRateBars(runs), [runs]);

  const totalRuns = runs.length;
  const successCount = runs.filter((r) => r.status === 'success').length;
  const failedCount = runs.filter((r) => r.status === 'failed').length;
  const successRate = totalRuns > 0 ? ((successCount / totalRuns) * 100).toFixed(1) : '—';
  const avgDurationMs = totalRuns > 0 ? Math.round(runs.reduce((sum, r) => sum + r.durationMs, 0) / totalRuns) : 0;

  const kpis = [
    { label: `RUNS · ${totalRuns} TERBARU`, value: String(totalRuns), color: 'var(--zz-lime)' },
    { label: 'SUCCESS RATE', value: totalRuns > 0 ? `${successRate}%` : '—', color: 'var(--zz-lime)' },
    { label: 'DURASI RATA-RATA', value: totalRuns > 0 ? formatDuration(avgDurationMs) : '—', color: 'var(--zz-lime)' },
    { label: 'GAGAL', value: String(failedCount), color: failedCount > 0 ? 'var(--zz-pink)' : 'var(--zz-text-2)' },
  ];

  if (runs.length === 0) {
    return (
      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <Card className="flex flex-col items-center gap-2 py-14 text-center">
          <p className="text-text m-0 text-sm font-medium">Belum ada run tercatat</p>
          <p className="text-text-3 m-0 mt-1 text-xs">
            Run muncul di sini setelah workflow `live` kamu ter-trigger oleh komentar/DM masuk.
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="zz-scroll flex-1 overflow-y-auto p-6">
      {/* KPI strip */}
      <div className="mb-[18px] grid grid-cols-4 gap-2.5">
        {kpis.map((k) => (
          <div key={k.label} className="bg-bg-2 border-line rounded-[10px] border p-3.5">
            <div className="mono tracked text-text-3 text-[9.5px]">{k.label}</div>
            <div className="mono mt-1 text-[22px] font-medium" style={{ color: k.color }}>
              {k.value}
            </div>
          </div>
        ))}
      </div>

      {/* Run rate timeline */}
      <Card className="mb-3.5 p-[18px]">
        <div className="mb-3 flex items-center justify-between">
          <span className="mono tracked text-text-3 text-[9.5px]">RUN RATE · PER JAM (SAMPEL TERAMBIL)</span>
          <div className="mono flex gap-3 text-[11px]">
            <span className="inline-flex items-center gap-1.5">
              <Dot color="var(--zz-lime)" /> sukses
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Dot color="var(--zz-warn)" /> dilewati
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Dot color="var(--zz-pink)" /> gagal
            </span>
          </div>
        </div>
        <RunRateChart bars={rateBars} />
      </Card>

      <div className="grid gap-3.5" style={{ gridTemplateColumns: '1.5fr 1fr' }}>
        {/* Run list */}
        <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
          <div className="border-line flex items-center justify-between border-b px-4 py-3.5">
            <span className="text-sm font-medium">Run terbaru</span>
            <div className="flex gap-1.5">
              {RUN_FILTERS.map((f) => (
                <button
                  key={f.key}
                  type="button"
                  onClick={() => setFilter(f.key)}
                  className={`mono rounded-full px-2.5 py-1 text-[10.5px] ${filter === f.key ? 'bg-bg-3 text-text' : 'text-text-3'}`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>
          <div className={`grid ${GRID} border-line border-b px-4 py-2`} style={{ background: '#0d0d0d' }}>
            {['WORKFLOW', 'TRIGGER', 'DURASI', 'LANGKAH', 'STATUS'].map((h) => (
              <span key={h} className="mono tracked text-text-3 text-[9.5px]">
                {h}
              </span>
            ))}
          </div>
          {filtered.map((r) => {
            const total = r.steps.length;
            return (
              <button
                key={r.id}
                type="button"
                onClick={() => setSelectedId(r.id)}
                className={`grid ${GRID} border-bg-3 w-full items-center gap-2 border-b px-4 py-3 text-left last:border-b-0`}
                style={{
                  background: r.id === selected?.id ? 'var(--zz-bg-3)' : 'transparent',
                  borderLeft: r.id === selected?.id ? '2px solid var(--zz-lime)' : '2px solid transparent',
                }}
              >
                <span className="mono text-text-2 truncate text-[11.5px]">{r.workflowName || '—'}</span>
                <div className="min-w-0">
                  <div className="truncate text-[12.5px]">{r.triggerSummary}</div>
                  <div className="mono text-text-3 text-[10.5px]">{new Date(r.at).toLocaleString('id-ID')}</div>
                </div>
                <span className="mono tnum text-xs">{formatDuration(r.durationMs)}</span>
                <span className="mono text-text-2 text-[11.5px]">{total > 0 ? `${total} langkah` : '—'}</span>
                <Pill tone={STATUS_TONE[r.status]}>{r.status}</Pill>
              </button>
            );
          })}
        </div>

        {/* Run detail */}
        <Card className="p-[18px]">
          {selected ? (
            <>
              <div className="mb-3.5 flex items-center justify-between">
                <div>
                  <div className="mono tracked text-text-3 text-[9.5px]">DETAIL RUN</div>
                  <div className="mono mt-0.5 text-sm">{selected.id.slice(0, 8)}</div>
                </div>
                <Pill tone={STATUS_TONE[selected.status]}>{selected.status}</Pill>
              </div>

              <div className="mb-4 grid grid-cols-2 gap-2 text-[11.5px]">
                {(
                  [
                    ['Trigger', selected.triggerSummary],
                    ['Waktu', new Date(selected.at).toLocaleString('id-ID')],
                    ['Durasi', formatDuration(selected.durationMs)],
                    ['Workflow', selected.workflowName || '—'],
                  ] as [string, string][]
                ).map(([k, v]) => (
                  <div key={k} className="flex justify-between py-[5px]" style={{ borderBottom: '1px solid #1a1a1d' }}>
                    <span className="text-text-3">{k}</span>
                    <span className="mono truncate">{v}</span>
                  </div>
                ))}
              </div>

              <div className="mono tracked text-text-3 mb-2.5 text-[9.5px]">STEP TIMELINE</div>
              {selected.steps.length === 0 ? (
                <p className="text-text-3 text-xs">Tidak ada langkah tercatat untuk run ini.</p>
              ) : (
                <div className="relative">
                  <div className="bg-line absolute" style={{ left: 9, top: 14, bottom: 14, width: 1 }} />
                  {selected.steps.map((s, i) => (
                    <div key={i} className="relative pb-3 pl-7">
                      <span
                        className="absolute rounded-full"
                        style={{
                          left: 5,
                          top: 6,
                          width: 10,
                          height: 10,
                          background: STEP_COLOR[s.kind],
                          border: '2px solid var(--zz-bg-2)',
                        }}
                      />
                      <div className="mb-[3px] flex items-center justify-between">
                        <span className="mono tracked text-[9px]" style={{ color: STEP_COLOR[s.kind] }}>
                          {s.kind.toUpperCase()} · {s.nodeKey.slice(0, 8)}
                        </span>
                        <span className="mono text-text-3 text-[10px]">{s.status}</span>
                      </div>
                      <div className="mb-[3px] text-[12.5px]">{s.detail}</div>
                    </div>
                  ))}
                </div>
              )}
            </>
          ) : (
            <p className="text-text-3 text-xs">Pilih run di daftar kiri untuk lihat detailnya.</p>
          )}
        </Card>
      </div>
    </div>
  );
}

const STEP_COLOR: Record<'trigger' | 'filter' | 'action', string> = {
  trigger: 'var(--zz-lime)',
  filter: 'var(--zz-warn)',
  action: 'var(--zz-pink)',
};
