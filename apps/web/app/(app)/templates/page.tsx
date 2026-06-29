import { Button, I } from '@zosmed/ui';
import { getTemplates } from '@/lib/mock/api';
import { PageHeader } from '../_components/PageHeader';

export default async function TemplatesPage() {
  const data = await getTemplates();

  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Templates</span>
        <Button variant="lime" className="px-3 py-[7px] text-xs">
          <I.plus /> Submit template
        </Button>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <div className="mb-6 flex items-end justify-between">
          <div>
            <span className="mono tracked text-text-3 text-[10px]">54 TEMPLATES · CURATED BY ZOSMED + COMMUNITY</span>
            <h1 className="m-0 mt-1.5 text-3xl font-medium tracking-tight">Templates library</h1>
            <p className="text-text-2 m-0 mt-1.5 max-w-[540px] text-sm">
              Workflow siap pakai. One-click clone ke workspace, edit sesuai brand, publish.
            </p>
          </div>
          <div className="flex gap-1.5">
            {data.filters.map((t, i) => (
              <span
                key={t}
                className="mono rounded-full px-3 py-1.5 text-[11.5px]"
                style={
                  i === 0
                    ? { background: 'var(--zz-lime)', color: 'var(--zz-bg)' }
                    : { background: 'var(--zz-bg-2)', color: 'var(--zz-text-2)', border: '1px solid var(--zz-line)' }
                }
              >
                {t}
              </span>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3.5">
          {data.templates.map((t, i) => (
            <div key={t.t} className="bg-bg-2 border-line overflow-hidden rounded-xl border">
              <div
                className="border-line relative border-b p-4"
                style={{ height: 120, background: `linear-gradient(135deg, color-mix(in oklch, ${t.color} 24%, transparent), var(--zz-bg))` }}
              >
                <span
                  className="bg-bg inline-flex h-9 w-9 items-center justify-center rounded-lg"
                  style={{ border: `1px solid ${t.color}`, color: t.color }}
                >
                  {I[t.iconKey]()}
                </span>
                {t.tag ? (
                  <span
                    className="mono absolute right-4 top-4 rounded px-2 py-[3px] text-[10px] font-semibold"
                    style={{ background: t.color, color: 'var(--zz-bg)' }}
                  >
                    {t.tag}
                  </span>
                ) : null}
                <div className="mt-[18px] flex items-center gap-1">
                  {[0, 1, 2, 3].map((k) => (
                    <span key={k} className="flex items-center gap-1">
                      <span
                        className="bg-bg rounded-[2px]"
                        style={{ width: 22, height: 12, border: `1px solid ${k === 0 ? t.color : 'var(--zz-line-2)'}` }}
                      />
                      {k < 3 ? <span className="bg-line-2" style={{ width: 8, height: 1 }} /> : null}
                    </span>
                  ))}
                </div>
              </div>
              <div className="p-[18px]">
                <h3 className="m-0 text-base font-medium">{t.t}</h3>
                <p className="text-text-2 m-0 mb-3 mt-1.5 min-h-9 text-[12.5px] leading-normal">{t.d}</p>
                <div className="border-bg-3 flex items-center justify-between border-t pt-2.5">
                  <span className="mono text-text-3 text-[10.5px]">
                    {t.runs} clones · ★ 4.{4 + (i % 6)}
                  </span>
                  <span className="text-lime inline-flex items-center gap-1 text-xs">
                    Use template <I.arrow />
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
