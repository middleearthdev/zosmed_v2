import { Button, ChatBubble, I, Pill } from '@zosmed/ui';
import { getAIStudio } from '@/lib/mock/api';
import { STATUS_TONE } from '@/lib/mock/system';
import { PageHeader } from '../_components/PageHeader';

export default async function AIStudioPage() {
  const data = await getAIStudio();

  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">AI Studio</span>
        <div className="flex items-center gap-2">
          <span className="mono text-text-3 text-[11px]">last trained 12 min ago · 187 conversations</span>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Test playground
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs">
            <I.sparkle /> Retrain
          </Button>
        </div>
      </PageHeader>

      <div className="flex-1 overflow-auto p-6">
        <div className="grid gap-3.5" style={{ gridTemplateColumns: '1.3fr 1fr' }}>
          {/* LEFT */}
          <div className="flex flex-col gap-3.5">
            {/* Brand voice */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mono tracked text-text-3 mb-3.5 text-[9.5px]">BRAND VOICE</div>
              <div className="grid grid-cols-2 gap-4">
                {data.brandVoice.map((s) => (
                  <div key={s.l}>
                    <div className="mb-1.5 flex justify-between text-[12.5px]">
                      <span>{s.l}</span>
                      <span className="mono text-lime">{Math.round(s.v * 100)}%</span>
                    </div>
                    <div className="bg-bg-3 relative rounded-[3px]" style={{ height: 6 }}>
                      <div className="rounded-[3px]" style={{ width: `${s.v * 100}%`, height: 6, background: 'var(--zz-lime)' }} />
                      <span className="bg-lime absolute rounded-full" style={{ left: `calc(${s.v * 100}% - 6px)`, top: -3, width: 12, height: 12, border: '2px solid var(--zz-bg)' }} />
                    </div>
                    <div className="mono text-text-3 mt-1.5 flex justify-between text-[10.5px]">
                      <span>{s.lo}</span>
                      <span>{s.hi}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* System prompt */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mb-3.5 flex items-center justify-between">
                <span className="mono tracked text-text-3 text-[9.5px]">SYSTEM PROMPT</span>
                <Pill tone="lime">v3.2</Pill>
              </div>
              <div className="bg-bg mono text-text-2 rounded-lg p-3.5 text-[12.5px] leading-relaxed" style={{ border: '1px solid #1a1a1d' }}>
                <span className="text-lime">Kamu</span> adalah asisten DM untuk{' '}
                <span style={{ color: 'var(--zz-blue)' }}>@ataka.studio</span>,
                <br />
                brand fashion sustainable dari Bandung.
                <br />
                <br />
                <span className="text-text-3">{'// tone'}</span>
                <br />
                Friendly, casual, sedikit playful. Pakai emoji secukupnya 💚.
                <br />
                Bahasa Indonesia santai (kak/sis), boleh switch ke English
                <br />
                kalau user mulai duluan.
                <br />
                <br />
                <span className="text-text-3">{'// rules'}</span>
                <br />
                – Jangan buat janji harga sebelum cek{' '}
                <span className="rounded-[3px] px-1" style={{ background: 'oklch(0.85 0.16 75 / 0.18)', color: 'var(--zz-warn)' }}>
                  {'{{products.csv}}'}
                </span>
                <br />
                – Hand-off ke admin kalau topic = refund/komplain
                <br />
                – Selalu close dengan CTA (link/order/follow)
              </div>
            </div>

            {/* Knowledge base */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mono tracked text-text-3 mb-3.5 text-[9.5px]">KNOWLEDGE BASE · 4 SOURCES</div>
              {data.knowledge.map((f, i) => (
                <div key={f.f} className={`grid grid-cols-[24px_1fr_120px_90px_60px] items-center gap-3 py-2.5 ${i ? 'border-t' : ''}`} style={{ borderColor: '#1a1a1d' }}>
                  <span className="bg-bg-3 mono text-text-2 inline-flex h-5 w-5 items-center justify-center rounded">📄</span>
                  <span className="mono text-[12.5px]">{f.f}</span>
                  <span className="mono text-text-3 text-[11px]">{f.m}</span>
                  <span className="mono text-text-3 text-[11px]">{f.last}</span>
                  <Pill tone={STATUS_TONE[f.status] ?? 'neutral'}>{f.status}</Pill>
                </div>
              ))}
              <div className="mono text-text-3 border-line-2 mt-3 rounded-lg border border-dashed p-3 text-center text-xs">
                + drop file or connect data source (Notion · Sheets · Webhook)
              </div>
            </div>
          </div>

          {/* RIGHT */}
          <div className="flex flex-col gap-3.5">
            {/* Test playground */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mb-3.5 flex items-center justify-between">
                <span className="mono tracked text-text-3 text-[9.5px]">TEST PLAYGROUND</span>
                <Pill tone="blue">model: claude-haiku-4-5</Pill>
              </div>
              <div className="bg-bg flex flex-col gap-2 rounded-lg p-3.5" style={{ border: '1px solid #1a1a1d' }}>
                <ChatBubble side="them" text={'"berapa harga yang sage size M? bisa cod jaksel ga?"'} />
                <ChatBubble
                  side="us"
                  ai
                  text="Hai kak! Yang sage size M Rp 189rb 💚 COD jaksel BISA banget — minimum order Rp 150rb, free dalam ring 1. Mau aku kirim link order?"
                />
                <div className="mt-1 flex gap-1.5">
                  <span className="mono rounded-[3px] px-[7px] py-[3px] text-[10px]" style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: 'var(--zz-blue)' }}>
                    used: products.csv
                  </span>
                  <span className="mono rounded-[3px] px-[7px] py-[3px] text-[10px]" style={{ background: 'oklch(0.85 0.16 75 / 0.18)', color: 'var(--zz-warn)' }}>
                    used: shipping-policy.pdf
                  </span>
                  <span className="mono ml-auto rounded-[3px] px-[7px] py-[3px] text-[10px]" style={{ background: 'oklch(0.9 0.2 130 / 0.18)', color: 'var(--zz-lime)' }}>
                    1.2s · 248 tok
                  </span>
                </div>
              </div>
              <div className="bg-bg mt-3 flex items-center gap-2 rounded-lg p-2.5" style={{ border: '1px solid #1a1a1d' }}>
                <span className="text-text-3 flex-1 text-[13px]">type your test message…</span>
                <Button variant="lime" className="px-3 py-1.5 text-xs">
                  Run <I.send />
                </Button>
              </div>
            </div>

            {/* Eval */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mono tracked text-text-3 mb-3.5 text-[9.5px]">EVAL · LAST 7 DAYS</div>
              <div className="grid grid-cols-2 gap-3">
                {data.evals.map(([k, v, d]) => (
                  <div key={k} className="bg-bg rounded-lg p-3.5" style={{ border: '1px solid #1a1a1d' }}>
                    <div className="mono tracked text-text-3 text-[9.5px]">{k}</div>
                    <div className="mono mt-1 text-2xl font-medium">{v}</div>
                    <div className="mono mt-0.5 text-[11px]" style={{ color: d.startsWith('+') ? 'var(--zz-lime)' : 'var(--zz-pink)' }}>
                      {d}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Recent hand-offs */}
            <div className="bg-bg-2 border-line rounded-xl border p-[22px]">
              <div className="mono tracked text-text-3 mb-3.5 text-[9.5px]">RECENT HAND-OFFS · NEED REVIEW</div>
              {data.handoffs.map(([u, q, r]) => (
                <div key={u} className="flex items-center gap-2.5 py-2.5" style={{ borderTop: '1px solid #1a1a1d' }}>
                  <span className="mono bg-bg-3 text-text-2 inline-flex h-7 w-7 items-center justify-center rounded-full text-[11px]">
                    {u.slice(0, 2).toUpperCase()}
                  </span>
                  <div className="flex-1">
                    <div className="mono text-xs">@{u}</div>
                    <div className="text-text-2 text-xs">
                      {q} · <span className="mono text-[10.5px]" style={{ color: 'var(--zz-warn)' }}>{r}</span>
                    </div>
                  </div>
                  <span className="mono text-lime text-[11px]">Review →</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
