import { Avatar, Button, I, Pill } from '@zosmed/ui';
import { getTeam } from '@/lib/mock/api';
import { ROLE_COLOR } from '@/lib/mock/system';
import { PageHeader } from '../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../_components/PageHeaderBreadcrumb';

const MEMBER_GRID = 'grid-cols-[2fr_1fr_1.4fr_1fr_80px]';
const ACT_GRID = 'grid-cols-[90px_100px_1fr_140px]';

export default async function TeamPage() {
  const data = await getTeam();

  return (
    <>
      <PageHeader>
        <PageHeaderBreadcrumb crumbs={[{ label: 'Settings', href: '/settings' }, { label: 'Members' }]} />
        <Button variant="lime" className="px-3 py-[7px] text-xs">
          <I.plus /> Invite member
        </Button>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-8">
        <h1 className="m-0 mb-1.5 text-3xl font-medium tracking-tight">Team &amp; permissions</h1>
        <p className="text-text-2 m-0 mb-6 text-sm">5 members · 1 pending · Pro plan supports up to 10 seats</p>

        {/* Roles legend */}
        <div className="mb-[18px] grid grid-cols-4 gap-2.5">
          {data.roles.map((r) => (
            <div key={r.r} className="bg-bg-2 border-line rounded-[10px] border p-3.5">
              <div className="flex items-center gap-2">
                <span className="h-2 w-2 rounded-full" style={{ background: r.c }} />
                <span className="text-[13px] font-medium">{r.r}</span>
              </div>
              <div className="text-text-2 mt-1 text-[11.5px] leading-normal">{r.d}</div>
            </div>
          ))}
        </div>

        {/* Members table */}
        <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
          <div className={`grid ${MEMBER_GRID} border-line border-b px-[18px] py-2.5`} style={{ background: '#0d0d0d' }}>
            {['MEMBER', 'ROLE', 'IG ACCOUNTS', 'LAST ACTIVE', ''].map((h, i) => (
              <span key={i} className="mono tracked text-text-3 text-[9.5px]">
                {h}
              </span>
            ))}
          </div>
          {data.members.map((m, i) => (
            <div
              key={m.e || m.n}
              className={`grid ${MEMBER_GRID} border-bg-3 items-center gap-2 px-[18px] py-3.5 ${i < data.members.length - 1 ? 'border-b' : ''}`}
              style={{ opacity: m.pending ? 0.6 : 1 }}
            >
              <div className="flex items-center gap-3">
                <Avatar name={m.avatar} color={m.color} size={36} />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{m.n}</span>
                    {m.you ? <Pill tone="lime">YOU</Pill> : null}
                    {m.pending ? <Pill tone="warn">PENDING</Pill> : null}
                  </div>
                  {m.e ? <div className="mono text-text-3 mt-0.5 text-[11.5px]">{m.e}</div> : null}
                </div>
              </div>
              <span className="mono text-text-2 inline-flex items-center gap-1.5 text-xs">
                <span className="h-1.5 w-1.5 rounded-full" style={{ background: ROLE_COLOR[m.role] }} />
                {m.role} {!m.pending && !m.you ? <span className="text-line-2">▾</span> : null}
              </span>
              <div className="flex flex-wrap gap-1">
                {m.acc.map((a) => (
                  <Pill key={a} tone="neutral">
                    @{a}
                  </Pill>
                ))}
              </div>
              <span className="mono text-[11.5px]" style={{ color: m.last === 'online now' ? 'var(--zz-lime)' : 'var(--zz-text-3)' }}>
                {m.last}
              </span>
              <span className="mono text-right text-[11px]" style={{ color: m.you ? '#3a3a40' : 'var(--zz-text-2)' }}>
                {m.you ? '—' : '⋯'}
              </span>
            </div>
          ))}
        </div>

        <h2 className="mb-3 mt-8 text-lg font-medium">Recent team activity</h2>
        <div className="bg-bg-2 border-line rounded-xl border p-1.5">
          {data.activity.map((r, i) => (
            <div key={i} className={`grid ${ACT_GRID} border-bg-3 items-center gap-2.5 px-3.5 py-2.5 ${i ? 'border-t' : ''}`}>
              <span className="mono text-text-3 text-[11px]">{r[0]}</span>
              <span className="mono text-xs" style={{ color: r[4] }}>
                {r[1]}
              </span>
              <span className="text-text-2 text-[12.5px]">
                {r[2]} <span className="text-text">{r[3]}</span>
              </span>
              <span className="mono text-lime text-right text-[11px]">View change →</span>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
