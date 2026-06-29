import { Avatar, Button, I, Pill, Placeholder } from '@zosmed/ui';
import { getCommentOrder } from '@/lib/mock/api';
import { WA_GREEN, type ReservationRow } from '@/lib/mock/workflows';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';

const RES_TONE: Record<ReservationRow['st'], 'lime' | 'pink' | 'warn'> = {
  reserved: 'warn',
  'waiting-pay': 'warn',
  'closed-wa': 'lime',
  expired: 'pink',
};
const QUEUE_GRID = 'grid-cols-[70px_1.4fr_1fr_0.8fr_1fr_90px]';

export default async function CommentToOrderPage() {
  const data = await getCommentOrder();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: 'Comment-to-Order' }]} />
          <Pill tone="lime">● ACTIVE</Pill>
          <span className="mono text-text-3 text-[11px]">trigger: komentar di post/Reel</span>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Keyword settings
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            <I.box /> Stock board
          </Button>
        </div>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        {/* Source post + incoming comments */}
        <div className="border-bg-3 flex w-[420px] flex-col border-r">
          <div className="relative m-4 flex-shrink-0 overflow-hidden rounded-xl">
            <Placeholder label="post / reel — katalog drop" height={220} />
            <div className="absolute left-2.5 top-2.5 flex gap-1.5">
              <Pill tone="lime">CATALOG POST</Pill>
              <span className="mono text-text rounded-full px-2 py-[3px] text-[11px]" style={{ background: '#0a0a0acc' }}>
                {data.postComments}
              </span>
            </div>
            <div className="absolute bottom-2.5 left-2.5 right-2.5 flex items-center gap-1.5">
              <span className="mono rounded-md px-2.5 py-1 text-[11px] font-semibold" style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}>
                komen kode untuk order
              </span>
              <span className="mono text-text ml-auto rounded-md px-2.5 py-1 text-[11px]" style={{ background: '#0a0a0acc' }}>
                @ataka.studio
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between px-4 pb-2">
            <span className="mono tracked text-text-3 text-[9.5px]">KOMENTAR MASUK</span>
            <span className="mono text-[10.5px]" style={{ color: 'var(--zz-lime)' }}>
              auto-detect: keep · C# · 1
            </span>
          </div>
          <div className="zz-scroll flex flex-1 flex-col gap-2 overflow-y-auto px-4 pb-4">
            {data.comments.map((c, i) => (
              <div key={i} className="flex items-start gap-2">
                <Avatar name={c.u.slice(0, 2).toUpperCase()} color="#2a2a2e" size={24} />
                <div className="flex-1">
                  <div className="text-[12.5px]">
                    <span className="mono text-text-2">@{c.u}</span> <span className="text-text">{c.t}</span>
                    <span className="mono text-line-2 ml-1.5 text-[10px]">{c.tm}</span>
                  </div>
                  {c.match ? (
                    <span className="mono mt-0.5 inline-block text-[9.5px]" style={{ color: c.dupe ? 'var(--zz-text-3)' : 'var(--zz-lime)' }}>
                      {c.dupe ? `↳ duplicate of ${c.match} — skipped` : `↳ matched ${c.match} → private reply + reserve`}
                    </span>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Reserved queue + product board */}
        <div className="zz-scroll flex-1 overflow-y-auto p-6">
          <div className="mb-[18px] grid grid-cols-4 gap-2.5">
            {data.stats.map((s) => (
              <div key={s.label} className="bg-bg-2 border-line rounded-[10px] border p-3.5">
                <div className="mono tracked text-text-3 text-[9.5px]">{s.label}</div>
                <div className="mono mt-1 text-[26px] font-medium" style={{ color: s.color }}>
                  {s.value}
                </div>
              </div>
            ))}
          </div>

          <div className="mb-3 flex items-center justify-between">
            <h3 className="m-0 text-base font-medium">Reserved stock queue</h3>
            <span className="mono text-text-3 text-[11px]">auto-release saat countdown habis</span>
          </div>
          <div className="bg-bg-2 border-line mb-[18px] overflow-hidden rounded-xl border">
            <div className={`grid ${QUEUE_GRID} border-line border-b px-4 py-2.5`} style={{ background: '#0d0d0d' }}>
              {['CODE', 'BUYER', 'PRODUCT', 'PRICE', 'STATUS', 'COUNTDOWN'].map((h) => (
                <span key={h} className="mono tracked text-text-3 text-[9.5px]">
                  {h}
                </span>
              ))}
            </div>
            {data.reservations.map((r, i) => (
              <div key={i} className={`grid ${QUEUE_GRID} border-bg-3 items-center gap-2 border-b px-4 py-3 last:border-b-0`}>
                <span className="mono text-xs font-semibold" style={{ color: 'var(--zz-lime)' }}>
                  {r.code}
                </span>
                <div className="flex items-center gap-2">
                  <Avatar name={r.u.slice(0, 2).toUpperCase()} color="#2a2a2e" size={24} />
                  <span className="mono text-xs">@{r.u}</span>
                </div>
                <span className="text-[12.5px]">{r.p}</span>
                <span className="mono tnum text-xs">{r.price}</span>
                <Pill tone={RES_TONE[r.st]}>{r.st}</Pill>
                <span className="mono text-xs" style={{ color: r.color }}>
                  {r.cd}
                </span>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-2 gap-3.5">
            {/* Private reply config */}
            <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
              <div className="mono tracked text-text-3 mb-3 text-[9.5px]">PRIVATE REPLY SAAT KODE TERDETEKSI</div>
              <div className="bg-bg rounded-lg p-3 text-[13px] leading-normal" style={{ border: '1px solid #1a1a1d' }}>
                Yeay{' '}
                <span className="mono" style={{ color: 'var(--zz-blue)' }}>
                  {'{{nama}}'}
                </span>
                ! Kamu keep <b style={{ color: 'var(--zz-lime)' }}>C3 · Sage Tote M</b> 🛍️
                <br />
                Stok ditahan <b style={{ color: 'var(--zz-warn)' }}>5 menit</b> ya. Lanjut checkout via WhatsApp:
                <br />
                <span className="mt-1 inline-flex items-center gap-1.5" style={{ color: WA_GREEN }}>
                  <I.whatsapp /> wa.me/62…?text=Order C3 Sage M
                </span>
              </div>
              <div className="mt-2.5 flex flex-wrap gap-1.5">
                <span className="mono bg-bg-3 text-text-2 rounded px-2 py-1 text-[10.5px]">hold: 5 min</span>
                <span className="mono bg-bg-3 text-text-2 rounded px-2 py-1 text-[10.5px]">reminder: 1 min sebelum habis</span>
                <span className="mono rounded px-2 py-1 text-[10.5px]" style={{ background: 'oklch(0.78 0.2 145 / 0.14)', color: WA_GREEN }}>
                  closing di WhatsApp
                </span>
              </div>
              <div
                className="mt-3 flex items-start gap-2 rounded-[7px] px-2.5 py-2"
                style={{ background: 'oklch(0.78 0.16 240 / 0.08)', border: '1px solid oklch(0.78 0.16 240 / 0.25)' }}
              >
                <span className="mt-px text-xs" style={{ color: 'var(--zz-blue)' }}>
                  ⓘ
                </span>
                <span className="mono text-text-2 text-[10.5px] leading-normal">
                  1 private reply / komentar (window 7 hari). Lanjutan chat di window 24 jam. Hold stok 5 min — aman di dalam window.
                </span>
              </div>
            </div>

            {/* Product catalog */}
            <div className="bg-bg-2 border-line rounded-xl border p-[18px]">
              <div className="mb-3 flex justify-between">
                <span className="mono tracked text-text-3 text-[9.5px]">PRODUK DI POST</span>
                <span className="mono text-text-3 text-[10.5px]">stok realtime</span>
              </div>
              {data.products.map((p) => {
                const ratio = p.left / p.total;
                const barColor = p.left === 0 ? 'var(--zz-pink)' : ratio < 0.3 ? 'var(--zz-warn)' : 'var(--zz-lime)';
                return (
                  <div key={p.code} className="flex items-center gap-2.5 py-2" style={{ borderTop: '1px solid #1a1a1d' }}>
                    <span className="mono w-6 text-[11px] font-semibold" style={{ color: 'var(--zz-lime)' }}>
                      {p.code}
                    </span>
                    <span className="flex-1 text-[12.5px]">{p.name}</span>
                    <div className="bg-bg-3 h-1 rounded-full" style={{ width: 70 }}>
                      <div className="h-1 rounded-full" style={{ width: `${ratio * 100}%`, background: barColor }} />
                    </div>
                    <span className="mono w-[52px] text-right text-[11px]" style={{ color: p.left === 0 ? 'var(--zz-pink)' : 'var(--zz-text-2)' }}>
                      {p.left === 0 ? 'sold out' : `${p.left}/${p.total}`}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
