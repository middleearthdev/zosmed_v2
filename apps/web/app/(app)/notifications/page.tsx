import { Button, Pill } from '@zosmed/ui';
import { getNotifications } from '@/lib/mock/api';
import { PageHeader } from '../_components/PageHeader';

const NOTIF_GRID = 'grid-cols-[36px_1fr_80px_60px]';
const QUICK = ['Snooze 2 hours', 'Snooze until tomorrow', 'Create slack channel hook'];

export default async function NotificationsPage() {
  const data = await getNotifications();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <span className="text-sm font-medium">Notifications</span>
          <Pill tone="lime">11 NEW</Pill>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Mark all read
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Notification settings
          </Button>
        </div>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        {/* Filter rail */}
        <div className="border-bg-3 w-[220px] border-r p-[18px]">
          <div className="mono tracked text-text-3 mb-2.5 text-[9.5px]">FILTER</div>
          {data.filters.map(([l, n], i) => {
            const active = i === 0;
            return (
              <div
                key={l}
                className={`mb-0.5 flex justify-between rounded-md py-[7px] pr-2.5 text-[13px] ${active ? 'bg-bg-3 text-text' : 'text-text-2'}`}
                style={{ borderLeft: active ? '2px solid var(--zz-lime)' : '2px solid transparent', paddingLeft: 9 }}
              >
                <span>{l}</span>
                <span className="mono text-text-3 text-[11px]">{n}</span>
              </div>
            );
          })}
          <div className="mono tracked text-text-3 mb-2.5 mt-6 text-[9.5px]">QUICK ACTIONS</div>
          {QUICK.map((q) => (
            <div key={q} className="text-text-2 px-[9px] py-[7px] text-[12.5px]">
              {q}
            </div>
          ))}
        </div>

        {/* Notification list */}
        <div className="zz-scroll flex-1 overflow-y-auto px-7 pb-8 pt-[18px]">
          {data.groups.map((g) => (
            <div key={g.d} className="mb-[22px]">
              <div className="mono tracked text-text-3 my-2.5 text-[9.5px]">{g.d}</div>
              <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
                {g.items.map((n, i) => (
                  <div key={i} className={`grid ${NOTIF_GRID} border-bg-3 items-start gap-3.5 px-4 py-3.5 ${i ? 'border-t' : ''}`}>
                    <span
                      className="bg-bg inline-flex h-[30px] w-[30px] items-center justify-center rounded-full text-[13px]"
                      style={{ border: `1px solid ${n.c}`, color: n.c }}
                    >
                      {n.icon}
                    </span>
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-[13.5px] font-medium">{n.t}</span>
                        {i === 0 && g.d === 'TODAY' ? <span className="h-1.5 w-1.5 rounded-full" style={{ background: 'var(--zz-lime)' }} /> : null}
                      </div>
                      <div className="text-text-2 mt-[3px] text-xs">{n.sub}</div>
                      {n.val ? (
                        <div className="mono text-text-2 bg-bg mt-2 rounded-md px-2.5 py-1.5 text-[11.5px]" style={{ border: '1px solid #1a1a1d' }}>
                          {n.val}
                        </div>
                      ) : null}
                    </div>
                    <span className="mono text-text-3 text-[11px]">{n.t2}</span>
                    <span className="mono text-lime text-right text-[11px]">View →</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
