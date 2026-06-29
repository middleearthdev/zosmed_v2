import type { IconName } from '@zosmed/ui';

export interface DashboardStat {
  label: string;
  value: string;
  delta: string;
  accent: string;
  spark: number[];
}

export interface DashboardWorkflow {
  name: string;
  status: 'running' | 'paused';
  triggers: number;
  dms: number;
  leads: number;
  post: string;
  progress: number;
}

export interface FeedEvent {
  t: string;
  who: string;
  what: string;
  wf: string;
  iconKey: IconName;
  color: string;
}

export interface KeywordHit {
  keyword: string;
  hits: number;
}

export interface SafetyRow {
  label: string;
  value: string;
  cap: string;
  pct: number;
}

export interface QuickStartItem {
  iconKey: IconName;
  title: string;
  sub: string;
}

export interface DashboardData {
  greeting: string;
  dateLabel: string;
  summary: string;
  stats: DashboardStat[];
  workflows: DashboardWorkflow[];
  feed: FeedEvent[];
  keywords: KeywordHit[];
  safety: SafetyRow[];
  quickStart: QuickStartItem[];
}

const LIME = 'var(--zz-lime)';
const BLUE = 'var(--zz-blue)';
const PINK = 'var(--zz-pink)';
const WARN = 'var(--zz-warn)';

export const mockDashboard: DashboardData = {
  dateLabel: 'SELASA · 28 APR 2026',
  greeting: 'Halo Maya, hari ini ramai 👋',
  summary: '2 workflow aktif memproses 1,247 komentar dalam 24 jam terakhir.',
  stats: [
    { label: 'COMMENTS RECEIVED', value: '1,247', delta: '18.2%', accent: LIME, spark: [3, 5, 4, 7, 6, 9, 8, 11, 10, 13, 12, 15] },
    { label: 'AUTO-DM SENT', value: '892', delta: '12.4%', accent: BLUE, spark: [2, 4, 3, 5, 4, 6, 5, 7, 6, 8, 7, 9] },
    { label: 'LEADS CAPTURED', value: '247', delta: '24.1%', accent: PINK, spark: [1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 7, 8] },
    { label: 'EST. REVENUE', value: 'Rp 14.8jt', delta: '31.0%', accent: WARN, spark: [2, 3, 4, 3, 5, 4, 6, 5, 7, 6, 8, 9] },
  ],
  workflows: [
    { name: 'promo-launch-mei', status: 'running', triggers: 847, dms: 612, leads: 178, post: 'IG: @ataka.studio · 4 posts', progress: 0.74 },
    { name: 'giveaway-buku-baru', status: 'running', triggers: 312, dms: 248, leads: 56, post: 'IG: @ataka.studio · 1 post', progress: 0.62 },
    { name: 'faq-bot-default', status: 'running', triggers: 88, dms: 32, leads: 13, post: 'IG: @ataka.studio · all posts', progress: 0.28 },
    { name: 'win-back-juni', status: 'paused', triggers: 0, dms: 0, leads: 0, post: 'paused 2 hari lalu', progress: 0 },
  ],
  feed: [
    { t: 'just now', who: 'rina_susanti', what: 'commented "info dong"', wf: 'promo-launch-mei', iconKey: 'chat', color: LIME },
    { t: '12s', who: 'rina_susanti', what: 'received DM with product link', wf: 'promo-launch-mei', iconKey: 'send', color: PINK },
    { t: '34s', who: 'arief.daud', what: 'replied to AI: "berapa lama pengirimannya?"', wf: 'promo-launch-mei', iconKey: 'ai', color: BLUE },
    { t: '1m', who: 'arief.daud', what: 'tagged as warm-lead', wf: 'promo-launch-mei', iconKey: 'check', color: LIME },
    { t: '2m', who: 'mira.hidayah', what: 'added to giveaway entry list', wf: 'giveaway-buku-baru', iconKey: 'heart', color: PINK },
    { t: '3m', who: 'system', what: 'rate-limit hit: pausing 8 minutes', wf: 'safety-monitor', iconKey: 'shield', color: WARN },
    { t: '5m', who: 'budi.s', what: 'commented "harga?" — replied + DM sent', wf: 'promo-launch-mei', iconKey: 'chat', color: LIME },
  ],
  keywords: [
    { keyword: 'info', hits: 412 },
    { keyword: 'harga', hits: 287 },
    { keyword: 'ready', hits: 198 },
    { keyword: 'pre-order', hits: 142 },
    { keyword: 'size', hits: 96 },
    { keyword: 'cod?', hits: 78 },
  ],
  safety: [
    { label: 'Comment replies / hour', value: '142', cap: '750', pct: 0.19 },
    { label: 'DMs / hour', value: '89', cap: '200', pct: 0.45 },
    { label: '24h window msgs', value: '187k', cap: '1M', pct: 0.19 },
    { label: 'IG API calls', value: '4,210', cap: '20k', pct: 0.21 },
  ],
  quickStart: [
    { iconKey: 'bolt', title: 'Tambah trigger ke post baru', sub: '2 menit' },
    { iconKey: 'ai', title: 'Train AI dengan FAQ kamu', sub: '5 menit' },
    { iconKey: 'chart', title: 'Setup Meta Pixel attribution', sub: '3 menit' },
    { iconKey: 'user', title: 'Invite teammate ke workspace', sub: '1 menit' },
  ],
};
