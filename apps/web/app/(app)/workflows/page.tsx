import type { ReactNode } from 'react';
import { Button, I, Pill, Placeholder } from '@zosmed/ui';
import { getWorkflowBuilder } from '@/lib/mock/api';
import { PageHeader } from '../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../_components/PageHeaderBreadcrumb';
import { Palette } from './_components/Palette';
import { FlowCanvas } from './_components/FlowCanvas';

const TABS = ['Build', 'Test', 'Logs'];
const DELAYS = ['0s', '5s', '15s', '30s', '60s'];

export default async function WorkflowsPage() {
  const data = await getWorkflowBuilder();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: data.name }]} />
          <Pill tone="lime">{data.status}</Pill>
          <span className="mono text-text-3 text-[11px]">{data.meta}</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="bg-bg-2 border-line flex rounded-lg border p-0.5">
            {TABS.map((t, i) => (
              <span
                key={t}
                className={`rounded-md px-3 py-[5px] text-xs ${i === 0 ? 'bg-bg-3 text-text' : 'text-text-2'}`}
              >
                {t}
              </span>
            ))}
          </div>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Save draft
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            Publish <I.arrow />
          </Button>
        </div>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        <Palette sections={data.palette} />

        {/* Canvas */}
        <div className="bg-bg relative flex-1 overflow-hidden">
          <div
            className="absolute inset-0"
            style={{ backgroundImage: 'radial-gradient(#1f1f23 1px, transparent 1px)', backgroundSize: '20px 20px' }}
          />
          <FlowCanvas nodes={data.nodes} links={data.links} />

          {/* run banner */}
          <div className="bg-bg-2 border-line absolute left-4 top-4 flex items-center gap-2.5 rounded-full border px-3.5 py-1.5">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: 'var(--zz-lime)', animation: 'zz-pulse 1.4s infinite' }} />
            <span className="mono text-xs">{data.runs}</span>
            <span className="bg-line h-3 w-px" />
            <span className="mono text-text-3 text-[11px]">{data.errorRate}</span>
          </div>

          {/* minimap */}
          <div className="bg-bg-2 border-line absolute bottom-4 right-4 rounded-lg border p-2" style={{ width: 180, height: 110 }}>
            <div className="mono tracked text-text-3 mb-1 text-[9px]">MINIMAP</div>
            <div className="bg-bg-3 relative w-full rounded" style={{ height: 80 }}>
              {[[10, 10], [40, 5], [40, 30], [70, 15], [70, 40], [100, 25], [130, 25]].map(([x, y], i) => (
                <span
                  key={i}
                  className="absolute rounded-[1px]"
                  style={{ left: x, top: y, width: 18, height: 6, background: i === 6 ? 'var(--zz-lime)' : 'var(--zz-line-2)' }}
                />
              ))}
              <div className="absolute rounded-[3px]" style={{ left: 8, top: 4, width: 60, height: 50, border: '1px solid var(--zz-lime)' }} />
            </div>
          </div>

          {/* zoom controls */}
          <div className="bg-bg-2 border-line absolute bottom-4 left-4 flex gap-1 rounded-lg border p-1">
            {['−', '100%', '+', '⛶'].map((t) => (
              <span key={t} className="mono text-text-2 rounded px-2.5 py-1 text-xs">
                {t}
              </span>
            ))}
          </div>
        </div>

        {/* Inspector */}
        <div className="border-bg-3 w-[320px] overflow-y-auto border-l px-[18px] py-5">
          <div className="mb-1 flex items-center gap-2">
            <span
              className="inline-flex h-7 w-7 items-center justify-center rounded-md"
              style={{ background: 'oklch(0.78 0.2 0 / 0.14)', color: 'var(--zz-pink)' }}
            >
              <I.send />
            </span>
            <span className="mono tracked text-[10px]" style={{ color: 'var(--zz-pink)' }}>
              ACTION · NODE 04
            </span>
          </div>
          <h3 className="mb-1 mt-1.5 text-lg font-medium">Send DM</h3>
          <p className="text-text-3 m-0 text-xs">Kirim direct message ke user yang lulus filter sebelumnya.</p>

          <div className="bg-bg-3 my-5 h-px" />

          <Field label="MESSAGE TEMPLATE">
            <div className="bg-bg-2 border-line rounded-lg border p-3 text-[13px] leading-normal">
              Hai{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: 'var(--zz-blue)' }}>
                {'{{first_name}}'}
              </span>
              ! 👋
              <br />
              Makasih udah komen di post kami. Ini link produk yang kamu tanyain:
              <br />
              <span className="text-lime">→ ataka.id/promo-mei</span>
              <br />
              <br />
              Pakai kode{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.9 0.2 130 / 0.18)', color: 'var(--zz-lime)' }}>
                MEI20
              </span>{' '}
              untuk diskon 20% ✨
            </div>
          </Field>

          <Field label="ATTACHMENTS">
            <div className="flex gap-1.5">
              <Placeholder label="product.jpg" height={64} style={{ flex: 1 }} />
              <span className="border-line-2 text-text-3 inline-flex h-16 w-16 items-center justify-center rounded-md border border-dashed">
                <I.plus />
              </span>
            </div>
          </Field>

          <Field label="DELAY BEFORE SEND">
            <div className="flex gap-1.5">
              {DELAYS.map((t, i) => (
                <span
                  key={t}
                  className="mono flex-1 rounded-md py-1.5 text-center text-xs"
                  style={
                    i === 2
                      ? { background: 'var(--zz-lime)', color: 'var(--zz-bg)' }
                      : { background: 'var(--zz-bg-2)', color: 'var(--zz-text-2)', border: '1px solid var(--zz-line)' }
                  }
                >
                  {t}
                </span>
              ))}
            </div>
            <span className="mono text-text-3 mt-1.5 block text-[10.5px]">Manusiawi — hindari deteksi spam IG.</span>
          </Field>

          <Field label="ON SUCCESS">
            <div className="bg-bg-2 border-line flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs">
              <span className="h-1.5 w-1.5 rounded-full" style={{ background: 'var(--zz-lime)' }} />
              <span>Continue → AI Follow-up</span>
            </div>
          </Field>

          <Field label="ON FAIL">
            <div className="bg-bg-2 border-line flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs">
              <span className="h-1.5 w-1.5 rounded-full" style={{ background: 'var(--zz-warn)' }} />
              <span className="text-text-2">Retry 2x, then log + alert</span>
            </div>
          </Field>
        </div>
      </div>
    </>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="mb-[18px]">
      <div className="mono tracked text-text-3 mb-2 text-[9.5px]">{label}</div>
      {children}
    </div>
  );
}
