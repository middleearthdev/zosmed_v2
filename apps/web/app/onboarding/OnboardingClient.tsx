'use client';

import { useState, type ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import type { AccountStatus, AppUser, Segment } from '@zosmed/types';
import { ChatBubble, I, Logo, Pill, Placeholder } from '@zosmed/ui';
import { InstagramConnectStatus } from '@/app/_components/InstagramConnectStatus';
import { ConnectedBanner } from '@/app/_components/ConnectedBanner';
import { authErrorMessage, completeOnboarding, setSegment } from '@/lib/auth';

const STEPS = ['Connect IG', 'Pilih Kit', 'Train AI', 'Go live'];

interface SegmentOption {
  value: Segment;
  t: string;
  k: string;
  d: string;
  accent: string;
  iconKey: 'box' | 'sparkle' | 'calendar';
  /** Hanya Seller Kit yang live di MVP (CLAUDE.md §13) — sisanya preview "SOON". */
  live: boolean;
}

const SEGMENTS: SegmentOption[] = [
  { value: 'seller', t: 'Jualan', k: 'Seller Kit', d: 'comment-to-order, trust-kit, commerce calendar', accent: 'var(--zz-lime)', iconKey: 'box', live: true },
  { value: 'creator', t: 'Edukasi', k: 'Creator Kit', d: 'lead-magnet, link-in-DM, waitlist, afiliasi', accent: 'var(--zz-blue)', iconKey: 'sparkle', live: false },
  { value: 'booking', t: 'Jasa', k: 'Booking Kit', d: 'komentar → janji temu via WA / kalender', accent: 'var(--zz-warn)', iconKey: 'calendar', live: false },
];

export interface OnboardingClientProps {
  user: AppUser;
  account: AccountStatus | null;
  connectUrl: string;
  justConnected: boolean;
}

/**
 * Onboarding "pilih segmen → connect IG → selesai" (CLAUDE.md §9, ADR-003
 * §5.2). Step 1 & 2 di bawah ini benar-benar terwire ke backend; step 3
 * ("Train AI") tetap preview desain — belum ada endpoint-nya (di luar scope
 * ADR-003) sehingga tidak diberi aksi apa pun.
 */
export function OnboardingClient({ user, account, connectUrl, justConnected }: OnboardingClientProps) {
  const router = useRouter();
  const [segment, setSegmentState] = useState<Segment | null>(user.segment);
  const [segmentSaving, setSegmentSaving] = useState(false);
  const [segmentError, setSegmentError] = useState<string | null>(null);
  const [completing, setCompleting] = useState(false);
  const [completeError, setCompleteError] = useState<string | null>(null);

  const connected = account?.status === 'connected';
  const ready = Boolean(segment) && connected;

  const stepDone = [connected, Boolean(segment)];
  const currentStepIndex = stepDone.includes(false) ? stepDone.indexOf(false) : STEPS.length - 1;

  async function handleSelectSegment(option: SegmentOption) {
    if (!option.live || option.value === segment || segmentSaving) return;
    setSegmentSaving(true);
    setSegmentError(null);
    const result = await setSegment(option.value);
    setSegmentSaving(false);
    if (!result.ok || !result.data) {
      setSegmentError(authErrorMessage(result.error));
      return;
    }
    setSegmentState(result.data.user.segment);
  }

  async function handleGoLive() {
    if (!ready || completing) return;
    setCompleting(true);
    setCompleteError(null);
    const result = await completeOnboarding();
    setCompleting(false);
    if (!result.ok) {
      setCompleteError(authErrorMessage(result.error));
      return;
    }
    router.push('/dashboard');
    router.refresh();
  }

  const goLiveCaption = ready
    ? 'Akan aktif di 4 post · safety limits ON'
    : !connected
      ? 'Hubungkan Instagram dulu di langkah 1'
      : 'Pilih segmen dulu di langkah 2';

  return (
    <div className="bg-bg text-text min-h-screen p-6">
      {/* Top bar: logo + stepper */}
      <div className="mb-[18px] flex items-center justify-between">
        <Logo size={20} />
        <div className="text-text-2 flex items-center gap-1.5 text-xs">
          {STEPS.map((s, i) => {
            const done = i < 2 && stepDone[i];
            const isCurrent = i === currentStepIndex;
            const active = done || isCurrent;
            return (
              <span key={s} className="flex items-center gap-1.5">
                <span className="flex items-center gap-1.5" style={{ color: active ? 'var(--zz-lime)' : 'var(--zz-text-2)' }}>
                  <span
                    className="mono inline-flex items-center justify-center rounded-full text-[10px]"
                    style={{ width: 18, height: 18, background: active ? 'var(--zz-lime)' : 'var(--zz-bg-3)', color: active ? 'var(--zz-bg)' : 'var(--zz-text-2)' }}
                  >
                    {done ? <I.check width={10} height={10} /> : i + 1}
                  </span>
                  {s}
                </span>
                {i < STEPS.length - 1 ? <span className="bg-line" style={{ width: 18, height: 1 }} /> : null}
              </span>
            );
          })}
        </div>
        <span className="mono text-text-3 text-[11px]">{user.email}</span>
      </div>

      {justConnected ? <ConnectedBanner closeHref="/onboarding" /> : null}

      <div className="grid grid-cols-2 gap-3.5">
        {/* Step 1 — Connect Instagram (real) */}
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

            <div className="border-bg-3 mt-3.5 border-t pt-3.5">
              <InstagramConnectStatus account={account} connectUrl={connectUrl} />
            </div>
          </div>
        </StepCard>

        {/* Step 2 — Pilih segmen (real) */}
        <StepCard num="02" title="Kamu jualan, edukasi, atau jasa?" sub="Pilih jalur — Zosmed muat Kit yang relevan. Engine & AI tetap sama.">
          <div className="flex flex-col gap-2">
            {SEGMENTS.map((c) => {
              const selected = c.value === segment;
              return (
                <button
                  key={c.value}
                  type="button"
                  disabled={!c.live || segmentSaving}
                  aria-pressed={selected}
                  onClick={() => handleSelectSegment(c)}
                  className="relative flex items-center gap-3 rounded-[10px] p-3 text-left"
                  style={{
                    background: selected ? 'oklch(0.9 0.2 130 / 0.08)' : 'var(--zz-bg)',
                    border: `1px solid ${selected ? 'var(--zz-lime)' : 'var(--zz-line)'}`,
                    cursor: c.live ? 'pointer' : 'not-allowed',
                    opacity: c.live ? 1 : 0.6,
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
                  {selected ? (
                    <span className="inline-flex h-[18px] w-[18px] flex-shrink-0 items-center justify-center rounded-full" style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}>
                      <I.check />
                    </span>
                  ) : null}
                </button>
              );
            })}
            {segmentError ? (
              <p role="alert" className="text-pink text-xs">
                {segmentError}
              </p>
            ) : null}
          </div>
        </StepCard>

        {/* Step 3 — preview desain, belum ada endpoint (di luar scope ADR-003) */}
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

        {/* Step 4 — "Go live" memicu POST /onboarding/complete (real) */}
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
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-[13px] font-medium">Siap publish?</div>
                <div className="mono text-text-3 text-[10.5px]">{goLiveCaption}</div>
              </div>
              <button className="btn-lime flex-shrink-0" disabled={!ready || completing} onClick={handleGoLive}>
                <I.bolt /> {completing ? 'Menyelesaikan…' : 'Go live'}
              </button>
            </div>
            {completeError ? (
              <p role="alert" className="text-pink mt-2.5 text-xs">
                {completeError}
              </p>
            ) : null}
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
