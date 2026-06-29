import type { IconName } from '@zosmed/ui';

// ── Node colors (CLAUDE.md §7 catalog) ───────────────────────────────────────
export type WfNodeType = 'TRIGGER' | 'FILTER' | 'ACTION' | 'AI' | 'OUTPUT';

export const NODE_COLORS: Record<WfNodeType, string> = {
  TRIGGER: 'var(--zz-lime)',
  FILTER: 'var(--zz-warn)',
  ACTION: 'var(--zz-pink)',
  AI: 'var(--zz-blue)',
  OUTPUT: 'var(--zz-lime)',
};

// ── Builder ──────────────────────────────────────────────────────────────────
export interface PaletteItem {
  iconKey: IconName;
  label: string;
}
export interface PaletteSection {
  title: string;
  color: string;
  items: PaletteItem[];
}
export interface FlowNode {
  id: string;
  x: number;
  y: number;
  type: WfNodeType;
  title: string;
  sub: string;
  badge: string;
  iconKey: IconName;
  selected?: boolean;
  focus?: boolean;
}
export interface FlowLink {
  from: string;
  to: string;
  active?: boolean;
}
export interface BuilderData {
  name: string;
  status: string;
  meta: string;
  runs: string;
  errorRate: string;
  palette: PaletteSection[];
  nodes: FlowNode[];
  links: FlowLink[];
}

export const mockBuilder: BuilderData = {
  name: 'promo-launch-mei',
  status: '● RUNNING',
  meta: 'last edited 14m ago by maya',
  runs: '847 runs',
  errorRate: 'error rate 0.4%',
  palette: [
    {
      title: 'TRIGGERS',
      color: 'var(--zz-lime)',
      items: [
        { iconKey: 'bolt', label: 'IG Comment received' },
        { iconKey: 'inbox', label: 'IG DM received' },
        { iconKey: 'heart', label: 'Story reply' },
        { iconKey: 'sparkle', label: 'Story mention' },
        { iconKey: 'box', label: 'Comment-to-order (post/Reel)' },
        { iconKey: 'bolt', label: 'Click-to-DM ad' },
      ],
    },
    {
      title: 'FILTERS',
      color: 'var(--zz-warn)',
      items: [
        { iconKey: 'filter', label: 'Keyword match' },
        { iconKey: 'chat', label: 'Conversation state' },
        { iconKey: 'shield', label: 'Intent: ragu / trust' },
        { iconKey: 'filter', label: 'Post selection' },
        { iconKey: 'cog', label: 'Time window' },
      ],
    },
    {
      title: 'ACTIONS',
      color: 'var(--zz-pink)',
      items: [
        { iconKey: 'chat', label: 'Reply comment' },
        { iconKey: 'send', label: 'Send DM' },
        { iconKey: 'ai', label: 'AI reply (olshop)' },
        { iconKey: 'whatsapp', label: 'Kirim link WhatsApp' },
        { iconKey: 'shield', label: 'Kirim trust-kit' },
        { iconKey: 'box', label: 'Reserve stok (comment-to-order)' },
        { iconKey: 'bell', label: 'Notify opt-in' },
        { iconKey: 'user', label: 'Hand-off to human' },
        { iconKey: 'check', label: 'Tag contact' },
      ],
    },
  ],
  nodes: [
    { id: 'n1', x: 60, y: 220, type: 'TRIGGER', title: 'IG Comment received', sub: 'Post: promo-launch-mei', badge: '847 fired', iconKey: 'bolt' },
    { id: 'n2', x: 360, y: 100, type: 'FILTER', title: 'Keyword: "info" / "harga"', sub: 'includes · case insensitive', badge: '612 pass · 235 skip', iconKey: 'filter' },
    { id: 'n3', x: 360, y: 360, type: 'FILTER', title: 'Conversation state', sub: 'within 24h window', badge: '418 pass · 194 skip', iconKey: 'chat' },
    { id: 'n4', x: 680, y: 60, type: 'ACTION', title: 'Reply Comment', sub: '"Cek DM ya 👀✨"', badge: '612 sent · 0.4% err', iconKey: 'chat', selected: true },
    { id: 'n5', x: 680, y: 200, type: 'ACTION', title: 'Send DM', sub: 'template + product link', badge: '578 sent', iconKey: 'send', focus: true },
    { id: 'n6', x: 680, y: 380, type: 'ACTION', title: 'Hand-off to human', sub: 'if refund / complaint', badge: '178 routed', iconKey: 'user' },
    { id: 'n7', x: 1000, y: 130, type: 'AI', title: 'AI Follow-up', sub: 'gpt-4o-mini · brand tone', badge: '247 conversations', iconKey: 'ai' },
    { id: 'n8', x: 1000, y: 320, type: 'OUTPUT', title: 'Tag → warm-lead', sub: 'add to segment', badge: '178 tagged', iconKey: 'check' },
  ],
  links: [
    { from: 'n1', to: 'n2' },
    { from: 'n1', to: 'n3' },
    { from: 'n2', to: 'n4' },
    { from: 'n2', to: 'n5', active: true },
    { from: 'n3', to: 'n6' },
    { from: 'n5', to: 'n7', active: true },
    { from: 'n4', to: 'n8' },
    { from: 'n6', to: 'n8' },
  ],
};

