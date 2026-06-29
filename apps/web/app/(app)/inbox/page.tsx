import { Avatar, Button, I, Pill } from '@zosmed/ui';
import { getInbox } from '@/lib/mock/api';
import type { InboxContact, InboxThread } from '@/lib/mock/inbox';
import { PageHeader } from '../_components/PageHeader';
import { ConvBubble } from './_components/ConvBubble';

const VAR_HL = {
  background: 'oklch(0.78 0.16 240 / 0.18)',
  color: 'var(--zz-blue)',
  padding: '0 4px',
  borderRadius: 3,
  fontFamily: 'var(--font-mono)',
  fontSize: 12,
} as const;

const QUICK_ACTIONS = ['Send pricing', 'Send sizes', 'Ask address', 'Send checkout', 'Mark closed'];

export default async function InboxPage() {
  const data = await getInbox();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">Inbox</span>
          <Pill tone="lime">{data.unreadCount} UNREAD</Pill>
          <span className="mono text-text-3 text-[11px]">{data.aiNote}</span>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Filter
          </Button>
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Mark all read
          </Button>
        </div>
      </PageHeader>

      <div className="flex flex-1 overflow-hidden">
        {/* Thread list */}
        <div className="border-bg-3 flex w-[360px] flex-shrink-0 flex-col overflow-hidden border-r">
          <div className="border-bg-3 border-b p-3.5">
            <div className="bg-bg-2 border-line flex items-center gap-2 rounded-lg border px-2.5 py-1.5">
              <I.search />
              <span className="text-text-3 text-xs">Search messages…</span>
            </div>
            <div className="mt-2.5 flex gap-1.5">
              {data.filters.map((f, i) => (
                <span
                  key={f}
                  className="mono rounded-full px-2.5 py-1 text-[11px]"
                  style={
                    i === 0
                      ? { background: 'var(--zz-lime)', color: 'var(--zz-bg)' }
                      : { background: 'var(--zz-bg-2)', color: 'var(--zz-text-2)', border: '1px solid var(--zz-line)' }
                  }
                >
                  {f}
                </span>
              ))}
            </div>
          </div>
          <div className="zz-scroll flex-1 overflow-y-auto">
            {data.threads.map((th) => (
              <ThreadItem key={th.u} th={th} />
            ))}
          </div>
        </div>

        {/* Conversation */}
        <div className="flex flex-1 flex-col overflow-hidden">
          <div className="border-bg-3 flex flex-shrink-0 items-center gap-3 border-b px-5 py-3.5">
            <Avatar name={data.contact.avatar} color={data.contact.color} size={36} />
            <div className="flex-1">
              <div className="text-sm font-medium">@{data.contact.handle}</div>
              <div className="mono text-text-3 text-[11px]">{data.contact.meta}</div>
            </div>
            <Pill tone="lime">● AI HANDLING</Pill>
            <Button variant="ghost" className="px-2.5 py-1.5 text-[11.5px]">
              Take over
            </Button>
          </div>

          <div className="zz-scroll flex flex-1 flex-col gap-3.5 overflow-y-auto px-6 pb-3 pt-6">
            <div className="mono text-text-3 tracked self-center text-[10.5px]">SELASA · 28 APR · 14:32</div>

            <ConvBubble
              side="them"
              name="rina_susanti"
              time="14:32"
              text="info dong sis, harga berapa?"
              source="comment on @ataka.studio post"
            />
            <ConvBubble side="us" ai time="14:32" tag="reply-comment" text="Hai kak Rina! 👋 Cek DM ya, kita kirim detailnya 💚" />
            <ConvBubble
              side="us"
              ai
              time="14:32"
              tag="dm-template"
              text={
                <>
                  Halo kak <span style={VAR_HL}>{'{{first_name}}'}</span>! 👋
                  <br />
                  Bundle Mei lagi diskon 20% — Rp 189rb saja.
                  <br />
                  Pakai kode{' '}
                  <b
                    style={{
                      background: 'var(--zz-bg)',
                      color: 'var(--zz-lime)',
                      padding: '0 5px',
                      borderRadius: 3,
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                    }}
                  >
                    MEI20
                  </b>{' '}
                  di checkout ✨
                </>
              }
            />
            <ConvBubble side="them" name="rina_susanti" time="14:35" text="Wah lumayan, ini link ke produk yang mana ya?" />

            {/* AI thinking indicator */}
            <div
              className="bg-bg-2 inline-flex max-w-[480px] items-center gap-2 self-start rounded-[10px] px-3 py-2 text-xs"
              style={{ border: '1px solid oklch(0.78 0.16 240 / 0.4)' }}
            >
              <span
                className="inline-flex h-[18px] w-[18px] items-center justify-center rounded-full"
                style={{ background: 'oklch(0.78 0.16 240 / 0.18)', color: 'var(--zz-blue)' }}
              >
                <I.ai />
              </span>
              <span className="mono text-text-2 text-[11px]">
                AI thinking · matching intent: <span style={{ color: 'var(--zz-blue)' }}>ask_product_link</span>
              </span>
              <span className="h-1 w-1 rounded-full" style={{ background: 'var(--zz-blue)', animation: 'zz-pulse 1s infinite' }} />
            </div>

            <ConvBubble
              side="us"
              ai
              suggested
              time="14:35"
              tag="ai-generated"
              text={
                <>
                  Yang lagi promo ini kak:{' '}
                  <a className="text-lime underline" href="#">
                    ataka.id/promo-mei
                  </a>
                  <br />
                  Ada 4 varian warna — sage, terracotta, ivory, midnight 🎨
                  <br />
                  Mau aku kirim foto detailnya?
                </>
              }
            />
          </div>

          {/* Composer */}
          <div className="border-bg-3 flex-shrink-0 border-t p-4">
            <div className="mb-2 flex flex-wrap gap-1.5">
              {QUICK_ACTIONS.map((t) => (
                <span key={t} className="mono bg-bg-2 border-line text-text-2 rounded-full border px-2.5 py-[5px] text-[11px]">
                  ↗ {t}
                </span>
              ))}
            </div>
            <div className="bg-bg-2 border-line rounded-[10px] border p-3">
              <div className="text-text-2 min-h-9 text-[13px]">Type a reply, or let AI continue…</div>
              <div className="mt-2 flex items-center gap-2 border-t pt-2" style={{ borderColor: '#1a1a1d' }}>
                <span className="text-text-3 text-sm">@</span>
                <span className="text-text-3 text-sm">📎</span>
                <span className="text-text-3 text-sm">📷</span>
                <span className="mono text-text-3 ml-auto text-[10.5px]">⌘+⏎ to send</span>
                <Button variant="ghost" icon={<I.ai />} className="px-3 py-1.5 text-xs">
                  Generate AI reply
                </Button>
                <Button variant="lime" className="px-3.5 py-1.5 text-xs">
                  Send <I.send />
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Context panel */}
        <ContextPanel contact={data.contact} />
      </div>
    </>
  );
}

