/**
 * Mock fixtures for Contacts list + Contact Profile screens (Phase 4).
 * View-model interfaces are screen-specific; Contact domain type is in @zosmed/types.
 * CLAUDE.md §12a: SoC — display shapes live here, not in domain contracts.
 */
import type { PillTone } from '@zosmed/ui';

// ── Contacts list ────────────────────────────────────────────────────────────

export interface ContactTag {
  label: string;
  tone: PillTone;
}

/** Display row for the contacts table. */
export interface ContactRow {
  /** URL slug used in /contacts/[id] */
  id: string;
  handle: string;
  name: string;
  avatar: string;
  avatarColor: string;
  tags: ContactTag[];
  igFollowers: number;
  conversations: number;
  ltv: string;
  ltvIsZero: boolean;
  source: string;
  lastSeen: string;
}

export interface SegmentChip {
  label: string;
  count: number | null;
  active: boolean;
}

export interface ContactsData {
  eyebrow: string;
  segments: SegmentChip[];
  rows: ContactRow[];
}

// ── Contact Profile ──────────────────────────────────────────────────────────

export interface TimelineEvent {
  date: string;
  kind: string;
  icon: string;
  iconColor: string;
  title: string;
  sub: string;
  quoted?: string;
}

export interface LeadScoreBreakdown {
  label: string;
  value: number;
  max: number;
}

export interface FlowRef {
  name: string;
  status: string;
  statusColor: string;
}