// ── Runs ─────────────────────────────────────────────────────────────────────
export type RunStatusKey = 'success' | 'failed' | 'review';
export interface RunRow {
  id: string;
  t: string;
  wf: string;
  trig: string;
  dur: string;
  steps: string;
  status: RunStatusKey;
}
export interface RunRateBar {
  success: number;
  review: number;
  failed: number;
}
export interface RunStep {
  k: WfNodeType;
  l: string;
  d: string;
  dur: string;
  payload: string;
  color: string;
}
export interface RunsData {
  kpis: { label: string; value: string; delta: string; color: string }[];
  runRate: RunRateBar[];
  runs: RunRow[];
  detail: {
    id: string;
    facts: [string, string][];
    steps: RunStep[];
  };
}

// Deterministic run-rate bars (no Math.random — SSR-safe).
const runRate: RunRateBar[] = Array.from({ length: 48 }, (_, i) => ({
  success: Math.round(18 + Math.sin(i * 0.4) * 10 + ((i * 7) % 9)),
  review: i % 7 === 3 ? 3 : 0,
  failed: i % 11 === 5 ? 2 : 0,
}));

export const mockRuns: RunsData = {
  kpis: [
    { label: 'RUNS · 24H', value: '1,284', delta: '+18%', color: 'var(--zz-lime)' },
    { label: 'SUCCESS RATE', value: '97.4%', delta: '+0.3pt', color: 'var(--zz-lime)' },
    { label: 'AVG DURATION', value: '3.4s', delta: '−0.2s', color: 'var(--zz-lime)' },
    { label: 'ERRORS · 24H', value: '34', delta: '+12', color: 'var(--zz-pink)' },
    { label: 'NEED REVIEW', value: '8', delta: '+3', color: 'var(--zz-warn)' },
  ],
  runRate,
  runs: [
    { id: 'run_8821', t: '14:32:08', wf: 'launch-promo-mei', trig: 'comment by @rina_susanti', dur: '3.2s', steps: '6/6', status: 'success' },
    { id: 'run_8820', t: '14:31:54', wf: 'launch-promo-mei', trig: 'comment by @arief.daud', dur: '4.1s', steps: '6/6', status: 'success' },
    { id: 'run_8819', t: '14:31:12', wf: 'faq-bot-default', trig: 'dm by @sintia.f', dur: '2.7s', steps: '4/4', status: 'success' },
    { id: 'run_8818', t: '14:30:48', wf: 'launch-promo-mei', trig: 'comment by @putu.gita', dur: '6.4s', steps: '4/6', status: 'failed' },
    { id: 'run_8817', t: '14:30:22', wf: 'giveaway-buku', trig: 'comment by @mira.hidayah', dur: '1.8s', steps: '5/5', status: 'success' },
    { id: 'run_8816', t: '14:29:51', wf: 'launch-promo-mei', trig: 'comment by @budi.s', dur: '3.4s', steps: '6/6', status: 'success' },
    { id: 'run_8815', t: '14:29:18', wf: 'win-back-juni', trig: 'schedule daily', dur: '12.4s', steps: '8/8', status: 'success' },
    { id: 'run_8814', t: '14:28:42', wf: 'faq-bot-default', trig: 'dm by @nadya.p', dur: '5.1s', steps: '4/4', status: 'review' },
    { id: 'run_8813', t: '14:28:11', wf: 'launch-promo-mei', trig: 'comment by @lely.r', dur: '2.9s', steps: '6/6', status: 'success' },
    { id: 'run_8812', t: '14:27:54', wf: 'faq-bot-default', trig: 'dm by @rizky_p', dur: '2.2s', steps: '4/4', status: 'success' },
  ],
  detail: {
    id: 'run_8821',
    facts: [
      ['Trigger', '@rina_susanti'],
      ['Started', '14:32:08'],
      ['Duration', '3.2s'],
      ['Tokens', '184'],
      ['Cost', '$0.0042'],
      ['Result', '+1 lead'],
    ],
    steps: [
      { k: 'TRIGGER', l: 'Comment received', d: '+0.0s', dur: '12ms', payload: '"info dong sis"', color: 'var(--zz-lime)' },
      { k: 'FILTER', l: 'Keyword "info" matched', d: '+0.1s', dur: '8ms', payload: 'matched: ["info","price","harga"]', color: 'var(--zz-warn)' },
      { k: 'ACTION', l: 'Reply public comment', d: '+0.5s', dur: '320ms', payload: '"Cek DM ya kak 💚"', color: 'var(--zz-pink)' },
      { k: 'ACTION', l: 'Send DM template', d: '+1.1s', dur: '480ms', payload: 'tpl: launch-promo-mei-v3', color: 'var(--zz-pink)' },
      { k: 'AI', l: 'AI follow-up generated', d: '+2.4s', dur: '1240ms', payload: 'intent: ask_link · 184 tok', color: 'var(--zz-blue)' },
      { k: 'OUTPUT', l: 'Lead captured · warm', d: '+3.2s', dur: '24ms', payload: 'tagged: warm-lead, jakarta', color: 'var(--zz-lime)' },
    ],
  },
};

