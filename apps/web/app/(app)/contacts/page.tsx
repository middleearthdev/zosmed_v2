import Link from 'next/link';
import { Avatar, Button, I, Pill } from '@zosmed/ui';
import { getContacts } from '@/lib/mock/api';
import type { ContactRow, SegmentChip } from '@/lib/mock/contacts';
import { PageHeader } from '../_components/PageHeader';

export default async function ContactsPage() {
  const data = await getContacts();

  return (
    <>
      <PageHeader>
        <span className="text-sm font-medium">Contacts</span>
        <div className="flex gap-2">
          <Button variant="ghost" className="px-3 py-[7px] text-xs">
            Export CSV
          </Button>
          <Button variant="lime" icon={<I.plus />} className="px-3 py-[7px] text-xs">
            New segment
          </Button>
        </div>
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto px-6 pb-12 pt-6">
        {/* Eyebrow + heading */}
        <div className="mb-5 flex items-end justify-between">
          <div>
            <span className="mono tracked text-text-3 text-[10px]">{data.eyebrow}</span>
            <h1 className="mono mt-1.5 text-3xl font-medium tracking-tight">
              People &amp; segments
            </h1>
          </div>
        </div>

        {/* Segment chips */}
        <div className="mb-4 flex flex-wrap gap-2">
          {data.segments.map((s) => (
            <SegmentPill key={s.label} chip={s} />
          ))}
        </div>

        {/* Search + filter row */}
        <div className="mb-3 flex gap-2">
          <div className="bg-bg-2 border-line flex flex-1 items-center gap-2 rounded-lg border px-3 py-2">
            <I.search />
            <span className="text-text-3 text-[13px]">
              Cari nama, username, atau tag…
            </span>
          </div>
          <Button variant="ghost" icon={<I.filter />} className="px-3 py-2 text-xs">
            Filters · 0
          </Button>
          <Button variant="ghost" className="px-3 py-2 text-xs">
            Sort · Terakhir dilihat ↓
          </Button>
        </div>

        {/* Contacts table */}
        <div className="bg-bg-2 border-line overflow-hidden rounded-xl border">
          {/* Table header */}
          <div
            className="border-line grid border-b px-4 py-[10px]"
            style={{
              gridTemplateColumns: '1.6fr 1.4fr 0.7fr 0.7fr 0.9fr 1fr 0.6fr',
              background: '#0d0d0d',
            }}
          >
            {['CONTACT', 'TAGS', 'IG FOLLOWERS', 'CONV.', 'LTV', 'SUMBER', 'TERAKHIR'].map(
              (h) => (
                <span key={h} className="mono tracked text-text-3 text-[9.5px]">
                  {h}
                </span>
              ),
            )}
          </div>

          {/* Rows */}
          {data.rows.map((row, i) => (
            <ContactTableRow key={row.id} row={row} isLast={i === data.rows.length - 1} />
          ))}
        </div>
      </div>
    </>
  );
}

function SegmentPill({ chip }: { chip: SegmentChip }) {
  return (
    <span
      className="mono inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-[12px]"
      style={
        chip.active
          ? { background: 'var(--zz-lime)', color: 'var(--zz-bg)' }
          : {
              background: 'var(--zz-bg-2)',
              color: 'var(--zz-text-2)',
              border: '1px solid var(--zz-line)',
            }
      }
    >
      {chip.label}
      {chip.count != null && (
        <span style={{ opacity: 0.6 }}>{chip.count.toLocaleString('id-ID')}</span>
      )}
    </span>
  );
}

function ContactTableRow({ row, isLast }: { row: ContactRow; isLast: boolean }) {
  return (
    <Link
      href={`/contacts/${row.id}`}
      className="grid items-center gap-2 px-4 py-3 transition-colors hover:bg-white/[0.02]"
      style={{
        gridTemplateColumns: '1.6fr 1.4fr 0.7fr 0.7fr 0.9fr 1fr 0.6fr',
        borderBottom: isLast ? 'none' : '1px solid #1a1a1d',
        display: 'grid',
      }}
    >
      {/* Contact name + handle */}
      <div className="flex items-center gap-2.5">
        <Avatar name={row.avatar} color={row.avatarColor} size={28} />
        <div>
          <div className="text-[13px] font-medium">{row.name}</div>
          <div className="mono text-text-3 text-[11px]">@{row.handle}</div>
        </div>
      </div>

      {/* Tags */}
      <div className="flex flex-wrap gap-1">
        {row.tags.map((t) => (
          <Pill key={t.label} tone={t.tone}>
            {t.label}
          </Pill>
        ))}
      </div>

      {/* IG followers */}
      <span className="mono tnum text-[12px]">{row.igFollowers.toLocaleString('id-ID')}</span>

      {/* Conversations */}
      <span className="mono tnum text-[12px]">{row.conversations}</span>

      {/* LTV */}
      <span
        className="mono tnum text-[12px]"
        style={{ color: row.ltvIsZero ? 'var(--zz-text-3)' : 'var(--zz-lime)' }}
      >
        {row.ltv}
      </span>

      {/* Source */}
      <span className="mono text-text-2 text-[11px]">{row.source}</span>

      {/* Last seen */}
      <span className="mono text-text-3 text-[11px]">{row.lastSeen}</span>
    </Link>
  );
}
