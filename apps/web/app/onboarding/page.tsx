import type { ReactNode } from 'react';
import { ChatBubble, I, Logo, Pill, Placeholder } from '@zosmed/ui';
import { getAccount, getInstagramConnectUrl } from '@/lib/mock/api';
import { ACCOUNT_STATUS_LABEL, ACCOUNT_STATUS_TONE } from '@/lib/account-status';

const STEPS = ['Connect IG', 'Pilih Kit', 'Train AI', 'Go live'];

const SEGMENTS = [
  { t: 'Jualan', k: 'Seller Kit', d: 'comment-to-order, trust-kit, commerce calendar', accent: 'var(--zz-lime)', iconKey: 'box' as const, selected: true, live: true },
  { t: 'Edukasi', k: 'Creator Kit', d: 'lead-magnet, link-in-DM, waitlist, afiliasi', accent: 'var(--zz-blue)', iconKey: 'sparkle' as const, live: false },
  { t: 'Jasa', k: 'Booking Kit', d: 'komentar → janji temu via WA / kalender', accent: 'var(--zz-warn)', iconKey: 'calendar' as const, live: false },
];

export default async function OnboardingPage() {
  const account = await getAccount();
  const connectUrl = getInstagramConnectUrl();
  const connected = account.status === 'connected';

  return (
    <div className="bg-bg text-text min-h-screen p-6">
      {/* Top bar: logo + stepper + skip */}
      <div className="mb-[18px] flex items-center justify-between">
        <Logo size={20} />
        <div className="text-text-2 flex items-center gap-1.5 text-xs">
          {STEPS.map((s, i) => (
            <span key={s} className="flex items-center gap-1.5">
              <span className="flex items-center gap-1.5" style={{ color: i === 0 ? 'var(--zz-lime)' : 'var(--zz-text-2)' }}>
                <span
                  className="mono inline-flex items-center justify-center rounded-full text-[10px]"
                  style={{ width: 18, height: 18, background: i === 0 ? 'var(--zz-lime)' : 'var(--zz-bg-3)', color: i === 0 ? 'var(--zz-bg)' : 'var(--zz-text-2)' }}
                >
                  {i + 1}
                </span>
                {s}
              </span>
              {i < STEPS.length - 1 ? <span className="bg-line" style={{ width: 18, height: 1 }} /> : null}
            </span>
          ))}
        </div>
        <span className="mono text-text-3 text-[11px]">skip · 4 min total</span>
      </div>

      <div className="grid grid-cols-2 gap-3.5">
        {/* Step 1 */}
        <StepCard num="01" title="Hubungkan Instagram" sub="OAuth resmi via Instagram Login. Scope baca & balas komentar/DM.">
          <div className="bg-bg border-line rounded-[10px] border p-4">
            <div className="mb-3.5 flex items-center gap-3">
              <Placeholder label="ig logo" height={44} style={{ width: 44 }} />
              <div>
                <div className="text-sm font-medium">Otorisasi Zosmed</div>
                <div className="mono text-text-3 text-[11px]">3 izin akses · bisa dicabut kapan saja</div>
              </div>
              <Pill tone="lime" style={{ marginLeft: 'auto' }}>
                VERIFIED
              </Pill>
            </div>
            {['Baca komentar & DM', 'Balas komentar & kirim DM', 'Kelola aturan otomatis'].map((l) => (
              <div key={l} className="text-text-2 flex items-center gap-2 py-1.5 text-xs">
                <span className="text-lime inline-flex h-4 w-4 items-center justify-center rounded-full" style={{ background: 'oklch(0.9 0.2 130 / 0.18)' }}>
                  <I.check />
                </span>
                {l}
              </div>
            ))}

            <div className="border-bg-3 mt-3.5 flex items-center gap-2 border-t pt-3.5">
              <Pill tone={ACCOUNT_STATUS_TONE[account.status]}>{ACCOUNT_STATUS_LABEL[account.status].toUpperCase()}</Pill>
              {connected ? (
                <span className="text-sm">
                  <span className="font-medium">{account.displayName}</span>{' '}
                  <span className="mono text-text-3 text-[12px]">@{account.handle}</span>
                </span>
              ) : null}
            </div>

            {connected ? (
              <div className="text-text-2 mt-2.5 text-xs">Akun sudah terhubung — lanjut ke langkah berikutnya.</div>
            ) : (
              <a href={connectUrl} className="btn-lime mt-3.5 w-full justify-center">
                {account.status === 'expired' ? 'Hubungkan ulang Instagram' : 'Hubungkan Instagram'} <I.arrow />
              </a>
            )}
          </div>
        </StepCard>

        {/* Step 2 */}
        <StepCard num="02" title="Kamu jualan, edukasi, atau jasa?" sub="Pilih jalur — Zosmed muat Kit yang relevan. Engine & AI tetap sama.">
          <div className="flex flex-col gap-2">
            {SEGMENTS.map((c) => (
              <div
                key={c.t}
                className="relative flex items-center gap-3 rounded-[10px] p-3"
                style={{
                  background: c.selected ? 'oklch(0.9 0.2 130 / 0.08)' : 'var(--zz-bg)',
                  border: `1px solid ${c.selected ? 'var(--zz-lime)' : 'var(--zz-line)'}`,
                }}
              >
                <span
                  className="inline-flex h-[34px] w-[34px] flex-shrink-0 items-center justify-center rounded-lg"
                  style={{ background: `color-mix(in oklch, ${c.accent} 16%, transparent)`, color: c.accent }}
                >
                  {I[c.iconKey]()}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 text-sm font-medium">
                    {c.t} <span className="mono text-[10px]" style={{ color: c.accent }}>{c.k}</span>
                  </div>
                  <div className="mono text-text-3 mt-0.5 text-[10.5px]">{c.d}</div>
                </div>
                {c.live ? <Pill tone="lime">LIVE</Pill> : <Pill tone="neutral">SOON</Pill>}
                {c.selected ? (
                  <span className="inline-flex h-[18px] w-[18px] flex-shrink-0 items-center justify-center rounded-full" style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}>
                    <I.check />
                  </span>
                ) : null}
              </div>
            ))}
          </div>
        </StepCard>

        {/* Step 3 */}
        <StepCard num="03" title="Train AI dengan brand voice" sub="Upload katalog produk + FAQ. Tone otomatis disesuaikan.">
          <div className="bg-bg border-line rounded-[10px] border p-3.5">
            <div className="mono tracked text-text-3 mb-2 text-[9.5px]">BRAND TONE</div>
            <div className="mb-3.5 flex gap-1.5">
              {[['Friendly', true], ['Professional', false], ['Casual', true], ['Formal', false], ['Playful', true]].map(([t, on]) => (
                <span
                  key={t as string}
                  className="rounded-full px-2.5 py-[5px] text-xs"
                  style={on ? { background: 'var(--zz-lime)', color: 'var(--zz-bg)' } : { background: 'var(--zz-bg-3)', color: 'var(--zz-text-2)', border: '1px solid var(--zz-line)' }}
                >
                  {t}
                </span>
              ))}
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9.5px]">KNOWLEDGE BASE</div>
            <div className="flex flex-col gap-1.5">
              {[['products.csv', '128 SKU', true], ['faq.md', '47 Q&A', true], ['policies.pdf', '12 pages', false]].map(([f, m, ok]) => (
                <div key={f as string} className="bg-bg-3 flex items-center gap-2.5 rounded-md px-2.5 py-2 text-xs">
                  <span className="mono text-lime">{ok ? '✓' : '↑'}</span>
                  <span className="mono text-text flex-1">{f}</span>
                  <span className="mono text-text-3 text-[10.5px]">{m}</span>
                </div>
              ))}
              <div className="mono border-line-2 text-text-3 rounded-md border border-dashed p-2.5 text-center text-[11.5px]">
                + drag &amp; drop more files
              </div>
            </div>
          </div>
        </StepCard>

        {/* Step 4 */}
        <StepCard num="04" title="Test & go live" sub="Simulasi komentar, lalu publish. Pause kapan aja dari dashboard.">
          <div className="bg-bg border-line rounded-[10px] border p-3.5">
            <div className="mb-2.5 flex items-center justify-between">
              <span className="mono tracked text-text-3 text-[9.5px]">SIMULATOR</span>
              <Pill tone="lime">PASS · 4/4</Pill>
            </div>
            <div className="mb-3 flex flex-col gap-1.5">
              <ChatBubble side="them" text='Tester: "info dong sis"' />
              <ChatBubble side="us" text='Comment reply: "Cek DM ya 👀"' />
              <ChatBubble side="us" ai text="DM: Halo kak Tester! Ini detail produknya..." />
            </div>
            <div className="bg-bg-3 my-3 h-px" />
            <div className="flex items-center justify-between">
              <div>
                <div className="text-[13px] font-medium">Siap publish?</div>
                <div className="mono text-text-3 text-[10.5px]">Akan aktif di 4 post · safety limits ON</div>
              </div>
              <button className="btn-lime">
                <I.bolt /> Go live
              </button>
            </div>
          </div>
        </StepCard>
      </div>
    </div>
  );
}

function StepCard({ num, title, sub, children }: { num: string; title: string; sub: string; children: ReactNode }) {
  return (
    <div className="bg-bg-2 border-line rounded-2xl border p-[18px]">
      <div className="mb-2.5 flex items-baseline gap-2.5">
        <span className="mono text-lime text-[11px]">{num}</span>
        <h3 className="m-0 text-lg font-medium tracking-tight">{title}</h3>
      </div>
      <p className="text-text-2 m-0 mb-3.5 text-[12.5px]">{sub}</p>
      {children}
    </div>
  );
}