// ── Inspector (deep editor state) ────────────────────────────────────────────
export interface InspectorNode {
  id: string;
  x: number;
  y: number;
  type: WfNodeType;
  title: string;
  sub: string;
  selected?: boolean;
}
export interface InspectorData {
  unpublished: number;
  nodes: InspectorNode[];
  variables: [string, string][];
  behavior: [string, string][];
}

export const mockInspector: InspectorData = {
  unpublished: 2,
  nodes: [
    { id: 'i1', x: 80, y: 150, type: 'TRIGGER', title: 'IG Comment', sub: '@ataka.studio · any post' },
    { id: 'i2', x: 320, y: 150, type: 'FILTER', title: 'Keyword match', sub: '"info" · "harga" · "price"' },
    { id: 'i3', x: 600, y: 150, type: 'ACTION', title: 'Reply public', sub: '"Cek DM ya kak 💚"', selected: true },
    { id: 'i4', x: 880, y: 90, type: 'ACTION', title: 'Send DM template', sub: 'launch-promo-mei-v3' },
    { id: 'i5', x: 880, y: 210, type: 'AI', title: 'AI Follow-up', sub: 'tone: friendly' },
    { id: 'i6', x: 600, y: 350, type: 'ACTION', title: 'Tag contact', sub: 'warm-lead' },
  ],
  variables: [
    ['{{first_name}}', 'extracted from IG profile name'],
    ['{{post_caption}}', 'first 60 chars of trigger post'],
    ['{{comment_text}}', 'original comment'],
    ['{{intent}}', 'classified intent (ai)'],
  ],
  behavior: [
    ['Random delay', '2 — 8 seconds (human-paced)'],
    ['Skip if same comment', 'in last 10 minutes'],
    ['Skip if length', '< 3 chars'],
    ['Mirror language', 'auto (id / en)'],
  ],
};

