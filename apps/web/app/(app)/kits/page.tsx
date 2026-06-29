import Link from 'next/link';
import { I, Pill } from '@zosmed/ui';
import { PageHeader } from '../_components/PageHeader';

const KITS = [
  {
    tag: 'SELLER KIT',
    href: '/kits/seller',
    title: 'Buat yang jualan',
    thesis: 'Comment → Cash',
    accent: 'var(--zz-lime)',
    iconKey: 'box' as const,
    items: ['Comment-to-order (keep / C)', 'Trust-kit anti-penipuan', 'Commerce calendar (gajian, Harbolnas)', 'Handoff checkout ke WhatsApp'],
    live: true,
  },
  {
    tag: 'CREATOR KIT',
    href: '/kits/creator',
    title: 'Buat yang edukasi',
    thesis: 'Comment → Audience',
    accent: 'var(--zz-blue)',
    iconKey: 'sparkle' as const,
    items: ['Lead-magnet delivery (PDF/ebook)', 'Link-in-DM otomatis', 'Waitlist & pre-launch', 'Handoff ke newsletter / komunitas', 'Kode afiliasi'],
    live: true,
  },
  {
    tag: 'BOOKING KIT',
    href: '/kits/booking',
    title: 'Buat yang jasa',
    thesis: 'Comment → Booking',
    accent: 'var(--zz-warn)',
    iconKey: 'calendar' as const,
    items: ['Komentar → jadwal janji temu', 'Handoff ke WhatsApp / kalender', 'Reminder & konfirmasi', 'Slot & ketersediaan'],
    live: true,
  },
];

export default function KitCenterPage() {
  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Kit Center</span>
        <Pill tone="neutral">1 engine · 3 kits</Pill>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <div className="mb-6">
          <span className="mono tracked text-lime text-[10px]">{'// satu engine, banyak jalur'}</span>
          <h1 className="m-0 mt-1.5 text-3xl font-medium tracking-tight">Kelola Kit aktif</h1>
          <p className="text-text-2 m-0 mt-1.5 max-w-[640px] text-sm">
            Engine netral (workflow, safety, AI persona) dipakai bersama semua Kit. Pasang Kit sesuai segmen — tiap Kit menambah preset node, template, intent, dan aset.
          </p>
        </div>

        <div className="grid grid-cols-3 gap-3.5">
          {KITS.map((k) => (
            <Link
              key={k.tag}
              href={k.href}
              className="bg-bg-2 flex flex-col rounded-2xl p-6 transition-colors"
              style={{ border: `1px solid ${k.live ? k.accent : 'var(--zz-line)'}` }}
            >
              <div className="mb-4 flex items-center justify-between">
                <span
                  className="inline-flex h-10 w-10 items-center justify-center rounded-[10px]"
                  style={{ background: `color-mix(in oklch, ${k.accent} 16%, transparent)`, color: k.accent }}
                >
                  {I[k.iconKey]()}
                </span>
                <Pill tone="lime">● INSTALLED</Pill>
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
            </Link>
          ))}
        </div>
      </div>
    </>
  );
}
