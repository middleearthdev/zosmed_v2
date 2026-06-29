import { ChatBubble, I, Pill, Placeholder } from '@zosmed/ui';
import { PageHeader } from '../../_components/PageHeader';
import { KitIntro } from '../_components/KitParts';

const WA_GREEN = 'oklch(0.82 0.2 145)';
const BLUE = 'var(--zz-blue)';
const LIME = 'var(--zz-lime)';
const AMBER = 'var(--zz-warn)';

const VOICE_TAGS: [string, boolean][] = [
  ['Sapaan kak/sis', true],
  ['Slang olshop', true],
  ['Code-switch ID/EN', true],
  ['Sedikit Jawa/Sunda', true],
  ['Tanggapi nego', true],
  ['Emoji secukupnya', true],
  ['Formal', false],
];

const CALENDAR = [
  { d: '25 Apr', t: 'Tanggal gajian', sub: 'reminder promo ke 2,341 opt-in', st: 'scheduled' },
  { d: '12 May', t: 'Flash sale tengah bulan', sub: 'draft · belum dijadwalkan', st: 'draft' },
  { d: '12.12', t: 'Harbolnas', sub: 'campaign bundle + countdown', st: 'scheduled' },
  { d: 'Ramadan', t: 'THR / Lebaran series', sub: '4 broadcast · H-7 sampai H+3', st: 'scheduled' },
];

function Toggle({ on, accent }: { on: boolean; accent: string }) {
  return (
    <span className="relative ml-auto rounded-full" style={{ width: 32, height: 18, background: on ? accent : '#2a2a2e' }}>
      <span className="bg-bg absolute rounded-full" style={{ top: 2, left: on ? 16 : 2, width: 14, height: 14 }} />
    </span>
  );
}

