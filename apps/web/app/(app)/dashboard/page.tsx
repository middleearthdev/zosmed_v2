import { Button, Card, Gauge, I, Meter, Pill, SectionHeader, Stat, StatCard } from '@zosmed/ui';
import { getAccount, getDashboard } from '@/lib/mock/api';
import type { DashboardWorkflow, FeedEvent } from '@/lib/mock/dashboard';
import { Topbar } from '../_components/Topbar';

export default async function DashboardPage() {
  const [account, data] = await Promise.all([getAccount(), getDashboard()]);
  const maxKeyword = Math.max(...data.keywords.map((k) => k.hits));

  return (
    <>
      <Topbar account={account} />
      <div className="zz-scroll flex-1 overflow-y-auto px-8 pb-12 pt-8">
        {/* Header */}
      <div className="mb-8 flex items-end justify-between">
        <div>
          <span className="mono tracked text-text-3 text-[10px]">{data.dateLabel}</span>
          <h1 className="m-0 mb-1.5 mt-1.5 text-4xl font-medium tracking-tight">{data.greeting}</h1>
          <p className="text-text-2 m-0 text-sm">{data.summary}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" icon={<I.plus />}>
            Buat workflow
          </Button>
          <Button variant="lime" icon={<I.bolt />}>
            Connect new account
          </Button>
        </div>
      </div>

      {/* Stat tiles */}
      <div className="mb-6 grid grid-cols-4 gap-3">
        {data.stats.map((s) => (
          <StatCard key={s.label} label={s.label} value={s.value} delta={s.delta} accent={s.accent} spark={s.spark} />
        ))}
      </div>

      {/* Two-column row */}
      <div className="grid gap-3" style={{ gridTemplateColumns: '1.5fr 1fr' }}>
        <Card>
          <SectionHeader
            title="Active workflows"
            subtitle="3 berjalan · 1 paused · 2 draft"
            action={
              <span className="text-lime flex items-center gap-1 text-xs">
                View all <I.arrow />
              </span>
            }
          />
          <div className="flex flex-col gap-2">
            {data.workflows.map((w) => (
              <WorkflowRow key={w.name} w={w} />
            ))}
          </div>
        </Card>

        <Card>
          <SectionHeader
            title="Live feed"
            action={
              <span className="text-lime flex items-center gap-1.5 text-[11px]">
                <span
                  className="h-1.5 w-1.5 rounded-full"
                  style={{ background: 'var(--zz-lime)', animation: 'zz-pulse 1.4s infinite' }}
                />
                REAL-TIME
              </span>
            }
          />
          <div className="relative flex flex-col">
            <div className="bg-line absolute bottom-2 top-2" style={{ left: 13, width: 1 }} />
            {data.feed.map((e, i) => (
              <FeedItem key={`${e.who}-${e.t}-${i}`} e={e} />
            ))}
          </div>
        </Card>
      </div>

      {/* Bottom row */}
      <div className="mt-3 grid grid-cols-3 gap-3">
        <Card>
          <SectionHeader size="sm" title="Top keywords (7d)" action={<span className="mono text-text-3 text-[10px]">HITS</span>} />
          {data.keywords.map((k) => {
            const ratio = k.hits / maxKeyword;
            return (
              <div key={k.keyword} className="mb-2 flex items-center gap-2.5">
                <span className="mono text-text-2 w-20 text-xs">&quot;{k.keyword}&quot;</span>
                <div className="bg-bg-3 h-1.5 flex-1 rounded-[3px]">
                  <div
                    className="h-1.5 rounded-[3px]"
                    style={{ width: `${ratio * 100}%`, background: 'var(--zz-lime)', opacity: 0.4 + ratio * 0.6 }}
                  />
                </div>
                <span className="mono tnum w-9 text-right text-xs">{k.hits}</span>
              </div>
            );
          })}
        </Card>

        <Card>
          <SectionHeader size="sm" title="Safety status" action={<Pill tone="lime">HEALTHY</Pill>} />
          {data.safety.map((s) => (
            <Gauge key={s.label} label={s.label} valueText={s.value} capText={s.cap} value={s.pct} />
          ))}
        </Card>

        <Card>
          <SectionHeader size="sm" title="Quick start" />
          {data.quickStart.map((q, i) => (
            <div
              key={q.title}
              className="flex items-center gap-2.5 py-2.5"
              style={{ borderTop: i > 0 ? '1px solid #1a1a1d' : 'none' }}
            >
              <span className="bg-bg-3 text-lime inline-flex h-7 w-7 items-center justify-center rounded-[7px]">
                {I[q.iconKey]()}
              </span>
              <div className="flex-1">
                <div className="text-[13px]">{q.title}</div>
                <div className="mono text-text-3 text-[10.5px]">{q.sub}</div>
              </div>
              {I.arrow({ style: { color: 'var(--zz-text-3)' } })}
            </div>
          ))}
        </Card>
      </div>
      </div>
    </>
  );
}

function WorkflowRow({ w }: { w: DashboardWorkflow }) {
  const running = w.status === 'running';
  return (
    <div className="bg-bg-3 border-line rounded-lg border px-3.5 py-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <span
            className="h-2 w-2 rounded-full"
            style={{
              background: running ? 'var(--zz-lime)' : 'var(--zz-text-3)',
              boxShadow: running ? '0 0 8px var(--zz-lime)' : 'none',
            }}
          />
          <span className="mono text-[13px]">{w.name}.flow</span>
          {w.status === 'paused' ? <Pill tone="neutral">PAUSED</Pill> : null}
        </div>
        <div className="flex items-center gap-6">
          <Stat value={w.triggers.toLocaleString()} label="triggers" />
          <Stat value={w.dms.toLocaleString()} label="dms" />
          <Stat value={w.leads.toLocaleString()} label="leads" highlight />
        </div>
      </div>
      <div className="mt-2.5 flex items-center gap-3">
        <span className="mono text-text-3 flex-1 text-[11px]">{w.post}</span>
        <div className="w-[200px]">
          <Meter value={w.progress} height={3} trackClassName="bg-line" />
        </div>
      </div>
    </div>
  );
}

function FeedItem({ e }: { e: FeedEvent }) {
  return (
    <div className="relative flex items-start gap-3 py-2.5">
      <span
        className="bg-bg-3 z-[1] inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full border"
        style={{ borderColor: e.color, color: e.color }}
      >
        {I[e.iconKey]()}
      </span>
      <div className="flex-1 pt-1">
        <div className="text-[13px]">
          <span className="mono text-text-2">@{e.who}</span> <span className="text-text">{e.what}</span>
        </div>
        <div className="mono text-text-3 mt-0.5 text-[10.5px]">{e.wf}.flow</div>
      </div>
      <span className="mono text-text-3 text-[10.5px]">{e.t}</span>
    </div>
  );
}
