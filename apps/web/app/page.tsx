import type { ReactNode } from 'react';
import { ChatBubble, Dot, I, Logo, Pill, Placeholder } from '@zosmed/ui';

const NAV = ['Product', 'Workflow', 'Pricing', 'Docs', 'Changelog'];
const BRANDS = ['ATAKA', 'folkstudio', 'ekuator', 'RUMAHKEBUN', 'noir.co', 'PIPA'];
const HERO_STATS = [
  ['12,847', 'comments processed'],
  ['9,213', 'DMs sent'],
  ['2,341', 'leads captured'],
  ['Rp 187jt', 'revenue tracked'],
];

const KITS = [
  {
    tag: 'SELLER KIT', title: 'Buat yang jualan', accent: 'var(--zz-lime)', iconKey: 'box' as const, thesis: 'Comment → Cash',
    items: ['Comment-to-order (keep / C)', 'Trust-kit anti-penipuan', 'Commerce calendar (gajian, Harbolnas)', 'Handoff checkout ke WhatsApp'], live: true,
  },
  {
    tag: 'CREATOR KIT', title: 'Buat yang edukasi', accent: 'var(--zz-blue)', iconKey: 'sparkle' as const, thesis: 'Comment → Audience',
    items: ['Lead-magnet delivery (PDF/ebook)', 'Link-in-DM otomatis', 'Waitlist & pre-launch', 'Handoff ke newsletter / komunitas', 'Kode afiliasi'], live: false,
  },
  {
    tag: 'BOOKING KIT', title: 'Buat yang jasa', accent: 'var(--zz-warn)', iconKey: 'calendar' as const, thesis: 'Comment → Booking',
    items: ['Komentar → jadwal janji temu', 'Handoff ke WhatsApp / kalender', 'Reminder & konfirmasi', 'Slot & ketersediaan'], live: false,
  },
];

