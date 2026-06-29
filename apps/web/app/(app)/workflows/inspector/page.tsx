import { Button, Pill } from '@zosmed/ui';
import { getWorkflowInspector } from '@/lib/mock/api';
import type { FlowLink } from '@/lib/mock/workflows';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';
import { InspectorCanvas } from '../_components/InspectorCanvas';

const LINKS: FlowLink[] = [
  { from: 'i1', to: 'i2' },
  { from: 'i2', to: 'i3' },
  { from: 'i3', to: 'i4', active: true },
  { from: 'i3', to: 'i5' },
  { from: 'i2', to: 'i6' },
];
const TABS = ['Config', 'Logic', 'A/B', 'History'];

export default async function WorkflowInspectorPage() {
  const data = await getWorkflowInspector();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb
            crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: 'launch-promo-mei' }, { label: 'Edit' }]}
          />
          <Pill tone="warn">DRAFT — {data.unpublished} unpublished changes</Pill>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            ↶ Undo
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Test on me
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            Publish changes
          </Button>
        </div>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        {/* Mini canvas */}
        <div
          className="relative flex-1 overflow-hidden"
          style={{ background: '#0d0d0d', backgroundImage: 'radial-gradient(#17171a 1px, transparent 1px)', backgroundSize: '18px 18px' }}
        >
          <div className="absolute left-4 top-4 flex gap-1.5">
            <div className="bg-bg-2 border-line flex items-center overflow-hidden rounded-md border">
              <span className="text-text-2 px-2.5 py-1.5 text-xs">−</span>
              <span className="mono text-text-2 border-line border-x px-2 py-1.5 text-[11px]">100%</span>
              <span className="text-text-2 px-2.5 py-1.5 text-xs">+</span>
            </div>
            <Button variant="ghost" className="px-2.5 py-1.5 text-[11px]">
              Fit
            </Button>
            <Button variant="ghost" className="px-2.5 py-1.5 text-[11px]">
              Auto-layout
            </Button>
          </div>
          <div className="absolute right-4 top-4 flex gap-1.5">
            <Button variant="ghost" className="px-2.5 py-1.5 text-[11px]">
              Minimap
            </Button>
            <Button variant="ghost" className="px-2.5 py-1.5 text-[11px]">
              Lint · 0
            </Button>
          </div>

          <InspectorCanvas nodes={data.nodes} links={LINKS} />

          {/* unpublished diff indicator on the Send DM node */}
          <span
            className="absolute rounded-full"
            style={{ left: 880 + 200 - 6, top: 90 - 6, width: 14, height: 14, background: 'var(--zz-warn)', border: '2px solid #0d0d0d' }}
          />
        </div>

        {/* Inspector panel */}
        <div className="border-bg-3 flex w-[380px] flex-col border-l">
          <div className="border-bg-3 border-b p-[18px]">
            <div className="mono tracked mb-1 text-[9.5px]" style={{ color: 'var(--zz-pink)' }}>
              ACTION · REPLY PUBLIC COMMENT
            </div>
            <h3 className="m-0 mt-1 text-base font-medium">Reply public</h3>
            <div className="mono text-text-3 mt-1 text-[11px]">node_id: act_reply_8a2f · runs in &lt; 500ms</div>
          </div>

          <div className="border-bg-3 flex border-b">
            {TABS.map((t, i) => (
              <span
                key={t}
                className="px-3.5 py-2.5 text-[12.5px]"
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

          <div className="zz-scroll flex flex-1 flex-col gap-4 overflow-y-auto p-[18px]">
            <div>
              <div className="mono tracked text-text-3 mb-1.5 text-[9px]">MESSAGE TEXT</div>
              <div className="bg-bg-2 border-line rounded-lg border p-3 text-[13px] leading-normal">
                Hai kak{' '}
                <span className="mono rounded-[3px] px-1 text-[11.5px]" style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: 'var(--zz-blue)' }}>
                  {'{{first_name}}'}
                </span>
                ! 👋 Cek DM ya, kita kirim detailnya 💚
              </div>
              <div className="mono mt-1.5 flex gap-1.5 text-[11px]">
                {['+ variable', '+ emoji', '+ A/B variant'].map((t) => (
                  <span key={t} className="bg-bg-2 border-line text-text-2 rounded-[3px] border px-[7px] py-[3px]">
                    {t}
                  </span>
                ))}
              </div>
            </div>

            <div>
              <div className="mono tracked text-text-3 mb-1.5 text-[9px]">VARIABLES AVAILABLE</div>
              {data.variables.map(([v, d]) => (
                <div key={v} className="flex items-center justify-between py-1.5 text-[11.5px]" style={{ borderBottom: '1px solid #1a1a1d' }}>
                  <span className="mono" style={{ color: 'var(--zz-blue)' }}>
                    {v}
                  </span>
                  <span className="text-text-3">{d}</span>
                </div>
              ))}
            </div>

            <div>
              <div className="mono tracked text-text-3 mb-1.5 text-[9px]">BEHAVIOR</div>
              {data.behavior.map(([k, v]) => (
                <div key={k} className="flex items-center justify-between py-2" style={{ borderBottom: '1px solid #1a1a1d' }}>
                  <span className="text-xs">{k}</span>
                  <div className="flex items-center gap-2">
                    <span className="mono text-text-2 text-[11px]">{v}</span>
                    <span className="relative inline-block flex-shrink-0 rounded-full" style={{ width: 28, height: 16, background: 'var(--zz-lime)' }}>
                      <span className="bg-bg absolute rounded-full" style={{ top: 2, left: 14, width: 12, height: 12 }} />
                    </span>
                  </div>
                </div>
              ))}
            </div>

            <div className="bg-bg-2 rounded-lg p-3" style={{ border: '1px solid oklch(0.85 0.16 75 / 0.3)' }}>
              <div className="mono tracked text-[9px]" style={{ color: 'var(--zz-warn)' }}>
                UNPUBLISHED CHANGE
              </div>
              <div className="text-text-2 mt-1 text-xs leading-normal">Edited message text · added &quot;💚&quot; · changed by Maya 8 min ago</div>
              <div className="mono mt-2 flex gap-2 text-[11px]">
                <span className="text-lime">Keep change</span>
                <span className="text-line-2">·</span>
                <span className="text-text-2">Revert</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
