import { Button, I, Pill } from '@zosmed/ui';
import { getSafety } from '@/lib/mock/api';
import { PageHeader } from '../_components/PageHeader';

export default async function SafetyPage() {
  const data = await getSafety();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <span className="text-sm font-medium">Safety center</span>
          <Pill tone="lime">● HEALTHY</Pill>
        </div>
        <span className="mono text-text-3 text-[11px]">0 incidents · last 30 days</span>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <div className="mb-6">
          <h1 className="m-0 text-3xl font-medium tracking-tight">Safety &amp; rate limits</h1>
          <p className="text-text-2 m-0 mt-1.5 max-w-[640px] text-sm">
            Pengaturan untuk jaga akun IG-mu aman dari deteksi spam. Default sudah konservatif — sesuaikan kalau kamu butuh throughput lebih.
          </p>
        </div>

        <div className="mb-3.5 grid gap-3.5" style={{ gridTemplateColumns: '1.3fr 1fr' }}>
          {/* Rate limits */}
          <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="m-0 text-base font-medium">Rate limits</h3>
              <span className="mono text-text-3 text-[11px]">per IG account</span>
            </div>
            {data.rateLimits.map((s) => (
              <div key={s.l} className="mb-3.5">
                <div className="mb-1.5 flex justify-between text-[13px]">
                  <span>{s.l}</span>
                  <span className="mono">
                    <span className="text-text">{s.v.toLocaleString()}</span>
                    <span className="text-text-3"> / {s.cap.toLocaleString()}</span>
                  </span>
                </div>
                <div className="bg-bg-3 relative rounded-[3px]" style={{ height: 6 }}>
                  <div className="rounded-[3px]" style={{ width: `${Math.min(100, (s.v / s.cap) * 100)}%`, height: 6, background: 'var(--zz-lime)' }} />
                  <div className="absolute" style={{ left: '80%', top: -2, width: 1, height: 10, background: 'var(--zz-warn)' }} />
                </div>
                <div className="mono text-text-3 mt-1 text-[10.5px]">{s.rec}</div>
              </div>
            ))}
          </div>

          {/* Right column */}
          <div className="flex flex-col gap-3.5">
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <h3 className="m-0 mb-3 text-base font-medium">Anti-spam patterns</h3>
              {data.antiSpam.map(([l, on, sub]) => (
                <div key={l} className="flex items-center gap-3 py-2.5" style={{ borderTop: '1px solid #1a1a1d' }}>
                  <span className="relative flex-shrink-0 rounded-full" style={{ width: 32, height: 18, background: on ? 'var(--zz-lime)' : '#2a2a2e' }}>
                    <span className="bg-bg absolute rounded-full" style={{ top: 2, left: on ? 16 : 2, width: 14, height: 14 }} />
                  </span>
                  <div className="flex-1">
                    <div className="text-[13px]">{l}</div>
                    <div className="mono text-text-3 text-[10.5px]">{sub}</div>
                  </div>
                </div>
              ))}
            </div>

            <div className="bg-bg-2 rounded-xl p-[22px]" style={{ border: '1px solid oklch(0.85 0.16 75 / 0.4)' }}>
              <div className="mb-2.5 flex items-center gap-2.5">
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-md" style={{ background: 'oklch(0.85 0.16 75 / 0.18)', color: 'var(--zz-warn)' }}>
                  <I.shield />
                </span>
                <h3 className="m-0 text-sm font-medium">Kill switch</h3>
              </div>
              <p className="text-text-2 m-0 mb-3 text-[12.5px] leading-normal">
                Pause semua workflow untuk akun ini secara instan. Berguna kalau kamu lihat anomaly atau dapet warning dari IG.
              </p>
              <Button variant="ghost" className="w-full justify-center" style={{ borderColor: 'var(--zz-warn)', color: 'var(--zz-warn)' }}>
                ⏸ Pause all workflows
              </Button>
            </div>
          </div>
        </div>

        {/* Activity log */}
        <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
          <div className="mb-3.5 flex items-center justify-between">
            <h3 className="m-0 text-base font-medium">Activity log</h3>
            <span className="mono text-text-3 text-[11px]">last 24 hours · 8 events</span>
          </div>
          {data.log.map((r, i) => (
            <div key={i} className="grid grid-cols-[120px_100px_1fr_100px] items-center gap-3 py-2.5" style={{ borderTop: i ? '1px solid #1a1a1d' : undefined }}>
              <span className="mono text-text-3 text-[11px]">{r[0]}</span>
              <span
                className="mono inline-block justify-self-start rounded-[3px] px-2 py-0.5 text-[11px]"
                style={{ color: r[4], background: `color-mix(in oklch, ${r[4]} 14%, transparent)` }}
              >
                {r[1]}
              </span>
              <span className="text-text-2 text-[13px]">{r[2]}</span>
              <span className="mono text-text-3 text-right text-[11px]">by {r[3]}</span>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
