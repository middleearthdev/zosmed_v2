import { ChatBubble, I, Pill } from '@zosmed/ui';
import { PageHeader } from '../../_components/PageHeader';
import { KitCard, KitIntro } from '../_components/KitParts';

const AMBER = 'var(--zz-warn)';
const WA_GREEN = 'oklch(0.82 0.2 145)';

const SLOTS: [string, string, boolean | null][] = [
  ['10:00', 'available', true],
  ['11:30', 'booked · @rina', false],
  ['13:00', 'available', true],
  ['14:30', 'available', true],
  ['16:00', 'held · @budi 4:52', null],
];

const REMINDERS = [
  { d: 'H-1 · 18:00', t: 'Reminder besok', sub: 'auto-DM + WA · "jangan lupa ya kak"', st: 'scheduled' as const },
  { d: 'H · 2 jam', t: 'Reminder hari-H', sub: 'konfirmasi datang? (Y/N)', st: 'scheduled' as const },
  { d: 'H+1', t: 'Follow-up & review', sub: 'minta rating + rebooking', st: 'draft' as const },
];

export default function BookingKitPage() {
  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Booking Kit</span>
        <div className="flex items-center gap-2">
          <Pill tone="neutral">shared engine + AI</Pill>
          <Pill tone="lime">● INSTALLED</Pill>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <KitIntro
          kit="Booking Kit"
          accent={AMBER}
          title="Ubah komentar jadi booking"
          sub="Buat jasa lokal — komentar jadi janji temu, handoff ke WhatsApp / kalender, slot & ketersediaan, reminder otomatis."
        />

        <div className="grid grid-cols-2 gap-3.5">
          <KitCard icon={<I.calendar />} accent={AMBER} title="Komentar → janji temu">
            <div className="mono tracked text-text-3 mb-2 text-[9px]">BALASAN OTOMATIS</div>
            <div className="mb-3 flex flex-col gap-2">
              <ChatBubble side="them" text='"mau booking potong rambut sabtu bisa?"' />
              <ChatBubble
                side="us"
                ai
                accent={AMBER}
                text={
                  <>
                    Bisa kak! Sabtu ada slot:
                    <br />
                    🕙 10:00 · 🕐 13:00 · 🕓 16:00
                    <br />
                    Mau yang mana? Aku kunci slotnya 💈
                  </>
                }
              />
            </div>
            <div className="flex flex-wrap gap-1.5">
              {['"booking"', '"jadwal"', '"reservasi"', '"DP dulu?"'].map((t) => (
                <span key={t} className="mono border-line-2 text-text-2 rounded-md border px-2.5 py-1 text-[11px]">
                  {t}
                </span>
              ))}
            </div>
          </KitCard>

          <KitCard icon={<I.cog />} accent={AMBER} title="Slot & ketersediaan" node="sinkron Google Calendar">
            <div className="mono tracked text-text-3 mb-2.5 text-[9px]">SABTU · 26 APR</div>
            <div className="flex flex-col gap-1.5">
              {SLOTS.map(([t, s, ok]) => (
                <div key={t} className="bg-bg flex items-center gap-2.5 rounded-md px-2.5 py-2" style={{ border: '1px solid #1a1a1d' }}>
                  <span className="mono text-text w-11 text-xs">{t}</span>
                  <span
                    className="rounded-full"
                    style={{ width: 7, height: 7, background: ok === true ? 'var(--zz-lime)' : ok === false ? '#66665f' : AMBER }}
                  />
                  <span className="mono flex-1 text-[11.5px]" style={{ color: ok === true ? 'var(--zz-text-2)' : ok === false ? 'var(--zz-text-3)' : AMBER }}>
                    {s}
                  </span>
                </div>
              ))}
            </div>
          </KitCard>

          <KitCard icon={<I.whatsapp />} accent={AMBER} title="Handoff ke WhatsApp / kalender">
            <div className="bg-bg mb-3 rounded-[7px] p-3 text-[12.5px] leading-normal" style={{ border: '1px solid #1a1a1d' }}>
              Slot <b style={{ color: 'var(--zz-lime)' }}>Sabtu 13:00</b> dikunci untuk{' '}
              <span className="mono rounded-[3px] px-1" style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: 'var(--zz-blue)' }}>
                {'{{nama}}'}
              </span>{' '}
              ✂️
              <br />
              Konfirmasi + lokasi via WhatsApp:
              <br />
              <span className="mt-1 inline-flex items-center gap-1.5" style={{ color: WA_GREEN }}>
                <I.whatsapp /> wa.me/62…?text=Konfirmasi booking Sabtu 13:00
              </span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              <span className="mono rounded px-2 py-1 text-[10.5px]" style={{ background: 'oklch(0.78 0.2 145 / 0.14)', color: WA_GREEN }}>
                + event ke Google Calendar
              </span>
              <span className="mono bg-bg-3 text-text-2 rounded px-2 py-1 text-[10.5px]">hold slot 5 min</span>
            </div>
          </KitCard>

          <KitCard icon={<I.bell />} accent={AMBER} title="Reminder & konfirmasi">
            {REMINDERS.map((e, i) => (
              <div key={i} className="flex items-center gap-3 py-[11px]" style={{ borderTop: i ? '1px solid #1a1a1d' : undefined }}>
                <span className="mono w-[72px] text-[11px]" style={{ color: AMBER }}>
                  {e.d}
                </span>
                <div className="flex-1">
                  <div className="text-[13px]">{e.t}</div>
                  <div className="mono text-text-3 text-[10.5px]">{e.sub}</div>
                </div>
                <Pill tone={e.st === 'scheduled' ? 'lime' : 'neutral'}>{e.st}</Pill>
              </div>
            ))}
          </KitCard>
        </div>
      </div>
    </>
  );
}
