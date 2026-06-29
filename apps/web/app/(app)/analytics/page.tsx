import Link from 'next/link';
import { Button, I, Pill } from '@zosmed/ui';
import { getAnalytics } from '@/lib/mock/api';
import type { FunnelStep, WorkflowRevRow, IntentRow, LeaderboardEntry } from '@/lib/mock/analytics';
import { PageHeader } from '../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../_components/PageHeaderBreadcrumb';
import { BigChart } from './_components/BigChart';
import { Heatmap } from './_components/Heatmap';

const PERIODS = ['7d', '30d', '90d', 'YTD', 'All'];

export default async function AnalyticsPage() {
  const data = await getAnalytics();

  return (
    <>
      <PageHeader>
        <PageHeaderBreadcrumb
          crumbs={[{ label: 'Analytics' }, { label: 'Conversion funnel' }]}
        />
        <div className="flex gap-2">
          {/* Period selector */}
          <div
            className="border-line flex rounded-lg border p-[2px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            {PERIODS.map((t, i) => (
              <span
                key={t}
                className="mono cursor-pointer rounded-md px-3 py-[5px] text-[12px]"
                style={
                  i === 1
                    ? { background: 'var(--zz-bg-3)', color: 'var(--zz-text)' }
                    : { color: 'var(--zz-text-2)' }
                }
              >
                {t}
              </span>
            ))}
          </div>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Compare
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Export CSV
          </Button>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto px-6 pb-12 pt-6">
        {/* Heading row */}
        <div className="mb-6 flex items-end justify-between">
          <div>
            <span className="mono tracked text-text-3 text-[10px]">{data.dateLabel}</span>
            <h1 className="mono mt-1.5 text-3xl font-medium tracking-tight">
              Performance overview
            </h1>
          </div>
          <div className="flex gap-2">
            <Pill tone="neutral">All workflows</Pill>
            <Pill tone="neutral">All accounts</Pill>
          </div>
        </div>

        {/* Big chart card */}
        <div
          className="border-line mb-3 rounded-xl border p-5"
          style={{ background: 'var(--zz-bg-2)' }}
        >
          <div className="mb-[18px] flex items-start justify-between">
            <div className="flex gap-7">
              {data.bigMetrics.map((m) => (
                <BigMetricBlock key={m.label} m={m} />
              ))}
            </div>
            {/* Legend */}
            <div className="flex gap-3 text-[11px]" style={{ color: 'var(--zz-text-2)' }}>
              {data.chartSeries.map((s, i) => (
                <span key={i} className="inline-flex items-center gap-1.5">
                  <span
                    className="inline-block h-[2px] w-2"
                    style={{ background: s.color }}
                  />
                  {['Comments', 'DMs', 'Leads'][i]}
                </span>
              ))}
            </div>
          </div>
          <BigChart series={data.chartSeries} />
        </div>

        {/* Mid row: funnel + workflows by revenue */}
        <div className="mb-3 grid gap-3" style={{ gridTemplateColumns: '1.4fr 1fr' }}>
          {/* Funnel */}
          <div
            className="border-line rounded-xl border p-5"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <h3 className="m-0 mb-4 text-base font-medium">Funnel comment-to-cash</h3>
            {data.funnel.map((step, i) => (
              <FunnelRow key={step.label} step={step} idx={i} />
            ))}
            {/* AI insight callout */}
            <div
              className="mt-3.5 flex items-center gap-2.5 rounded-lg p-3"
              style={{ background: 'var(--zz-bg-3)' }}
            >
              <span
                className="inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-[6px]"
                style={{
                  background: 'oklch(0.85 0.16 75 / 0.18)',
                  color: 'var(--zz-warn)',
                }}
              >
                <I.sparkle />
              </span>
              <div className="flex-1 text-[12.5px]">
                <span className="text-text">AI insight: </span>
                <span className="text-text-2">
                  Drop-off terbesar di AI conversation → leads. Coba revisi prompt closing untuk
                  naikin CVR ~5pt.
                </span>
              </div>
              <span className="text-lime flex items-center gap-1 text-[11px]">
                Apply suggestion <I.arrow />
              </span>
            </div>
          </div>

          {/* Workflows by revenue */}
          <div
            className="border-line rounded-xl border p-5"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="mb-3.5 flex items-center justify-between">
              <h3 className="m-0 text-base font-medium">Workflows by revenue</h3>
              <span className="mono text-text-3 text-[10.5px]">30D</span>
            </div>
            {/* Table header */}
            <div
              className="mono tracked grid pb-2 text-[10px]"
              style={{
                gridTemplateColumns: '1.6fr 1fr 1fr 1fr',
                color: 'var(--zz-text-3)',
                borderBottom: '1px solid #1a1a1d',
              }}
            >
              <span>WORKFLOW</span>
              <span style={{ textAlign: 'right' }}>RUNS</span>
              <span style={{ textAlign: 'right' }}>CVR</span>
              <span style={{ textAlign: 'right' }}>REVENUE</span>
            </div>
            {data.workflowsByRevenue.map((row) => (
              <WorkflowRevRow key={row.name} row={row} />
            ))}
          </div>
        </div>

        {/* Bottom: heatmap, top intents, leaderboard */}
        <div className="grid grid-cols-3 gap-3">
          {/* Hourly heatmap */}
          <div
            className="border-line rounded-xl border p-5"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <h3 className="m-0 mb-3.5 text-sm font-medium">Hourly heatmap (komentar)</h3>
            <Heatmap cells={data.heatmap} />
            <div
              className="mono mt-2 flex justify-between text-[10px]"
              style={{ color: 'var(--zz-text-3)' }}
            >
              <span>00</span>
              <span>06</span>
              <span>12</span>
              <span>18</span>
              <span>23</span>
            </div>
          </div>

          {/* Top intents */}
          <div
            className="border-line rounded-xl border p-5"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <h3 className="m-0 mb-3.5 text-sm font-medium">Top intents (AI)</h3>
            {data.topIntents.map((intent) => (
              <IntentItem key={intent.label} intent={intent} />
            ))}
          </div>

          {/* Leaderboard */}
          <div
            className="border-line rounded-xl border p-5"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <h3 className="m-0 mb-3.5 text-sm font-medium">Leaderboard akun</h3>
            {data.leaderboard.map((entry, i) => (
              <LeaderboardRow key={entry.handle} entry={entry} rank={i + 1} first={i === 0} />
            ))}
          </div>
        </div>
      </div>
    </>
  );
}

