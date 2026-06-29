import { notFound } from 'next/navigation';
import { Button } from '@zosmed/ui';
import { getAnalyticsDrilldown } from '@/lib/mock/api';
import type { StepConversion, PostRevRow, KeywordIntentBar } from '@/lib/mock/analytics';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';
import { RevenueChart } from './_components/RevenueChart';
import { ByTimeChart } from './_components/ByTimeChart';

interface Props {
  params: Promise<{ workflowId: string }>;
}

const METRIC_TABS = ['Revenue', 'Leads', 'Replies', 'Cost'];

export default async function AnalyticsDrilldownPage({ params }: Props) {
  const { workflowId } = await params;
  const data = await getAnalyticsDrilldown(workflowId);

  if (!data) notFound();

  return (
    <>
      <PageHeader>
        <PageHeaderBreadcrumb
          crumbs={[
            { label: 'Analytics', href: '/analytics' },
            { label: 'Workflow' },
            { label: data.workflowLabel },
          ]}
        />
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Last 7d ▾
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Compare
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Export
          </Button>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto px-6 pb-12 pt-6">
        {/* Hero row: revenue card + step conversion */}
        <div className="mb-3.5 grid gap-3.5" style={{ gridTemplateColumns: '2fr 1fr' }}>
          {/* Revenue card */}
          <div
            className="border-line rounded-xl border p-[22px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="flex items-start justify-between">
              <div>
                <div className="mono tracked text-text-3 text-[9.5px]">
                  REVENUE INFLUENCED · 7D
                </div>
                <div className="mt-1.5 flex items-baseline gap-2">
                  <span
                    className="mono tnum text-[42px] font-medium"
                    style={{ letterSpacing: '-0.02em' }}
                  >
                    {data.revenue}
                  </span>
                  <span className="mono text-sm" style={{ color: 'var(--zz-lime)' }}>
                    {data.revenueDelta}
                  </span>
                </div>
                <div className="mono text-text-3 mt-1 text-[11px]">{data.revenueVsPrior}</div>
              </div>
              {/* Metric tabs */}
              <div className="flex gap-1.5">
                {METRIC_TABS.map((t, i) => (
                  <span
                    key={t}
                    className="mono cursor-pointer rounded-md px-2.5 py-[5px] text-[11px]"
                    style={
                      i === 0
                        ? {
                            background: 'var(--zz-bg-3)',
                            color: 'var(--zz-text)',
                            border: '1px solid var(--zz-line)',
                          }
                        : { color: 'var(--zz-text-3)' }
                    }
                  >
                    {t}
                  </span>
                ))}
              </div>
            </div>
            <RevenueChart points={data.revenuePoints} highlightIdx={3} />
          </div>

          {/* Step conversion */}
          <div
            className="border-line rounded-xl border p-[22px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="mono tracked text-text-3 mb-3.5 text-[9.5px]">
              STEP CONVERSION
            </div>
            {data.stepConversion.map((step, i) => (
              <StepBar key={step.label} step={step} idx={i} prev={data.stepConversion[i - 1]} />
            ))}
          </div>
        </div>

        {/* Breakdown grids */}
        <div className="grid grid-cols-3 gap-3.5">
          {/* By time of day */}
          <div
            className="border-line rounded-xl border p-[18px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="mono tracked text-text-3 mb-3 text-[9.5px]">BY TIME OF DAY</div>
            <ByTimeChart bars={data.byTime} />
            <div className="mb-2 mt-2 flex justify-between">
              {['00', '06', '12', '18', '23'].map((h) => (
                <span key={h} className="mono text-text-3 text-[10px]">
                  {h}
                </span>
              ))}
            </div>
            <div className="mono text-text-2 mt-2 text-[11px]">
              Peak{' '}
              <span style={{ color: 'var(--zz-lime)' }}>20:00 — 22:00</span>{' '}
              (32% of conv.)
            </div>
          </div>

          {/* By post */}
          <div
            className="border-line rounded-xl border p-[18px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="mono tracked text-text-3 mb-3 text-[9.5px]">BY POST</div>
            {data.byPost.map((row, i) => (
              <ByPostRow key={row.title} row={row} rank={i + 1} first={i === 0} />
            ))}
          </div>

          {/* By keyword intent */}
          <div
            className="border-line rounded-xl border p-[18px]"
            style={{ background: 'var(--zz-bg-2)' }}
          >
            <div className="mono tracked text-text-3 mb-3 text-[9.5px]">BY KEYWORD INTENT</div>
            {data.byIntent.map((row) => (
              <IntentBar key={row.label} row={row} />
            ))}
          </div>
        </div>
      </div>
    </>
  );
}

// ── Sub-components ────────────────────────────────────────────────────────────

function StepBar({
  step,
  idx,
  prev,
}: {
  step: StepConversion;
  idx: number;
  prev: StepConversion | undefined;
}) {
  const dropPt = prev ? (prev.pct - step.pct).toFixed(1) : null;
  // color-mix fades from lime toward bg as steps progress
  const bgColor =
    idx === 0
      ? 'var(--zz-lime)'
      : `color-mix(in oklch, var(--zz-lime) ${100 - idx * 12}%, #0a0a0a)`;

  return (
    <div className="mb-2.5">
      <div className="mb-1 flex justify-between text-[12px]">
        <span>{step.label}</span>
        <span className="mono">
          <span>{step.n.toLocaleString('id-ID')}</span>
          <span className="text-text-3"> · {step.pct}%</span>
        </span>
      </div>
      <div className="relative overflow-hidden rounded-[3px]" style={{ height: 18, background: 'var(--zz-bg)' }}>
        <div
          className="absolute inset-y-0 left-0 rounded-[3px]"
          style={{ width: `${step.pct}%`, background: bgColor }}
        />
        {dropPt && (
          <span
            className="mono absolute right-1.5 top-[2px] text-[9.5px]"
            style={{ color: 'var(--zz-pink)' }}
          >
            −{dropPt}pt
          </span>
        )}
      </div>
    </div>
  );
}

function ByPostRow({ row, rank, first }: { row: PostRevRow; rank: number; first: boolean }) {
  return (
    <div
      className="flex items-center gap-2.5 py-2"
      style={{ borderTop: first ? 'none' : '1px solid #1a1a1d' }}
    >
      <span className="mono text-text-3 w-[18px] text-[11px]">{rank}</span>
      <div className="flex-1">
        <div className="text-[12.5px]">{row.title}</div>
        <div className="mono text-text-3 text-[10.5px]">
          {row.comments.toLocaleString('id-ID')} comments
        </div>
      </div>
      <span className="mono text-[12px]" style={{ color: 'var(--zz-lime)' }}>
        {row.revenue}
      </span>
    </div>
  );
}

function IntentBar({ row }: { row: KeywordIntentBar }) {
  return (
    <div className="mb-2">
      <div className="mb-[3px] flex justify-between text-[12px]">
        <span className="mono text-text-2">{row.label}</span>
        <span className="mono">
          <span className="text-text">{row.n.toLocaleString('id-ID')}</span>
          <span className="text-text-3"> · {row.pct}%</span>
        </span>
      </div>
      <div className="h-1 overflow-hidden rounded-[2px]" style={{ background: 'var(--zz-bg)' }}>
        <div
          className="h-1 rounded-[2px]"
          style={{ width: `${row.pct * 3}%`, background: row.color }}
        />
      </div>
    </div>
  );
}