function ThreadItem({ th }: { th: InboxThread }) {
  return (
    <div
      className="border-bg-3 relative flex gap-2.5 border-b px-4 py-3.5"
      style={{
        background: th.active ? 'var(--zz-bg-3)' : 'transparent',
        borderLeft: th.active ? '2px solid var(--zz-lime)' : '2px solid transparent',
      }}
    >
      <Avatar name={th.avatar} color={th.color} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between">
          <span className="mono text-text text-[13px]">@{th.u}</span>
          <span className="mono text-text-3 text-[10.5px]">{th.t}</span>
        </div>
        <div className="text-text-2 mt-0.5 truncate text-[12.5px]">{th.last}</div>
        <div className="mt-1.5 flex items-center gap-1.5">
          {th.ai ? (
            <span className="mono rounded-sm px-1.5 text-[9px]" style={{ color: 'var(--zz-blue)', background: 'oklch(0.78 0.16 240 / 0.12)' }}>
              AI
            </span>
          ) : null}
          {th.tag ? (
            <span className="mono rounded-sm px-1.5 text-[9px]" style={{ color: 'var(--zz-lime)', background: 'oklch(0.9 0.2 130 / 0.1)' }}>
              {th.tag}
            </span>
          ) : null}
          {th.unread > 0 ? (
            <span
              className="mono ml-auto inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full px-[5px] text-[10px] font-semibold"
              style={{ background: 'var(--zz-lime)', color: 'var(--zz-bg)' }}
            >
              {th.unread}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function ContextPanel({ contact }: { contact: InboxContact }) {
  return (
    <aside className="border-bg-3 w-[280px] flex-shrink-0 overflow-y-auto border-l p-[18px]">
      <div className="mono tracked text-text-3 mb-2 text-[9.5px]">CONTACT</div>
      <div className="mb-3.5 flex items-center gap-2.5">
        <Avatar name={contact.avatar} color={contact.color} size={40} />
        <div>
          <div className="text-[13.5px] font-medium">{contact.name}</div>
          <div className="mono text-text-3 text-[11px]">@{contact.handle}</div>
        </div>
      </div>
      <div className="mb-3.5 flex flex-wrap gap-1.5">
        {contact.tags.map((t) => (
          <Pill key={t.label} tone={t.tone}>
            {t.label}
          </Pill>
        ))}
      </div>

      <div className="mono tracked text-text-3 mb-2 mt-3.5 text-[9.5px]">STATS</div>
      {contact.stats.map((s) => (
        <div key={s.label} className="border-bg-3 flex justify-between border-b py-1.5 text-xs">
          <span className="text-text-2">{s.label}</span>
          <span className="mono">{s.value}</span>
        </div>
      ))}

      <div className="mono tracked text-text-3 mb-2 mt-[18px] text-[9.5px]">JOURNEY</div>
      <div className="relative pl-4">
        <div className="bg-line absolute bottom-1.5 top-1.5" style={{ left: 4, width: 1 }} />
        {contact.journey.map((j) => (
          <div key={`${j.kind}-${j.ts}`} className="relative py-1.5 text-xs">
            <span className="absolute h-2 w-2 rounded-full" style={{ left: -16, top: 10, background: 'var(--zz-lime)' }} />
            <div className="mono text-text-3 text-[10.5px]">
              {j.kind} · {j.ts}
            </div>
            <div className="text-text-2">{j.text}</div>
          </div>
        ))}
      </div>
    </aside>
  );
}
