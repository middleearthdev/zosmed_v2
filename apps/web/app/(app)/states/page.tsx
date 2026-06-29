import type { ReactNode } from 'react';
import { Button, I } from '@zosmed/ui';
import { PageHeader } from '../_components/PageHeader';

function Cell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="border-bg-3 relative overflow-hidden rounded-xl border" style={{ background: '#0d0d0d', minHeight: 320 }}>
      <span className="mono tracked text-text-3 absolute left-3.5 top-3 text-[9.5px]">{label}</span>
      <div className="flex h-full flex-col items-center justify-center p-8 text-center">{children}</div>
    </div>
  );
}

export default function StatesPage() {
  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Empty &amp; error states</span>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-7">
        <h1 className="m-0 text-[22px] font-medium">Empty &amp; error states</h1>
        <p className="text-text-2 m-0 mb-[18px] mt-1 text-[13px]">Konten kosong, error, dan zero-state untuk setiap surface utama.</p>

        <div className="grid grid-cols-3 gap-3.5">
          <Cell label="EMPTY · INBOX">
            <div className="relative mb-4 flex h-16 w-16 items-center justify-center rounded-full" style={{ background: 'color-mix(in oklch, var(--zz-lime) 12%, var(--zz-bg-2))' }}>
              <I.chat />
              <span className="absolute rounded-full" style={{ inset: -8, border: '1px dashed var(--zz-lime)', opacity: 0.4 }} />
            </div>
            <h3 className="m-0 text-base font-medium">Inbox kamu kosong 💚</h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Belum ada DM masuk. Aktifkan workflow biar zosmed mulai bantu balas otomatis.
            </p>
            <Button variant="lime" className="px-3.5 py-[7px] text-xs">
              Browse templates →
            </Button>
          </Cell>

          <Cell label="EMPTY · WORKFLOWS">
            <div className="mb-4 flex gap-1.5">
              {['var(--zz-lime)', 'var(--zz-warn)', 'var(--zz-pink)'].map((c, i) => (
                <span key={i} className="bg-bg-2 rounded-[3px]" style={{ width: 32, height: 18, border: `1px dashed ${c}` }} />
              ))}
            </div>
            <h3 className="m-0 text-base font-medium">Belum ada workflow</h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Mulai dari blank canvas atau pilih template — 1 klik clone, edit, publish.
            </p>
            <div className="flex gap-2">
              <Button variant="ghost" className="px-3 py-[7px] text-xs">
                Blank canvas
              </Button>
              <Button variant="lime" className="px-3 py-[7px] text-xs">
                Use template
              </Button>
            </div>
          </Cell>

          <Cell label="ERROR · IG DISCONNECTED">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full text-[22px]" style={{ background: 'oklch(0.78 0.2 0 / 0.14)', color: 'var(--zz-pink)' }}>
              ⚠
            </div>
            <h3 className="m-0 text-base font-medium">Instagram disconnected</h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Token @ataka.studio expired 4 jam lalu. Workflow auto-paused — re-auth untuk lanjut.
            </p>
            <Button variant="lime" className="px-3.5 py-[7px] text-xs">
              Re-authorize Instagram
            </Button>
            <span className="mono text-text-3 mt-2.5 text-[10.5px]">last error: oauth_token_expired · 14:21</span>
          </Cell>

          <Cell label="EMPTY · ANALYTICS">
            <svg width="180" height="60" viewBox="0 0 180 60">
              <line x1="0" y1="50" x2="180" y2="50" stroke="var(--zz-line)" strokeDasharray="3 3" />
              <text x="90" y="32" textAnchor="middle" fontFamily="var(--font-mono)" fontSize="11" fill="#3a3a40">
                no data yet
              </text>
            </svg>
            <h3 className="m-0 mt-3 text-base font-medium">No leads in last 7 days</h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Coba aktifkan workflow lead-gen atau cek date range. Data tampil setelah ada minimal 1 lead.
            </p>
            <Button variant="ghost" className="px-3 py-[7px] text-xs">
              Change date range
            </Button>
          </Cell>

          <Cell label="EMPTY · SEARCH">
            <div className="bg-bg-2 mb-4 flex h-16 w-16 items-center justify-center rounded-full">
              <I.search />
            </div>
            <h3 className="m-0 text-base font-medium">
              No matches for <span className="mono text-lime">&quot;refun&quot;</span>
            </h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Coba kata kunci lain — cek ejaan, atau pakai filter tag/segment.
            </p>
            <div className="mono flex gap-1.5 text-[11px]">
              <span className="text-text-3">did you mean:</span>
              <span className="text-lime">refund</span>
              <span className="text-line-2">·</span>
              <span className="text-lime">return</span>
            </div>
          </Cell>

          <Cell label="WARNING · RATE LIMIT">
            <div className="bg-bg-3 relative mb-4 rounded-[3px]" style={{ width: 240, height: 6 }}>
              <div className="rounded-[3px]" style={{ width: '100%', height: 6, background: 'var(--zz-warn)' }} />
              <span className="mono absolute right-0 text-[11px]" style={{ top: -22, color: 'var(--zz-warn)' }}>
                200 / 200 dm·hr
              </span>
            </div>
            <h3 className="m-0 text-base font-medium">Rate limit reached</h3>
            <p className="text-text-2 m-0 mb-3.5 mt-2 max-w-[280px] text-[12.5px] leading-normal">
              Auto-pacing aktif — workflow lanjut otomatis dalam <span className="mono" style={{ color: 'var(--zz-warn)' }}>23 min</span>. Atau upgrade ke Enterprise untuk limit tinggi.
            </p>
            <Button variant="ghost" className="px-3 py-[7px] text-xs">
              Compare plans
            </Button>
          </Cell>
        </div>
      </div>
    </>
  );
}
