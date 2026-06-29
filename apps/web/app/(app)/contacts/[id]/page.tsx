import { notFound } from 'next/navigation';
import { Avatar, Button, Meter, Pill } from '@zosmed/ui';
import { getContact } from '@/lib/mock/api';
import type { TimelineEvent, LeadScoreBreakdown, FlowRef } from '@/lib/mock/contacts';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';

interface Props {
  params: Promise<{ id: string }>;
}

export default async function ContactProfilePage({ params }: Props) {
  const { id } = await params;
  const profile = await getContact(id);

  if (!profile) notFound();

  return (
    <>
      <PageHeader>
        <PageHeaderBreadcrumb
          crumbs={[{ label: 'Contacts', href: '/contacts' }, { label: profile.name }]}
        />
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Add tag
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Send DM
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            Add to flow
          </Button>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto">
        {/* Hero section */}
        <div
          className="flex items-start gap-5 px-8 py-7"
          style={{ borderBottom: '1px solid #1a1a1d' }}
        >
          <Avatar name={profile.avatar} color={profile.avatarColor} size={84} />
          <div className="flex-1">
            <div className="mb-1.5 flex items-center gap-2.5">
              <h1 className="m-0 text-[30px] font-medium tracking-tight">{profile.name}</h1>
              {profile.tags.map((t) => (
                <Pill key={t.label} tone={t.tone}>
                  {t.label}
                </Pill>
              ))}
              <span className="text-text-3 text-[12px]">+ add tag</span>
            </div>
            <div className="mono text-text-2 text-[12.5px]">{profile.metaLine}</div>

            {/* Stats row */}
            <div className="mt-5 flex gap-6">
              {profile.stats.map((s) => (
                <div key={s.key}>
                  <div className="mono tracked text-text-3 text-[9.5px]">{s.key}</div>
                  <div
                    className="mono mt-0.5 text-base"
                    style={{ color: s.highlight ? 'var(--zz-lime)' : 'var(--zz-text-2)' }}
                  >
                    {s.value}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Tab strip */}
        <div className="flex gap-1.5 px-8" style={{ borderBottom: '1px solid #1a1a1d' }}>
          {profile.tabs.map((t, i) => (
            <span
              key={t}
              className="cursor-pointer py-3 px-3.5 text-[13px]"
              style={{
                color: i === 0 ? 'var(--zz-text)' : 'var(--zz-text-3)',
                borderBottom: i === 0 ? '2px solid var(--zz-lime)' : '2px solid transparent',
                marginBottom: -1,
              }}
            >
              {t}
            </span>
          ))}
        </div>

        {/* Body: timeline + right rail */}
        <div
          className="grid gap-[18px] p-8"
          style={{ gridTemplateColumns: '1.5fr 1fr' }}
        >
          {/* Activity timeline */}
          <ActivityTimeline events={profile.timeline} />

          {/* Right rail */}
          <div className="flex flex-col gap-3.5">
            <LeadScoreCard
              score={profile.leadScore}
              delta={profile.leadScoreDelta}
              breakdown={profile.leadScoreBreakdown}
            />
            <PropertiesCard properties={profile.properties} />
            <InFlowsCard flows={profile.flows} />
          </div>
        </div>
      </div>
    </>
  );
}

// ── Sub-components ────────────────────────────────────────────────────────────

function ActivityTimeline({ events }: { events: TimelineEvent[] }) {
  return (
    <div>
      <div className="mb-3.5 flex items-center justify-between">
        <h3 className="m-0 text-base font-medium">Activity timeline</h3>
        <span className="mono text-text-3 text-[10.5px]">
          {events.length} events · 30 hari terakhir
        </span>
      </div>

      {events.length === 0 ? (
        <p className="text-text-3 text-sm">Belum ada aktivitas.</p>
      ) : (
        <div className="relative">
          <div
            className="absolute"
            style={{ left: 11, top: 6, bottom: 6, width: 1, background: 'var(--zz-line)' }}
          />
          {events.map((e, i) => (
            <TimelineItem key={`${e.kind}-${i}`} e={e} />
          ))}
        </div>
      )}
    </div>
  );
}

function TimelineItem({ e }: { e: TimelineEvent }) {
  return (
    <div className="relative pb-4 pl-8">
      {/* Icon ring */}
      <span
        className="bg-bg-2 absolute left-0 top-0 inline-flex h-[22px] w-[22px] items-center justify-center rounded-full border text-[11px]"
        style={{ borderColor: e.iconColor }}
      >
        {e.icon}
      </span>

      <div className="mb-1 flex items-center justify-between">
        <span className="mono tracked text-[9.5px]" style={{ color: e.iconColor }}>
          {e.kind.toUpperCase()}
        </span>
        <span className="mono text-text-3 text-[10.5px]">{e.date}</span>
      </div>
      <div className="text-[13px] font-medium">{e.title}</div>
      <div className="text-text-2 mt-0.5 text-[12px]">{e.sub}</div>
      {e.quoted ? (
        <div
          className="mono text-text-2 mt-1.5 rounded-md px-2.5 py-1.5 text-[11.5px] leading-relaxed"
          style={{ background: 'var(--zz-bg-2)', border: '1px solid #1a1a1d' }}
        >
          {e.quoted}
        </div>
      ) : null}
    </div>
  );
}

function LeadScoreCard({
  score,
  delta,
  breakdown,
}: {
  score: number;
  delta: string;
  breakdown: LeadScoreBreakdown[];
}) {
  return (
    <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
      <div className="mono tracked text-text-3 mb-3 text-[9.5px]">LEAD SCORE BREAKDOWN</div>
      <div className="mb-3 flex items-baseline gap-1.5">
        <span className="mono text-[36px] font-medium">{score}</span>
        <span className="mono text-text-3 text-sm">/ 100</span>
        {delta ? (
          <span className="mono text-lime ml-auto text-[11px]">{delta}</span>
        ) : null}
      </div>
      {breakdown.map((b) => (
        <div key={b.label} className="mb-2">
          <div className="mb-[3px] flex justify-between text-[11.5px]">
            <span className="text-text-2">{b.label}</span>
            <span className="mono">
              {b.value} / {b.max}
            </span>
          </div>
          <Meter value={b.value / b.max} height={4} trackClassName="bg-bg-3" />
        </div>
      ))}
    </div>
  );
}

function PropertiesCard({ properties }: { properties: { key: string; value: string }[] }) {
  return (
    <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
      <div className="mono tracked text-text-3 mb-3 text-[9.5px]">PROPERTIES</div>
      {properties.map((p) => (
        <div
          key={p.key}
          className="flex justify-between py-1.5 text-[12px]"
          style={{ borderBottom: '1px solid #1a1a1d' }}
        >
          <span className="text-text-3">{p.key}</span>
          <span
            className="mono"
            style={{ color: p.value === '—' ? 'var(--zz-text-3)' : 'var(--zz-text)' }}
          >
            {p.value}
          </span>
        </div>
      ))}
    </div>
  );
}

function InFlowsCard({ flows }: { flows: FlowRef[] }) {
  if (flows.length === 0) return null;
  return (
    <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
      <div className="mono tracked text-text-3 mb-3 text-[9.5px]">
        IN FLOWS · {flows.length} AKTIF
      </div>
      {flows.map((f, i) => (
        <div
          key={f.name}
          className="py-2"
          style={{ borderBottom: i < flows.length - 1 ? '1px solid #1a1a1d' : 'none' }}
        >
          <div className="mono text-[12.5px]">{f.name}</div>
          <div className="mono mt-0.5 text-[10.5px]" style={{ color: f.statusColor }}>
            {f.status}
          </div>
        </div>
      ))}
    </div>
  );
}