export default function LandingPage() {
  return (
    <div className="bg-bg text-text relative overflow-hidden">
      {/* Nav */}
      <div className="border-bg-3 flex items-center justify-between border-b px-14 py-6">
        <Logo size={24} />
        <div className="text-text-2 flex gap-7 text-sm">
          {NAV.map((n) => (
            <span key={n}>{n}</span>
          ))}
        </div>
        <div className="flex items-center gap-2.5">
          <span className="text-text-2 text-[13px]">Sign in</span>
          <button className="btn-lime px-3.5 py-2 text-[13px]">
            Mulai gratis <I.arrow />
          </button>
        </div>
      </div>

      {/* Hero */}
      <div className="relative px-14 pb-[60px] pt-20">
        <div
          className="absolute inset-0 opacity-70"
          style={{
            backgroundImage: 'linear-gradient(#1a1a1d 1px, transparent 1px), linear-gradient(90deg, #1a1a1d 1px, transparent 1px)',
            backgroundSize: '56px 56px',
            maskImage: 'radial-gradient(ellipse 70% 60% at 50% 30%, black 30%, transparent 80%)',
            WebkitMaskImage: 'radial-gradient(ellipse 70% 60% at 50% 30%, black 30%, transparent 80%)',
          }}
        />
        <div className="relative">
          <div className="bg-bg-2 border-line inline-flex items-center gap-2 rounded-full border px-3 py-1.5">
            <Dot /> <span className="mono tracked text-text-2 text-[11px]">v0.4 — AI replies live</span>
          </div>
          <h1 className="m-0 mt-6 max-w-[1000px] font-semibold" style={{ fontSize: 96, lineHeight: 0.96, letterSpacing: '-0.04em' }}>
            Ubah komentar Instagram
            <br />
            jadi <span style={{ fontStyle: 'italic', fontFamily: 'serif', fontWeight: 400 }}>hasil</span>{' '}
            <span className="inline-flex items-center rounded-2xl px-[18px]" style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}>
              nyata.
            </span>
          </h1>
          <p className="text-text-2 mt-8 max-w-[660px] text-xl leading-relaxed">
            Satu engine netral untuk auto-reply komentar → DM dalam window resmi Instagram. Pasang <b className="text-text">Kit</b> sesuai jalurmu — jualan, edukasi, atau jasa — didukung AI.
          </p>
          <div className="mt-9 flex items-center gap-3">
            <button className="btn-lime">
              Coba gratis 14 hari <I.arrow />
            </button>
            <button className="btn-ghost">Lihat demo workflow</button>
            <span className="mono text-text-3 ml-2 text-[11px]">NO CARD · IG OAUTH</span>
          </div>

          {/* Hero workflow preview */}
          <div className="bg-bg-2 border-line relative mt-16 rounded-2xl border p-6">
            <div className="mb-5 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Pill tone="lime">● LIVE</Pill>
                <span className="mono text-text-3 text-[11px]">workflows / promo-launch-mei.flow</span>
              </div>
              <div className="flex gap-1.5">
                {[0, 1, 2].map((i) => (
                  <span key={i} className="rounded-full" style={{ width: 10, height: 10, background: '#2a2a2e' }} />
                ))}
              </div>
            </div>
            <MiniFlow />
            <div className="flex justify-between border-t pt-5" style={{ marginTop: 24, borderColor: '#1a1a1d' }}>
              {HERO_STATS.map(([n, l]) => (
                <div key={l}>
                  <div className="mono tnum text-[22px] font-medium">{n}</div>
                  <div className="mono tracked text-text-3 mt-1 text-[10px]">{l}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Logos strip */}
      <div className="border-bg-3 flex items-center justify-between border-y px-14 py-8">
        <span className="mono tracked text-text-3 text-[10px]">Dipakai 1,200+ creator & brand</span>
        {BRANDS.map((b) => (
          <span key={b} className="text-base font-medium" style={{ color: '#555048', letterSpacing: '-0.02em' }}>
            {b}
          </span>
        ))}
      </div>

      {/* Feature grid */}
      <div className="px-14 py-20">
        <SectionHead eyebrow="// fitur" titleA="Stack lengkap untuk" titleItalic="comment-to-cash" desc="Dari trigger pertama sampai conversion. Semua node yang kamu butuhkan, satu kanvas." />
        <div className="grid grid-cols-6 gap-4">
          <FeatureCard span={3} title="Visual Workflow Builder" desc="Drag-and-drop node editor. Trigger → Filter → Action. Bercabang sesuai logika bisnis." icon={<I.workflow />}>
            <Placeholder label="node editor preview" height={160} />
          </FeatureCard>
          <FeatureCard span={3} title="AI Reply Engine" desc="Balas DM dengan tone brand-mu. Trained dari product catalog & FAQ." icon={<I.ai />}>
            <div className="flex flex-col gap-2">
              <ChatBubble side="them" text="info dong sis, harga berapa?" />
              <ChatBubble side="us" ai text="Halo kak! Bundle Mei lagi diskon 20% — Rp 189rb saja. Mau dikirim link checkout?" />
            </div>
          </FeatureCard>
          <FeatureCard span={2} title="Comment → DM Funnel" desc="Auto-reply komentar publik, lalu pindah ke DM untuk closing." icon={<I.chat />} />
          <FeatureCard span={2} title="Keyword Filter" desc="Include/exclude keyword. Per-post targeting. Regex support." icon={<I.filter />} />
          <FeatureCard span={2} title="Safety Limits" desc="Rate limit ~200 DM/jam + queue. 24h window-aware. Anti-spam, IG-compliant." icon={<I.shield />} />
          <FeatureCard span={3} title="Analytics & Attribution" desc="Tracking dari komentar pertama sampai revenue. UTM auto-tagged." icon={<I.chart />}>
            <Placeholder label="conversion funnel chart" height={120} />
          </FeatureCard>
          <FeatureCard span={3} title="Templates Library" desc="50+ workflow siap pakai: launch produk, giveaway, lead magnet, FAQ bot." icon={<I.sparkle />}>
            <div className="flex flex-wrap gap-2">
              {['Launch Produk', 'Giveaway', 'Lead Magnet', 'FAQ Bot', 'Win-back', 'Quiz Funnel'].map((t) => (
                <span key={t} className="border-line-2 text-text-2 rounded-md border px-2.5 py-1.5 text-xs">
                  {t}
                </span>
              ))}
            </div>
          </FeatureCard>
        </div>
      </div>

      {/* Engine + Kits */}
      <div className="px-14 pb-5 pt-10">
        <SectionHead
          eyebrow="// satu engine, banyak jalur"
          titleA="Engine netral."
          titleItalic="Kit"
          titleAfter=" sesuai jalurmu."
          desc="Workflow builder, safety layer, dan AI persona dipakai bersama. Pasang Kit di atasnya — setiap segmen dapat trigger, template, dan node spesialis."
        />
        <div className="grid grid-cols-3 gap-4">
          {KITS.map((k) => (
            <div key={k.tag} className="bg-bg-2 flex flex-col rounded-2xl p-6" style={{ border: `1px solid ${k.live ? k.accent : 'var(--zz-line)'}` }}>
              <div className="mb-4 flex items-center justify-between">
                <span className="inline-flex h-10 w-10 items-center justify-center rounded-[10px]" style={{ background: `color-mix(in oklch, ${k.accent} 16%, transparent)`, color: k.accent }}>
                  {I[k.iconKey]()}
                </span>
                {k.live ? <Pill tone="lime">● LIVE</Pill> : <Pill tone="neutral">SOON</Pill>}
              </div>
              <span className="mono tracked text-[10px]" style={{ color: k.accent }}>
                {k.tag}
              </span>
              <h3 className="m-0 mb-0.5 mt-1.5 text-2xl font-medium tracking-tight">{k.title}</h3>
              <div className="mono text-text-3 mb-4 text-xs">thesis: {k.thesis}</div>
              <div className="mt-auto flex flex-col gap-2">
                {k.items.map((it) => (
                  <div key={it} className="flex items-start gap-2 text-[13px]">
                    <span className="mt-0.5 flex-shrink-0" style={{ color: k.accent }}>
                      <I.check />
                    </span>
                    <span className="text-text-2">{it}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Seller Kit deep-dive */}
      <div className="px-14 pb-5 pt-10">
        <SectionHead
          eyebrow="// seller kit · dibuat untuk olshop indonesia"
          titleA="Yang"
          titleItalic="pemain luar"
          titleAfter=" nggak punya."
          desc="Closing pindah ke WhatsApp, gaya chat olshop, comment-to-order, momen gajian — semua dipahami Zosmed."
        />
        <div className="grid grid-cols-6 gap-4">
          <FeatureCard span={3} title="Handoff ke WhatsApp" desc="DM IG menyodorkan tombol WhatsApp dengan teks terisi otomatis: nama, produk, dari post mana. Closing pindah ke chat tanpa friksi." icon={<I.whatsapp />}>
            <div className="bg-bg flex items-center gap-2.5 rounded-lg px-3 py-2.5" style={{ border: '1px solid #1a1a1d' }}>
              <span className="inline-flex h-[30px] w-[30px] items-center justify-center rounded-[7px]" style={{ background: 'oklch(0.78 0.2 145 / 0.15)', color: 'oklch(0.82 0.2 145)' }}>
                <I.whatsapp />
              </span>
              <div className="min-w-0 flex-1">
                <div className="text-[12.5px] font-medium">Lanjut ke WhatsApp →</div>
                <div className="mono text-text-3 truncate text-[10px]">wa.me/62…?text=Halo, saya Rina mau order Sage M dari post promo-mei</div>
              </div>
            </div>
          </FeatureCard>
          <FeatureCard span={3} title="AI bahasa olshop" desc={'Fasih gaya chat olshop — "ready ga", "PO brp lama", "nego dong" — code-switching, persona kak/sis. Bukan bot generik.'} icon={<I.ai />}>
            <div className="flex flex-col gap-2">
              <ChatBubble side="them" text="sis nego dong, ambil 2 sage M 😬" />
              <ChatBubble side="us" ai text="Hehe boleh kak 🙏 ambil 2 aku kasih 175rb/pcs ya (dari 189rb). Deal? Aku kirim link checkout 🛒" />
            </div>
          </FeatureCard>
          <FeatureCard span={2} title="Trust-kit anti-penipuan" desc={'Deteksi "ini real kak?" / "ga tipu2 kan?" → auto-kirim testimoni, real-pict, bukti resi.'} icon={<I.shield />}>
            <div className="flex gap-1.5">
              <Placeholder label="testi" height={48} style={{ flex: 1 }} />
              <Placeholder label="real-pict" height={48} style={{ flex: 1 }} />
              <Placeholder label="resi" height={48} style={{ flex: 1 }} />
            </div>
          </FeatureCard>
          <FeatureCard span={2} title="Comment-to-order (keep / C)" desc={'Pelanggan komen kode "keep" / "C1" di post/Reel → auto private-reply, tahan stok + countdown, closing di WhatsApp.'} icon={<I.box />}>
            <div className="flex items-center gap-2">
              <Pill tone="lime">@budi: &quot;keep C3&quot;</Pill>
              <span className="mono text-lime ml-auto text-[11px]">stok ditahan 5:00</span>
            </div>
          </FeatureCard>
          <FeatureCard span={2} title="Autopilot kalender ID" desc="Auto-nembak momen belanja: tanggal gajian, Harbolnas 9.9–12.12, Ramadan/THR — via notify opt-in." icon={<I.calendar />}>
            <div className="flex flex-wrap gap-1.5">
              {['Gajian 25', 'Harbolnas 12.12', 'Ramadan', 'THR Lebaran'].map((t) => (
                <span key={t} className="mono border-line-2 text-text-2 rounded-md border px-2 py-1 text-[11.5px]">
                  {t}
                </span>
              ))}
            </div>
          </FeatureCard>
        </div>
      </div>

      {/* CTA banner */}
      <div className="px-14 pb-20 pt-[60px]">
        <div className="relative overflow-hidden rounded-[20px]" style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)', padding: '64px 56px' }}>
          <div className="flex items-end justify-between">
            <div>
              <span className="mono tracked text-[11px]" style={{ opacity: 0.6 }}>
                {'// mulai sekarang'}
              </span>
              <h2 className="m-0 mt-3 max-w-[700px] font-medium" style={{ fontSize: 72, lineHeight: 0.95, letterSpacing: '-0.03em' }}>
                Setup pertama
                <br />
                dalam 4 menit.
              </h2>
              <p className="mt-5 max-w-[480px] text-[17px]" style={{ opacity: 0.7 }}>
                Connect IG, pilih template, aktifkan. Tidak perlu coding, tidak perlu admin tambahan.
              </p>
            </div>
            <div className="flex gap-2.5">
              <button className="inline-flex items-center gap-2 rounded-[10px] px-[22px] py-3.5 text-[15px] font-medium" style={{ background: 'var(--zz-bg)', color: 'var(--zz-text)' }}>
                Mulai gratis <I.arrow />
              </button>
              <button className="rounded-[10px] px-[22px] py-3.5 text-[15px]" style={{ background: 'transparent', color: 'var(--zz-bg)', border: '1px solid var(--zz-bg)' }}>
                Book demo
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="border-bg-3 text-text-3 flex justify-between border-t px-14 py-8 text-xs">
        <span>© 2026 Zosmed — Made in Indonesia</span>
        <span className="mono">status · privacy · terms · careers</span>
      </div>
    </div>
  );
}

function SectionHead({ eyebrow, titleA, titleItalic, titleAfter, desc }: { eyebrow: string; titleA: string; titleItalic: string; titleAfter?: string; desc: string }) {
  return (
    <div className="mb-12 flex items-end justify-between">
      <div>
        <span className="mono tracked text-lime text-[11px]">{eyebrow}</span>
        <h2 className="m-0 mt-3 max-w-[760px] font-medium" style={{ fontSize: 56, lineHeight: 1, letterSpacing: '-0.03em' }}>
          {titleA}
          <br />
          <span style={{ fontStyle: 'italic', fontFamily: 'serif', fontWeight: 400 }}>{titleItalic}</span>
          {titleAfter ?? '.'}
        </h2>
      </div>
      <div className="text-text-2 max-w-[360px] text-[15px] leading-relaxed">{desc}</div>
    </div>
  );
}

function FeatureCard({ span, title, desc, icon, children }: { span: number; title: string; desc: string; icon: ReactNode; children?: ReactNode }) {
  return (
    <div className="bg-bg-2 border-line flex flex-col gap-4 rounded-[14px] border p-6" style={{ gridColumn: `span ${span}`, minHeight: 180 }}>
      <div className="flex items-center gap-2.5">
        <span className="text-lime inline-flex h-8 w-8 items-center justify-center rounded-lg" style={{ background: 'oklch(0.9 0.2 130 / 0.12)' }}>
          {icon}
        </span>
        <span className="mono tracked text-text-3 text-[10px]">{title.split(' ')[0]?.toLowerCase()}.node</span>
      </div>
      <div>
        <h3 className="m-0 text-[22px] font-medium tracking-tight">{title}</h3>
        <p className="text-text-2 m-0 mt-2 text-sm leading-normal">{desc}</p>
      </div>
      {children ? <div className="mt-auto">{children}</div> : null}
    </div>
  );
}

function MiniFlow() {
  const nodes = [
    { x: 0, y: 60, w: 200, type: 'TRIGGER', title: 'IG Comment', sub: 'on post: "promo-mei"', color: 'var(--zz-lime)', iconKey: 'bolt' as const },
    { x: 240, y: 20, w: 180, type: 'FILTER', title: 'Keyword', sub: 'includes: "info"', color: 'var(--zz-warn)', iconKey: 'filter' as const },
    { x: 240, y: 130, w: 180, type: 'FILTER', title: 'Conversation', sub: 'within 24h window', color: 'var(--zz-warn)', iconKey: 'chat' as const },
    { x: 460, y: 20, w: 200, type: 'ACTION', title: 'Reply Comment', sub: '"Cek DM ya 👀"', color: 'var(--zz-pink)', iconKey: 'chat' as const },
    { x: 460, y: 130, w: 200, type: 'ACTION', title: 'Send DM', sub: 'with product link', color: 'var(--zz-pink)', iconKey: 'send' as const },
    { x: 700, y: 75, w: 200, type: 'AI', title: 'AI Follow-up', sub: 'claude-haiku · brand tone', color: 'var(--zz-blue)', iconKey: 'ai' as const },
    { x: 940, y: 75, w: 160, type: 'OUTPUT', title: 'Lead Captured', sub: 'tag: warm-lead', color: 'var(--zz-lime)', iconKey: 'check' as const },
  ];
  const H = 240;
  const links: [number, number][] = [[0, 1], [0, 2], [1, 3], [2, 4], [3, 5], [4, 5], [5, 6]];
  return (
    <div className="relative w-full" style={{ height: H }}>
      <svg width="100%" height={H} className="absolute inset-0">
        {links.map(([a, b], i) => {
          const A = nodes[a];
          const B = nodes[b];
          if (!A || !B) return null;
          const x1 = A.x + A.w;
          const y1 = A.y + 40;
          const x2 = B.x;
          const y2 = B.y + 40;
          const mx = (x1 + x2) / 2;
          return <path key={i} d={`M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`} stroke="var(--zz-line-2)" strokeWidth="1.5" fill="none" />;
        })}
      </svg>
      {nodes.map((n, i) => (
        <div key={i} className="bg-bg-3 absolute overflow-hidden rounded-[10px]" style={{ left: n.x, top: n.y, width: n.w, border: '1px solid #2a2a2e' }}>
          <div className="flex items-center gap-1.5 px-2.5 py-1.5" style={{ background: `color-mix(in oklch, ${n.color} 14%, transparent)`, color: n.color, borderBottom: '1px solid #2a2a2e' }}>
            {I[n.iconKey]()}
            <span className="mono tracked text-[10px]">{n.type}</span>
          </div>
          <div className="px-2.5 py-2">
            <div className="text-[13px] font-medium">{n.title}</div>
            <div className="mono text-text-2 mt-[3px] text-[10.5px]">{n.sub}</div>
          </div>
          <span className="absolute rounded-full" style={{ left: -3, top: 38, width: 6, height: 6, background: 'var(--zz-line-2)' }} />
          <span className="absolute rounded-full" style={{ right: -3, top: 38, width: 6, height: 6, background: n.color }} />
        </div>
      ))}
    </div>
  );
}
