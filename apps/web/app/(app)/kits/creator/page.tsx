import { ChatBubble, I, Pill } from '@zosmed/ui';
import { PageHeader } from '../../_components/PageHeader';
import { KitCard, KitIntro } from '../_components/KitParts';

const BLUE = 'var(--zz-blue)';

export default function CreatorKitPage() {
  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Creator Kit</span>
        <div className="flex items-center gap-2">
          <Pill tone="neutral">shared engine + AI</Pill>
          <Pill tone="lime">● INSTALLED</Pill>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        <KitIntro
          kit="Creator Kit"
          accent={BLUE}
          title="Ubah komentar jadi audience"
          sub="Buat yang edukasi & bangun komunitas — lead-magnet, link-in-DM, waitlist, handoff ke newsletter, kode afiliasi."
        />

        <div className="grid grid-cols-2 gap-3.5">
          <KitCard icon={<I.send />} accent={BLUE} title="Lead-magnet delivery" node="node: kirim file via DM">
            <div className="mono tracked text-text-3 mb-1.5 text-[9px]">TRIGGER KEYWORD → KIRIM</div>
            <div className="mb-3 flex flex-wrap gap-1.5">
              {['"mau ebook"', '"PDF dong"', '"kirim materinya"'].map((t) => (
                <span key={t} className="mono border-line-2 text-text-2 rounded-md border px-2.5 py-1 text-[11px]">
                  {t}
                </span>
              ))}
            </div>
            <div className="flex flex-col gap-1.5">
              {[['7-hari-konten-plan.pdf', '2.4 MB · 184 kirim'], ['canva-template-pack.zip', '8.1 MB · 92 kirim']].map(([f, m]) => (
                <div key={f} className="bg-bg flex items-center gap-2.5 rounded-md px-2.5 py-2 text-xs" style={{ border: '1px solid #1a1a1d' }}>
                  <span style={{ color: BLUE }}>📄</span>
                  <span className="mono flex-1">{f}</span>
                  <span className="mono text-text-3 text-[10.5px]">{m}</span>
                </div>
              ))}
            </div>
          </KitCard>

          <KitCard icon={<I.chat />} accent={BLUE} title="Link-in-DM otomatis">
            <div className="mono tracked text-text-3 mb-2 text-[9px]">BALASAN OTOMATIS</div>
            <div className="mb-3 flex flex-col gap-2">
              <ChatBubble side="them" text='"link yang di bio mana kak?"' />
              <ChatBubble
                side="us"
                ai
                accent={BLUE}
                text={
                  <>
                    Ini ya kak 👇
                    <br />
                    📺 YouTube: <span style={{ color: 'var(--zz-bg)', fontWeight: 600 }}>yt.com/@kreator</span>
                    <br />
                    📝 Newsletter: <span style={{ color: 'var(--zz-bg)', fontWeight: 600 }}>kreator.substack.com</span>
                    <br />
                    🎁 Free guide: <span style={{ color: 'var(--zz-bg)', fontWeight: 600 }}>kreator.id/guide</span>
                  </>
                }
              />
            </div>
            <div className="flex flex-wrap gap-1.5">
              {['YouTube', 'Newsletter', 'Free guide', '+ link'].map((t, i) => (
                <span
                  key={t}
                  className="mono text-text-2 rounded-md px-2.5 py-1 text-[11px]"
                  style={i === 3 ? { background: 'var(--zz-bg)', border: '1px dashed var(--zz-line-2)' } : { background: 'var(--zz-bg-3)' }}
                >
                  {t}
                </span>
              ))}
            </div>
          </KitCard>

          <KitCard icon={<I.bell />} accent={BLUE} title="Waitlist & pre-launch" node="node: notify opt-in">
            <div className="mb-3 flex items-baseline gap-2">
              <span className="mono text-[30px] font-medium" style={{ color: BLUE }}>
                1,284
              </span>
              <span className="text-text-2 text-[13px]">di waitlist &quot;Kelas Editing&quot;</span>
              <span className="mono text-lime ml-auto text-[11px]">+86 hari ini</span>
            </div>
            <div className="bg-bg rounded-[7px] p-3 text-[12.5px] leading-normal" style={{ border: '1px solid #1a1a1d' }}>
              Komentar <span className="mono" style={{ color: BLUE }}>&quot;WAITLIST&quot;</span> → auto-DM konfirmasi + notify saat kelas dibuka. Broadcast 1×, opt-in only.
            </div>
            <div className="mt-2.5 flex gap-1.5">
              <span className="mono bg-bg-3 text-text-2 rounded px-2 py-1 text-[10.5px]">launch: H-3 reminder</span>
              <span className="mono bg-bg-3 text-text-2 rounded px-2 py-1 text-[10.5px]">cap: 1 broadcast / opt-in</span>
            </div>
          </KitCard>

          <KitCard icon={<I.users />} accent={BLUE} title="Handoff ke newsletter / komunitas">
            <div className="mb-3.5 grid grid-cols-2 gap-2">
              {[['Substack', 'connected', BLUE], ['Telegram grup', 'connected', BLUE], ['Discord', 'available', '#66665f'], ['Mailchimp', 'available', '#66665f']].map(([n, s, c]) => (
                <div key={n} className="bg-bg flex items-center gap-2 rounded-md px-2.5 py-2" style={{ border: '1px solid #1a1a1d' }}>
                  <span className="h-2 w-2 flex-shrink-0 rounded-full" style={{ background: c }} />
                  <span className="flex-1 text-xs">{n}</span>
                  <span className="mono text-text-3 text-[9.5px]">{s}</span>
                </div>
              ))}
            </div>
            <div className="mono tracked text-text-3 mb-2 text-[9px]">KODE AFILIASI</div>
            <div className="bg-bg flex items-center gap-2.5 rounded-[7px] px-3 py-2.5" style={{ border: '1px solid #1a1a1d' }}>
              <span style={{ color: BLUE }}>
                <I.tag />
              </span>
              <span className="mono flex-1 text-[13px]">KREATOR15</span>
              <span className="mono text-text-2 text-[10.5px]">312 klaim · Rp 4.2jt komisi</span>
            </div>
          </KitCard>
        </div>
      </div>
    </>
  );
}