export default function SellerKitPage() {
  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Seller Kit</span>
        <Pill tone="lime">🇮🇩 INDONESIA-NATIVE</Pill>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <KitIntro
          kit="seller kit · dibuat untuk olshop indonesia"
          accent={LIME}
          title="Fitur lokal yang nempel ke cara jualan kamu"
          sub="Engine & AI persona dipakai bersama Kit lain. Semua aktif via workflow node — tanpa integrasi tambahan, tanpa biaya ekstra."
        />

        <div className="grid grid-cols-2 gap-3.5">
          {/* WhatsApp handoff */}
          <div className="bg-bg-2 border-line rounded-xl border p-5">
            <div className="mb-3.5 flex items-center gap-2.5">
              <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg" style={{ background: 'oklch(0.78 0.2 145 / 0.15)', color: WA_GREEN }}>
                <I.whatsapp />
              </span>
              <div>
                <h3 className="m-0 text-[15px] font-medium">Handoff ke WhatsApp</h3>
                <span className="mono text-text-3 text-[10.5px]">node: kirim link WhatsApp</span>
              </div>
              <Toggle on accent={WA_GREEN} />
            </div>
            <div className="mono tracked text-text-3 mb-1.5 text-[9px]">NOMOR TUJUAN</div>
            <div className="bg-bg mono mb-3 rounded-[7px] px-3 py-2 text-[13px]" style={{ border: '1px solid #1a1a1d' }}>
              +62 812-3456-7890 <span style={{ color: WA_GREEN, marginLeft: 8 }}>● verified</span>
            </div>
            <div className="mono tracked text-text-3 mb-1.5 text-[9px]">TEKS OTOMATIS (PREFILLED)</div>
            <div className="bg-bg rounded-[7px] p-3 text-[12.5px] leading-normal" style={{ border: '1px solid #1a1a1d' }}>
              Halo kak, saya{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: BLUE }}>
                {'{{nama}}'}
              </span>{' '}
              mau order{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.9 0.2 130 / 0.18)', color: LIME }}>
                {'{{produk}}'}
              </span>{' '}
              dari post{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.85 0.16 75 / 0.18)', color: AMBER }}>
                {'{{post}}'}
              </span>
            </div>
            <div className="bg-bg mt-3 flex items-center gap-2 rounded-[7px] px-3 py-2.5" style={{ border: '1px solid #1a1a1d' }}>
              <span className="mono text-text-3 flex-1 truncate text-[10.5px]">wa.me/628123456789?text=Halo%20kak%2C%20saya%20Rina…</span>
              <span className="mono text-lime text-[10.5px]">preview</span>
            </div>
          </div>

          {/* AI olshop persona */}
          <div className="bg-bg-2 border-line rounded-xl border p-5">
            <div className="mb-3.5 flex items-center gap-2.5">
              <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg" style={{ background: 'oklch(0.78 0.16 240 / 0.15)', color: BLUE }}>
                <I.ai />
              </span>
              <div>
                <h3 className="m-0 text-[15px] font-medium">AI bahasa olshop</h3>
                <span className="mono text-text-3 text-[10.5px]">persona di AI Studio</span>
              </div>
              <Pill tone="lime" style={{ marginLeft: 'auto' }}>
                ACTIVE
              </Pill>
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9px]">GAYA BAHASA</div>
            <div className="mb-3.5 flex flex-wrap gap-1.5">
              {VOICE_TAGS.map(([t, on]) => (
                <span
                  key={t}
                  className="rounded-full px-2.5 py-[5px] text-[11.5px]"
                  style={on ? { background: LIME, color: 'var(--zz-bg)' } : { background: 'var(--zz-bg-3)', color: 'var(--zz-text-3)', border: '1px solid var(--zz-line)' }}
                >
                  {t}
                </span>
              ))}
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9px]">CONTOH FEW-SHOT</div>
            <div className="flex flex-col gap-2">
              <ChatBubble side="them" text='"ready ga kak? PO brp lama?"' />
              <ChatBubble side="us" ai text="Ready kak! 😍 Yang ini stok ada, bukan PO — bisa langsung kirim hari ini kalau order sebelum jam 3 sore 🚚" />
            </div>
          </div>

          {/* Trust-kit */}
          <div className="bg-bg-2 border-line rounded-xl border p-5">
            <div className="mb-3.5 flex items-center gap-2.5">
              <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg" style={{ background: 'oklch(0.9 0.2 130 / 0.15)', color: LIME }}>
                <I.shield />
              </span>
              <div>
                <h3 className="m-0 text-[15px] font-medium">Trust-kit anti-penipuan</h3>
                <span className="mono text-text-3 text-[10.5px]">auto-kirim saat intent &quot;ragu&quot;</span>
              </div>
              <Toggle on accent={LIME} />
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9px]">TRIGGER KEYWORD</div>
            <div className="mb-3.5 flex flex-wrap gap-1.5">
              {['"real kak?"', '"ga tipu2 kan"', '"ada testi?"', '"amanah?"', '"COD bisa?"'].map((t) => (
                <span key={t} className="mono border-line-2 text-text-2 rounded-md border px-2.5 py-1 text-[11px]">
                  {t}
                </span>
              ))}
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9px]">ASET YANG DIKIRIM</div>
            <div className="grid grid-cols-3 gap-2">
              <Placeholder label="testi · 12" height={72} />
              <Placeholder label="real-pict · 8" height={72} />
              <Placeholder label="bukti resi · 24" height={72} />
            </div>
          </div>

          {/* Commerce calendar */}
          <div className="bg-bg-2 border-line rounded-xl border p-5">
            <div className="mb-3.5 flex items-center gap-2.5">
              <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg" style={{ background: 'oklch(0.85 0.16 75 / 0.15)', color: AMBER }}>
                <I.calendar />
              </span>
              <div>
                <h3 className="m-0 text-[15px] font-medium">Autopilot kalender commerce</h3>
                <span className="mono text-text-3 text-[10.5px]">scheduler → notify opt-in</span>
              </div>
            </div>
            {CALENDAR.map((e, i) => (
              <div key={i} className="flex items-center gap-3 py-[11px]" style={{ borderTop: i ? '1px solid #1a1a1d' : undefined }}>
                <span className="mono w-[60px] text-[11px]" style={{ color: AMBER }}>
                  {e.d}
                </span>
                <div className="flex-1">
                  <div className="text-[13px]">{e.t}</div>
                  <div className="mono text-text-3 text-[10.5px]">{e.sub}</div>
                </div>
                <Pill tone={e.st === 'scheduled' ? 'lime' : 'neutral'}>{e.st}</Pill>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
