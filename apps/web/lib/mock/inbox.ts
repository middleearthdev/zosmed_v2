import type { PillTone } from '@zosmed/ui';

export interface InboxThread {
  u: string;
  last: string;
  t: string;
  unread: number;
  ai: boolean;
  tag?: string;
  avatar: string;
  color: string;
  active?: boolean;
}

export interface ContactStat {
  label: string;
  value: string;
}

export interface JourneyEvent {
  kind: string;
  text: string;
  ts: string;
}

export interface InboxContact {
  name: string;
  handle: string;
  avatar: string;
  color: string;
  meta: string;
  tags: { label: string; tone: PillTone }[];
  stats: ContactStat[];
  journey: JourneyEvent[];
}

export interface InboxData {
  unreadCount: number;
  aiNote: string;
  filters: string[];
  threads: InboxThread[];
  contact: InboxContact;
}

const PINK = 'var(--zz-pink)';
const LIME = 'var(--zz-lime)';
const BLUE = 'var(--zz-blue)';
const NEUTRAL = '#3a3a40';

export const mockInbox: InboxData = {
  unreadCount: 12,
  aiNote: 'AI handling 8 conversations · 4 need review',
  filters: ['All', 'AI', 'Need review', 'Tagged', 'Closed'],
  threads: [
    { u: 'rina_susanti', last: 'Oke kak, link checkout yang mana ya?', t: '2m', unread: 2, ai: true, tag: 'warm-lead', avatar: 'RS', color: PINK, active: true },
    { u: 'arief.daud', last: 'Berapa lama pengirimannya ke Bandung?', t: '14m', unread: 1, ai: true, tag: 'hot-lead', avatar: 'AD', color: LIME },
    { u: 'mira.hidayah', last: 'Saya udah follow kak, tag entry dong', t: '32m', unread: 0, ai: false, tag: 'giveaway', avatar: 'MH', color: BLUE },
    { u: 'budi.s', last: 'Minta foto detailnya yang warna sage', t: '1h', unread: 3, ai: true, tag: 'warm-lead', avatar: 'BS', color: PINK },
    { u: 'nadya.p', last: 'Cod jakarta selatan bisa kak?', t: '2h', unread: 0, ai: true, avatar: 'NP', color: NEUTRAL },
    { u: 'sintia.f', last: 'Stoknya masih ada untuk size M?', t: '3h', unread: 0, ai: true, avatar: 'SF', color: NEUTRAL },
    { u: 'rizky_p', last: 'Wah keren produknya, congrats launch!', t: '5h', unread: 0, ai: false, avatar: 'RP', color: NEUTRAL },
    { u: 'putu.gita', last: 'Ada bundle yang lebih murah ga?', t: '1d', unread: 0, ai: true, avatar: 'PG', color: NEUTRAL },
  ],
  contact: {
    name: 'Rina Susanti',
    handle: 'rina_susanti',
    avatar: 'RS',
    color: PINK,
    meta: '2.4k followers · 3 prior interactions · IG verified',
    tags: [
      { label: 'warm-lead', tone: 'lime' },
      { label: 'first-time', tone: 'neutral' },
      { label: 'jakarta', tone: 'blue' },
    ],
    stats: [
      { label: 'Comments', value: '4' },
      { label: 'DMs', value: '12' },
      { label: 'Last seen', value: '2 min ago' },
      { label: 'Avg response', value: '2m 14s' },
      { label: 'Lifetime value', value: 'Rp 0' },
    ],
    journey: [
      { kind: 'comment', text: '"info dong sis"', ts: '14:32' },
      { kind: 'dm-received', text: 'cek dm reply', ts: '14:32' },
      { kind: 'dm-sent', text: 'product link', ts: '14:32' },
      { kind: 'ai-handle', text: 'intent: ask_link', ts: '14:35' },
    ],
  },
};
