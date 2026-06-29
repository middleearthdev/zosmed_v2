import { Avatar, Button, I, Meter, Pill } from '@zosmed/ui';
import { getSettings } from '@/lib/mock/api';
import { STATUS_TONE } from '@/lib/mock/system';
import { PageHeader } from '../_components/PageHeader';

const ACC_GRID = 'grid-cols-[40px_1.4fr_0.8fr_0.8fr_1fr_80px]';

export default async function SettingsPage() {
  const data = await getSettings();

  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Settings</span>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        {/* Section nav */}
        <div className="border-bg-3 w-[220px] border-r p-[18px]">
          {data.nav.map((t) => {
            const active = t === data.activeNav;
            return (
              <div
                key={t}
                className={`mb-0.5 rounded-md py-2 pr-3 text-[13px] ${active ? 'bg-bg-3 text-text' : 'text-text-2'}`}
                style={{ borderLeft: active ? '2px solid var(--zz-lime)' : '2px solid transparent', paddingLeft: 10 }}
              >
                {t}
              </div>
            );
          })}
        </div>

        {/* Content */}
        <div className="zz-scroll flex-1 overflow-y-auto p-8">
          <h1 className="m-0 mb-1.5 text-3xl font-medium tracking-tight">Connected accounts</h1>
          <p className="text-text-2 m-0 mb-7 max-w-[600px] text-sm">
            Hubungkan akun Instagram untuk diatur otomatis. Workspace Pro support hingga 5 akun · Enterprise unlimited.
          </p>

          <div className="bg-bg-2 border-line mb-3.5 rounded-xl border p-1">
            {data.accounts.map((a, i) => (
              <div
                key={a.ig}
                className={`grid ${ACC_GRID} border-bg-3 items-center gap-3 px-4 py-3.5 ${i < data.accounts.length - 1 ? 'border-b' : ''}`}
              >
                <Avatar name={a.ig.slice(1, 3).toUpperCase()} color={i === 0 ? 'var(--zz-lime)' : '#3a3a40'} size={36} />
                <div>
                  <div className="flex items-center gap-1.5 text-sm font-medium">
                    {a.n}
                    {a.primary ? <Pill tone="lime">PRIMARY</Pill> : null}
                  </div>
                  <div className="mono text-text-3 text-[11.5px]">
                    {a.ig} · {a.f} followers
                  </div>
                </div>
                <Pill tone={a.perm === 'full' ? 'lime' : 'neutral'}>{a.perm}</Pill>
                <span className="mono text-text-3 text-[11px]">last sync {a.last}</span>
                <div className="flex gap-1 text-[11.5px]">
                  <span className="text-text-2">Configure</span>
                  <span className="text-line-2">·</span>
                  <span className="text-text-2">Re-auth</span>
                </div>
                <span className="mono text-right text-[11px]" style={{ color: 'var(--zz-pink)' }}>
                  Disconnect
                </span>
              </div>
            ))}
            <div className="border-bg-3 flex items-center gap-2.5 border-t p-3.5">
              <span className="border-line-2 text-text-3 inline-flex h-9 w-9 items-center justify-center rounded-lg border border-dashed">
                <I.plus />
              </span>
              <span className="text-text-2 text-[13px]">Connect another Instagram account</span>
              <Button variant="ghost" className="ml-auto px-3 py-1.5 text-xs">
                Connect
              </Button>
            </div>
          </div>

          <h2 className="mb-3 mt-8 text-lg font-medium">Other integrations</h2>
          <div className="grid grid-cols-3 gap-3">
            {data.integrations.map((it) => (
              <div key={it.n} className="bg-bg-2 border-line flex flex-col gap-2 rounded-[10px] border p-4">
                <div className="flex items-center justify-between">
                  <span className="bg-bg-3 mono inline-flex h-8 w-8 items-center justify-center rounded-md text-sm text-text-2">⌬</span>
                  <Pill tone={STATUS_TONE[it.s] ?? 'neutral'}>{it.s}</Pill>
                </div>
                <div className="text-sm font-medium">{it.n}</div>
                <div className="text-text-2 text-xs">{it.d}</div>
              </div>
            ))}
          </div>

          <h2 className="mb-3 mt-8 text-lg font-medium">Plan &amp; usage</h2>
          <div className="bg-bg-2 border-line grid items-center gap-6 rounded-xl border p-[22px]" style={{ gridTemplateColumns: '1.4fr 1fr' }}>
            <div>
              <div className="flex items-center gap-2.5">
                <span className="text-[22px] font-medium">Pro plan</span>
                <Pill tone="lime">CURRENT</Pill>
              </div>
              <div className="mono text-text-2 mt-1.5 text-[13px]">Rp 490,000 / bulan · billed monthly</div>
              <div className="mt-4 flex gap-4">
                <div>
                  <div className="mono tracked text-text-3 text-[9.5px]">NEXT INVOICE</div>
                  <div className="mono mt-0.5 text-sm">15 May 2026</div>
                </div>
                <div>
                  <div className="mono tracked text-text-3 text-[9.5px]">PAYMENT</div>
                  <div className="mono mt-0.5 text-sm">•••• 4821</div>
                </div>
              </div>
            </div>
            <div>
              {data.usage.map(([k, v, cap]) => (
                <div key={k} className="mb-2.5">
                  <div className="mb-1 flex justify-between text-xs">
                    <span className="text-text-2">{k}</span>
                    <span className="mono">
                      <span>{v.toLocaleString()}</span>
                      <span className="text-text-3"> / {cap.toLocaleString()}</span>
                    </span>
                  </div>
                  <Meter value={v / cap} />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
