import { Button, I, Meter, Pill } from '@zosmed/ui';
import { getBilling } from '@/lib/mock/api';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';

const INV_GRID = 'grid-cols-[120px_110px_1fr_90px_80px_80px]';

export default async function BillingPage() {
  const data = await getBilling();

  return (
    <>
      <PageHeader>
        <PageHeaderBreadcrumb crumbs={[{ label: 'Settings', href: '/settings' }, { label: 'Billing' }]} />
        <Button variant="lime" className="px-3 py-[7px] text-xs">
          Upgrade plan
        </Button>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-8">
        <h1 className="m-0 mb-1.5 text-3xl font-medium tracking-tight">Billing &amp; usage</h1>
        <p className="text-text-2 m-0 mb-7 text-sm">Pro plan · Rp 490,000 / bulan · billed monthly · next invoice 15 May 2026</p>

        {/* Plan compare */}
        <div className="mb-6 grid grid-cols-3 gap-3.5">
          {data.plans.map((p) => {
            const current = p.tone === 'current';
            return (
              <div
                key={p.n}
                className="relative rounded-2xl p-[22px]"
                style={{
                  background: current ? 'linear-gradient(180deg, color-mix(in oklch, var(--zz-lime) 8%, var(--zz-bg-2)), var(--zz-bg-2))' : 'var(--zz-bg-2)',
                  border: `1px solid ${current ? 'var(--zz-lime)' : 'var(--zz-line)'}`,
                }}
              >
                {current ? (
                  <span
                    className="mono absolute left-[22px] font-semibold"
                    style={{ top: -10, background: 'var(--zz-lime)', color: 'var(--zz-bg)', fontSize: 10, padding: '3px 8px', borderRadius: 3 }}
                  >
                    YOUR PLAN
                  </span>
                ) : null}
                <div className="text-sm font-medium">{p.n}</div>
                <div className="mt-2 flex items-baseline gap-1.5">
                  <span className="mono text-[28px] font-medium">{p.p}</span>
                  <span className="mono text-text-3 text-[11px]">{p.sub}</span>
                </div>
                <div className="mt-[18px] flex flex-col gap-2 pt-[18px]" style={{ borderTop: '1px solid #1a1a1d' }}>
                  {p.f.map((f) => (
                    <div key={f} className="flex items-center gap-2 text-[12.5px]">
                      <span className="text-lime">
                        <I.check />
                      </span>
                      <span className="text-text-2">{f}</span>
                    </div>
                  ))}
                </div>
                <Button
                  variant={p.tone === 'lime' ? 'lime' : 'ghost'}
                  className="mt-[18px] w-full justify-center px-3 py-2 text-[12.5px]"
                  style={current ? { opacity: 0.5, cursor: 'default' } : undefined}
                >
                  {p.cta}
                </Button>
              </div>
            );
          })}
        </div>

        <div className="grid gap-3.5" style={{ gridTemplateColumns: '1.4fr 1fr' }}>
          {/* Invoices */}
          <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
            <div className="border-line flex justify-between border-b px-[18px] py-3.5">
              <span className="text-sm font-medium">Invoices</span>
              <span className="mono text-text-3 text-[11px]">last 6 months</span>
            </div>
            <div className={`grid ${INV_GRID} border-line border-b px-[18px] py-2`} style={{ background: '#0d0d0d' }}>
              {['DATE', 'INVOICE #', 'DESCRIPTION', 'AMOUNT', 'STATUS', ''].map((h, i) => (
                <span key={i} className="mono tracked text-text-3 text-[9.5px]">
                  {h}
                </span>
              ))}
            </div>
            {data.invoices.map((r) => (
              <div key={r[1]} className={`grid ${INV_GRID} border-bg-3 items-center gap-2 border-b px-[18px] py-3 last:border-b-0`}>
                <span className="mono text-text-2 text-xs">{r[0]}</span>
                <span className="mono text-xs">{r[1]}</span>
                <span className="text-[13px]">{r[2]}</span>
                <span className="mono text-xs">{r[3]}</span>
                <Pill tone="lime">{r[4]}</Pill>
                <span className="mono text-text-2 text-right text-[11px]">↓ PDF</span>
              </div>
            ))}
          </div>

          {/* Right column */}
          <div className="flex flex-col gap-3.5">
            <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
              <div className="mono tracked text-text-3 mb-3 text-[9.5px]">USAGE THIS CYCLE · 1 — 30 APR</div>
              {data.usage.map((u) => {
                const pct = u.v / u.cap;
                return (
                  <div key={u.l} className="mb-3">
                    <div className="mb-1 flex justify-between text-xs">
                      <span>{u.l}</span>
                      <span className="mono">
                        <span>{u.v.toLocaleString()}</span>
                        <span className="text-text-3"> / {u.cap.toLocaleString()}</span>
                      </span>
                    </div>
                    <Meter value={pct} height={5} color={pct > 0.8 ? 'var(--zz-warn)' : 'var(--zz-lime)'} />
                    <div className="mono text-text-3 mt-1 text-[10.5px]">{u.sub}</div>
                  </div>
                );
              })}
            </div>

            <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
              <div className="mono tracked text-text-3 mb-3 text-[9.5px]">PAYMENT METHOD</div>
              <div className="bg-bg flex items-center gap-3 rounded-lg p-3" style={{ border: '1px solid #1a1a1d' }}>
                <div
                  className="mono flex items-center justify-center rounded font-bold text-white"
                  style={{ width: 44, height: 30, background: 'linear-gradient(135deg, #1d72f0, #0a47b8)', fontSize: 9 }}
                >
                  VISA
                </div>
                <div className="flex-1">
                  <div className="mono text-[13px]">•••• •••• •••• 4821</div>
                  <div className="mono text-text-3 text-[10.5px]">expires 09 / 28 · Maya R.</div>
                </div>
                <span className="mono text-text-2 text-[11px]">Update</span>
              </div>
              <Button variant="ghost" className="mt-2.5 w-full justify-center px-3 py-[7px] text-xs">
                + Add payment method
              </Button>
            </div>

            <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
              <div className="mono tracked text-text-3 mb-2.5 text-[9.5px]">BILLING CONTACT</div>
              <div className="text-[13px]">Maya Rahmawati</div>
              <div className="mono text-text-2 text-[11.5px]">maya@ataka.id</div>
              <div className="mono text-text-2 text-[11.5px]">NPWP: 12.345.678.9-012.000</div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
