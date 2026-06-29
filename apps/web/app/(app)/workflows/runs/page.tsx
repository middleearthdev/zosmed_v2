import { Button, Card, Dot, Pill } from '@zosmed/ui';
import { getWorkflowRuns } from '@/lib/mock/api';
import type { RunStatusKey } from '@/lib/mock/workflows';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';
import { RunRateChart } from '../_components/RunRateChart';

const STATUS_TONE: Record<RunStatusKey, 'lime' | 'pink' | 'warn'> = {
  success: 'lime',
  failed: 'pink',
  review: 'warn',
};
const RUN_FILTERS = ['All', 'Success', 'Failed', 'Review'];
const GRID = 'grid-cols-[90px_1.2fr_0.7fr_0.6fr_90px]';

export default async function WorkflowRunsPage() {
  const data = await getWorkflowRuns();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb
            crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: 'launch-promo-mei' }, { label: 'Runs' }]}
          />
          <Pill tone="lime">● LIVE</Pill>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            ⏸ Pause
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Edit canvas
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            Test run
          </Button>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        {/* KPI strip */}
        <div className="mb-[18px] grid grid-cols-5 gap-2.5">
          {data.kpis.map((k) => (
            <div key={k.label} className="bg-bg-2 border-line rounded-[10px] border p-3.5">
              <div className="mono tracked text-text-3 text-[9.5px]">{k.label}</div>
              <div className="mono mt-1 text-[22px] font-medium">{k.value}</div>
              <div className="mono mt-0.5 text-[11px]" style={{ color: k.color }}>
                {k.delta}
              </div>
            </div>
          ))}
        </div>

        {/* Run rate timeline */}
        <Card className="mb-3.5 p-[18px]">
          <div className="mb-3 flex items-center justify-between">
            <span className="mono tracked text-text-3 text-[9.5px]">RUN RATE · LAST 24 HOURS</span>
            <div className="mono flex gap-3 text-[11px]">
              <span className="inline-flex items-center gap-1.5">
                <Dot color="var(--zz-lime)" /> success
              </span>
              <span className="inline-flex items-center gap-1.5">
                <Dot color="var(--zz-warn)" /> review
              </span>
              <span className="inline-flex items-center gap-1.5">
                <Dot color="var(--zz-pink)" /> failed
              </span>
            </div>
          </div>
          <RunRateChart bars={data.runRate} />
        </Card>

        <div className="grid gap-3.5" style={{ gridTemplateColumns: '1.5fr 1fr' }}>
          {/* Run list */}
          <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
            <div className="border-line flex items-center justify-between border-b px-4 py-3.5">
              <span className="text-sm font-medium">Recent runs</span>
              <div className="flex gap-1.5">
                {RUN_FILTERS.map((t, i) => (
                  <span
                    key={t}
                    className={`mono rounded-full px-2.5 py-1 text-[10.5px] ${i === 0 ? 'bg-bg-3 text-text' : 'text-text-3'}`}
                  >
                    {t}
                  </span>
                ))}
              </div>
            </div>
            <div className={`grid ${GRID} border-line border-b px-4 py-2`} style={{ background: '#0d0d0d' }}>
              {['RUN ID', 'TRIGGER', 'DURATION', 'STEPS', 'STATUS'].map((h) => (
                <span key={h} className="mono tracked text-text-3 text-[9.5px]">
                  {h}
                </span>
              ))}
            </div>
            {data.runs.map((r, i) => {
              const [done, total] = r.steps.split('/');
              return (
                <div
                  key={r.id}
                  className={`grid ${GRID} border-bg-3 items-center gap-2 border-b px-4 py-3 last:border-b-0`}
                  style={{
                    background: i === 0 ? 'var(--zz-bg-3)' : 'transparent',
                    borderLeft: i === 0 ? '2px solid var(--zz-lime)' : '2px solid transparent',
                  }}
                >
                  <span className="mono text-text-2 text-[11.5px]">{r.id}</span>
                  <div>
                    <div className="text-[12.5px]">{r.trig}</div>
                    <div className="mono text-text-3 text-[10.5px]">
                      {r.t} · {r.wf}
                    </div>
                  </div>
                  <span className="mono tnum text-xs">{r.dur}</span>
                  <span className="mono text-[11.5px]" style={{ color: done === total ? 'var(--zz-text-2)' : 'var(--zz-pink)' }}>
                    {r.steps}
                  </span>
                  <Pill tone={STATUS_TONE[r.status]}>{r.status}</Pill>
                </div>
              );
            })}
          </div>

          {/* Run detail */}
          <Card className="p-[18px]">
            <div className="mb-3.5 flex items-center justify-between">
              <div>
                <div className="mono tracked text-text-3 text-[9.5px]">RUN DETAIL</div>
                <div className="mono mt-0.5 text-sm">{data.detail.id}</div>
              </div>
              <Pill tone="lime">success</Pill>
            </div>

            <div className="mb-4 grid grid-cols-2 gap-2 text-[11.5px]">
              {data.detail.facts.map(([k, v]) => (
                <div key={k} className="flex justify-between py-[5px]" style={{ borderBottom: '1px solid #1a1a1d' }}>
                  <span className="text-text-3">{k}</span>
                  <span className="mono">{v}</span>
                </div>
              ))}
            </div>

            <div className="mono tracked text-text-3 mb-2.5 text-[9.5px]">STEP TIMELINE</div>
            <div className="relative">
              <div className="bg-line absolute" style={{ left: 9, top: 14, bottom: 14, width: 1 }} />
              {data.detail.steps.map((s, i) => (
                <div key={i} className="relative pb-3 pl-7">
                  <span
                    className="absolute rounded-full"
                    style={{ left: 5, top: 6, width: 10, height: 10, background: s.color, border: '2px solid var(--zz-bg-2)' }}
                  />
                  <div className="mb-[3px] flex items-center justify-between">
                    <span className="mono tracked text-[9px]" style={{ color: s.color }}>
                      {s.k}
                    </span>
                    <span className="mono text-text-3 text-[10px]">
                      {s.d} · {s.dur}
                    </span>
                  </div>
                  <div className="mb-[3px] text-[12.5px]">{s.l}</div>
                  <div
                    className="mono text-text-3 rounded-[3px] px-1.5 py-[3px] text-[10.5px]"
                    style={{ background: 'var(--zz-bg)', border: '1px solid #1a1a1d' }}
                  >
                    {s.payload}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </>
  );
}