// ── Comment-to-Order (keep/C engine) ─────────────────────────────────────────
export const WA_GREEN = 'oklch(0.82 0.2 145)';

export interface IncomingComment {
  u: string;
  t: string;
  tm: string;
  match: string | null;
  ok?: boolean;
  dupe?: boolean;
}
export interface ReservationRow {
  code: string;
  u: string;
  p: string;
  price: string;
  st: 'reserved' | 'waiting-pay' | 'closed-wa' | 'expired';
  cd: string;
  color: string;
}
export interface CatalogProduct {
  code: string;
  name: string;
  left: number;
  total: number;
}
export interface CommentOrderData {
  postComments: string;
  comments: IncomingComment[];
  stats: { label: string; value: string; color: string }[];
  reservations: ReservationRow[];
  products: CatalogProduct[];
}

export const mockCommentOrder: CommentOrderData = {
  postComments: '💬 412 komentar',
  comments: [
    { u: 'budi.s', t: 'keep C3 dong sis', tm: '2m', match: 'C3', ok: true },
    { u: 'rina_susanti', t: 'C1 ya kak 🙏', tm: '5m', match: 'C1', ok: true },
    { u: 'arief.daud', t: 'mantap kualitasnya', tm: '8m', match: null },
    { u: 'mira.h', t: 'keep yang sage 2', tm: '12m', match: 'C3', ok: true },
    { u: 'sintia.f', t: 'C5 size M', tm: '18m', match: 'C5', ok: true },
    { u: 'nadya.p', t: 'harga C2 brp?', tm: '21m', match: null },
    { u: 'putu.gita', t: '1 buat ivory tote', tm: '24m', match: 'C7', ok: true },
    { u: 'lely.r', t: 'keepp C3!!', tm: '30m', match: 'C3', dupe: true },
  ],
  stats: [
    { label: 'CODE DETECTED', value: '147', color: 'var(--zz-lime)' },
    { label: 'RESERVED NOW', value: '38', color: 'var(--zz-warn)' },
    { label: 'CLOSED (WA)', value: '92', color: 'var(--zz-lime)' },
    { label: 'EXPIRED', value: '17', color: 'var(--zz-pink)' },
  ],
  reservations: [
    { code: 'C3', u: 'budi.s', p: 'Sage Tote · M', price: 'Rp 189rb', st: 'reserved', cd: '4:52', color: 'var(--zz-warn)' },
    { code: 'C1', u: 'rina_susanti', p: 'Terracotta Pouch', price: 'Rp 129rb', st: 'waiting-pay', cd: '3:18', color: 'var(--zz-warn)' },
    { code: 'C5', u: 'sintia.f', p: 'Ivory Bag · M', price: 'Rp 219rb', st: 'reserved', cd: '4:10', color: 'var(--zz-warn)' },
    { code: 'C7', u: 'putu.gita', p: 'Ivory Tote', price: 'Rp 189rb', st: 'closed-wa', cd: '✓ closed', color: 'var(--zz-lime)' },
    { code: 'C3', u: 'mira.h', p: 'Sage Tote · L', price: 'Rp 189rb', st: 'reserved', cd: '2:44', color: 'var(--zz-warn)' },
    { code: 'C2', u: 'dewi.k', p: 'Midnight Pouch', price: 'Rp 129rb', st: 'expired', cd: '— released', color: 'var(--zz-pink)' },
  ],
  products: [
    { code: 'C1', name: 'Terracotta Pouch', left: 8, total: 12 },
    { code: 'C3', name: 'Sage Tote', left: 3, total: 20 },
    { code: 'C5', name: 'Ivory Bag', left: 14, total: 15 },
    { code: 'C7', name: 'Ivory Tote', left: 0, total: 10 },
  ],
};