export interface ContactProfileData {
  id: string;
  handle: string;
  name: string;
  avatar: string;
  avatarColor: string;
  tags: ContactTag[];
  metaLine: string;
  stats: { key: string; value: string; highlight: boolean }[];
  tabs: string[];
  timeline: TimelineEvent[];
  leadScore: number;
  leadScoreDelta: string;
  leadScoreBreakdown: LeadScoreBreakdown[];
  properties: { key: string; value: string }[];
  flows: FlowRef[];
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function tagTone(label: string): PillTone {
  if (label.includes('hot')) return 'pink';
  if (label.includes('vip') || label.includes('repeat-buyer')) return 'lime';
  if (['jakarta', 'bandung', 'bali', 'surabaya'].includes(label)) return 'blue';
  return 'neutral';
}

function tags(...labels: string[]): ContactTag[] {
  return labels.map((l) => ({ label: l, tone: tagTone(l) }));
}

// ── Mock data ────────────────────────────────────────────────────────────────

export const mockContactsData: ContactsData = {
  eyebrow: '14.287 CONTACTS · 247 BARU MINGGU INI',
  segments: [
    { label: 'Semua kontak', count: 14287, active: true },
    { label: 'Hot leads', count: 642, active: false },
    { label: 'Warm leads', count: 2341, active: false },
    { label: 'Repeat buyers', count: 412, active: false },
    { label: 'VIP', count: 38, active: false },
    { label: 'Cold', count: 8214, active: false },
    { label: 'Giveaway entries', count: 1247, active: false },
    { label: 'Jakarta', count: 4128, active: false },
    { label: '+ segmen baru', count: null, active: false },
  ],
  rows: [
    {
      id: 'rina_susanti',
      handle: 'rina_susanti',
      name: 'Rina Susanti',
      avatar: 'RS',
      avatarColor: '#3a3a40',
      tags: tags('warm-lead', 'jakarta'),
      igFollowers: 2400,
      conversations: 12,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'promo-launch-mei',
      lastSeen: '2m',
    },
    {
      id: 'arief.daud',
      handle: 'arief.daud',
      name: 'Arief Daud',
      avatar: 'AD',
      avatarColor: '#3a3a40',
      tags: tags('hot-lead', 'bandung'),
      igFollowers: 8800,
      conversations: 21,
      ltv: 'Rp 540rb',
      ltvIsZero: false,
      source: 'promo-launch-mei',
      lastSeen: '14m',
    },
    {
      id: 'mira.hidayah',
      handle: 'mira.hidayah',
      name: 'Mira Hidayah',
      avatar: 'MH',
      avatarColor: '#3a3a40',
      tags: tags('giveaway-entry'),
      igFollowers: 1200,
      conversations: 4,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'giveaway-buku',
      lastSeen: '32m',
    },
    {
      id: 'budi.s',
      handle: 'budi.s',
      name: 'Budi Saputra',
      avatar: 'BS',
      avatarColor: '#3a3a40',
      tags: tags('warm-lead', 'jakarta'),
      igFollowers: 4100,
      conversations: 18,
      ltv: 'Rp 189rb',
      ltvIsZero: false,
      source: 'faq-bot',
      lastSeen: '1j',
    },
    {
      id: 'nadya.p',
      handle: 'nadya.p',
      name: 'Nadya Putri',
      avatar: 'NP',
      avatarColor: '#3a3a40',
      tags: tags('repeat-buyer'),
      igFollowers: 320,
      conversations: 47,
      ltv: 'Rp 1.8jt',
      ltvIsZero: false,
      source: 'organic',
      lastSeen: '2j',
    },
    {
      id: 'sintia.f',
      handle: 'sintia.f',
      name: 'Sintia F.',
      avatar: 'SF',
      avatarColor: '#3a3a40',
      tags: tags('warm-lead'),
      igFollowers: 760,
      conversations: 8,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'promo-launch-mei',
      lastSeen: '3j',
    },
    {
      id: 'rizky_p',
      handle: 'rizky_p',
      name: 'Rizky P.',
      avatar: 'RP',
      avatarColor: '#3a3a40',
      tags: tags('supporter'),
      igFollowers: 12800,
      conversations: 3,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'organic',
      lastSeen: '5j',
    },
    {
      id: 'putu.gita',
      handle: 'putu.gita',
      name: 'Putu Gita',
      avatar: 'PG',
      avatarColor: '#3a3a40',
      tags: tags('warm-lead', 'bali'),
      igFollowers: 540,
      conversations: 9,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'lead-magnet',
      lastSeen: '1h',
    },
    {
      id: 'lely.r',
      handle: 'lely.r',
      name: 'Lely Rahma',
      avatar: 'LR',
      avatarColor: '#3a3a40',
      tags: tags('repeat-buyer', 'vip'),
      igFollowers: 1840,
      conversations: 62,
      ltv: 'Rp 4.2jt',
      ltvIsZero: false,
      source: 'organic',
      lastSeen: '1h',
    },
    {
      id: 'dimas.o',
      handle: 'dimas.o',
      name: 'Dimas O.',
      avatar: 'DO',
      avatarColor: '#3a3a40',
      tags: tags('cold'),
      igFollowers: 280,
      conversations: 1,
      ltv: 'Rp 0',
      ltvIsZero: true,
      source: 'faq-bot',
      lastSeen: '3h',
    },
  ],
};

/** Profile data keyed by handle. Screens call getContact(id) from api.ts. */
const mockProfiles: Record<string, ContactProfileData> = {
  rina_susanti: {
    id: 'rina_susanti',
    handle: 'rina_susanti',
    name: 'Rina Susanti',
    avatar: 'RS',
    avatarColor: 'oklch(0.78 0.2 0)',
    tags: [
      { label: 'warm-lead', tone: 'lime' },
      { label: 'jakarta', tone: 'blue' },
      { label: 'first-time', tone: 'neutral' },
    ],
    metaLine: '@rina_susanti · 2.400 followers · bergabung 28 Apr 2026',
    stats: [
      { key: 'LIFETIME VALUE', value: 'Rp 0', highlight: false },
      { key: 'CONVERSATIONS', value: '12', highlight: false },
      { key: 'LAST SEEN', value: '2 menit lalu', highlight: true },
      { key: 'LEAD SCORE', value: '74 / 100', highlight: true },
      { key: 'ATTRIBUTION', value: 'promo-launch-mei', highlight: false },
    ],
    tabs: ['Activity', 'Conversations', 'Properties', 'Notes', 'Lifecycle'],
    timeline: [
      {
        date: 'Hari ini · 14:35',
        kind: 'AI handled',
        icon: '🤖',
        iconColor: 'var(--zz-blue)',
        title: 'AI membalas dengan tautan produk',
        sub: 'workflow: promo-launch-mei · intent: ask_product_link',
        quoted: '"Yang lagi promo ini kak: ataka.id/promo-mei…"',
      },
      {
        date: 'Hari ini · 14:32',
        kind: 'DM received',
        icon: '💬',
        iconColor: 'var(--zz-lime)',
        title: 'DM terkirim via trigger cek DM',
        sub: 'delivered & read',
        quoted: '"Wah lumayan, ini link ke produk yang mana ya?"',
      },
      {
        date: 'Hari ini · 14:32',
        kind: 'Public reply',
        icon: '💬',
        iconColor: 'var(--zz-lime)',
        title: 'Auto-reply ke komentar',
        sub: 'di @ataka.studio post — "Bundle Mei 20% off"',
        quoted: '"Hai kak Rina! 👋 Cek DM ya, kita kirim detailnya 💚"',
      },
      {
        date: 'Hari ini · 14:32',
        kind: 'Trigger',
        icon: '⚡',
        iconColor: 'var(--zz-warn)',
        title: 'Trigger: promo-launch-mei',
        sub: 'komentar cocok keyword "info"',
        quoted: '"info dong sis, harga berapa?"',
      },
      {
        date: 'Hari ini · 14:31',
        kind: 'Tagged',
        icon: '🏷',
        iconColor: 'var(--zz-text-2)',
        title: 'Auto-tagged: warm-lead',
        sub: 'rule: comment_intent_score > 0.6',
      },
      {
        date: '28 Apr · 09:14',
        kind: 'First touch',
        icon: '✨',
        iconColor: 'var(--zz-pink)',
        title: 'Follow @ataka.studio',
        sub: 'interaksi pertama dengan brand',
      },
    ],
    leadScore: 74,
    leadScoreDelta: '+12 hari ini',
    leadScoreBreakdown: [
      { label: 'Engagement', value: 28, max: 30 },
      { label: 'Intent signals', value: 24, max: 30 },
      { label: 'Recency', value: 18, max: 20 },
      { label: 'Reach', value: 4, max: 20 },
    ],
    properties: [
      { key: 'Email', value: '—' },
      { key: 'Phone', value: '—' },
      { key: 'Kota', value: 'Jakarta' },
      { key: 'Sumber', value: 'promo-launch-mei' },
      { key: 'Pertama kali', value: '28 Apr 2026' },
      { key: 'IG verified', value: 'tidak' },
      { key: 'Bahasa', value: 'id-ID' },
    ],
    flows: [
      { name: 'promo-launch-mei', status: 'step 4 of 6 · DM terkirim', statusColor: 'var(--zz-lime)' },
      { name: 'win-back-juni', status: 'queued · mulai +28h', statusColor: 'var(--zz-text-3)' },
    ],
  },
};

/** Returns profile data. Falls back to a generic stub if handle not found. */
export function getContactProfileByHandle(id: string): ContactProfileData | undefined {
  return mockProfiles[id];
}

/** Fallback profile generated from the contacts table row. */
export function getContactFallbackProfile(id: string): ContactProfileData | undefined {
  const row = mockContactsData.rows.find((r) => r.id === id);
  if (!row) return undefined;
  return {
    id: row.id,
    handle: row.handle,
    name: row.name,
    avatar: row.avatar,
    avatarColor: row.avatarColor,
    tags: row.tags,
    metaLine: `@${row.handle} · sumber: ${row.source}`,
    stats: [
      { key: 'LIFETIME VALUE', value: row.ltv, highlight: !row.ltvIsZero },
      { key: 'CONVERSATIONS', value: String(row.conversations), highlight: false },
      { key: 'LAST SEEN', value: row.lastSeen, highlight: false },
      { key: 'LEAD SCORE', value: '—', highlight: false },
      { key: 'ATTRIBUTION', value: row.source, highlight: false },
    ],
    tabs: ['Activity', 'Conversations', 'Properties', 'Notes', 'Lifecycle'],
    timeline: [],
    leadScore: 0,
    leadScoreDelta: '',
    leadScoreBreakdown: [],
    properties: [
      { key: 'Sumber', value: row.source },
      { key: 'IG Followers', value: row.igFollowers.toLocaleString('id-ID') },
    ],
    flows: [],
  };
}