// ── Sub-components ────────────────────────────────────────────────────────────

function BigMetricBlock({ m }: { m: { label: string; value: string; delta: string; color: string } }) {
  return (
    <div>
      <div className="mono tracked text-text-3 text-[9.5px]">{m.label}</div>
      <div className="mono tnum mt-1 text-[26px] font-medium" style={{ letterSpacing: '-0.02em' }}>
        {m.value}
      </div>
      <div className="mono mt-0.5 text-[11px]" style={{ color: m.color }}>
        ↑ {m.delta}
      </div>
    </div>
  );
}

function FunnelRow({ step, idx }: { step: FunnelStep; idx: number }) {
  return (
    <div className="mb-2 flex items-center gap-2.5">
      <span className="mono text-text-3 w-[22px] text-[10.5px]">
        {String(idx + 1).padStart(2, '0')}
      </span>
      <span className="flex-1 text-[13px]">{step.label}</span>
      <span className="mono tnum w-20 text-right text-[12px]">
        {step.n.toLocaleString('id-ID')}
      </span>
      <span className="mono tnum text-text-3 w-11 text-right text-[11px]">
        {(step.p * 100).toFixed(1)}%
      </span>
      {/* Bar */}
      <div
        className="relative w-[280px] rounded-[4px]"
        style={{ height: 22, background: 'var(--zz-bg-3)' }}
      >
        <div
          className="rounded-[4px]"
          style={{
            width: `${step.p * 100}%`,
            height: 22,
            background: step.color,
            opacity: 0.9,
          }}
        />
      </div>
      <span
        className="mono w-12 text-right text-[10.5px]"
        style={{ color: step.drop ? 'var(--zz-pink)' : 'var(--zz-text-3)' }}
      >
        {step.drop ?? ''}
      </span>
    </div>
  );
}

function WorkflowRevRow({ row }: { row: WorkflowRevRow }) {
  return (
    <Link
      href={`/analytics/${row.name}`}
      className="relative block py-[10px] transition-colors hover:bg-white/[0.02]"
      style={{ borderBottom: '1px solid #1a1a1d' }}
    >
      <div
        className="relative grid items-center text-[12.5px]"
        style={{ gridTemplateColumns: '1.6fr 1fr 1fr 1fr', zIndex: 1 }}
      >
        <span className="mono">{row.name}</span>
        <span className="mono tnum text-right">{row.runs.toLocaleString('id-ID')}</span>
        <span className="mono tnum text-right">{row.cvr}</span>
        <span className="mono tnum text-right" style={{ color: 'var(--zz-lime)' }}>
          {row.revenue}
        </span>
      </div>
      {/* Revenue bar underline */}
      <div
        className="absolute bottom-0 left-0 h-[2px] rounded-full"
        style={{ width: `${row.barRatio * 100}%`, background: 'var(--zz-lime)', opacity: 0.5 }}
      />
    </Link>
  );
}

function IntentItem({ intent }: { intent: IntentRow }) {
  return (
    <div className="mb-1.5 flex items-center gap-2">
      <span
        className="h-1.5 w-1.5 flex-shrink-0 rounded-full"
        style={{ background: intent.color }}
      />
      <span className="mono text-text-2 flex-1 text-[11.5px]">{intent.label}</span>
      <span className="mono tnum text-[11.5px]">{intent.count.toLocaleString('id-ID')}</span>
    </div>
  );
}

function LeaderboardRow({
  entry,
  rank,
  first,
}: {
  entry: LeaderboardEntry;
  rank: number;
  first: boolean;
}) {
  return (
    <div
      className="flex items-center gap-2.5 py-2"
      style={{ borderTop: first ? 'none' : '1px solid #1a1a1d' }}
    >
      <span className="mono text-text-3 w-3.5 text-[10.5px]">{rank}</span>
      <span
        className="inline-flex h-7 w-7 items-center justify-center rounded-[6px] text-[11px] font-semibold"
        style={{ background: entry.color, color: 'var(--zz-bg)' }}
      >
        {entry.initials}
      </span>
      <span className="mono flex-1 text-[12.5px]">@{entry.handle}</span>
      <span className="mono tnum text-[12px]" style={{ color: 'var(--zz-lime)' }}>
        {entry.revenue}
      </span>
    </div>
  );
}
