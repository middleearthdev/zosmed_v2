import type { IconName } from '@zosmed/ui';

/**
 * Shared view-model types for the workflow canvas (Palette / FlowCanvas /
 * InspectorCanvas). These are consumed with REAL data (mapped in
 * `lib/workflow-catalog.ts`, ADR-004 F4/F7) for the connected builder, and
 * with mock data (below + `mockInspector`) for the standalone
 * `/workflows/inspector` demo screen that iteration keeps untouched.
 */

// ── Node colors (CLAUDE.md §7 catalog) ───────────────────────────────────────
export type WfNodeType = 'TRIGGER' | 'FILTER' | 'ACTION' | 'AI' | 'OUTPUT';

export const NODE_COLORS: Record<WfNodeType, string> = {
  TRIGGER: 'var(--zz-lime)',
  FILTER: 'var(--zz-warn)',
  ACTION: 'var(--zz-pink)',
  AI: 'var(--zz-blue)',
  OUTPUT: 'var(--zz-lime)',
};

// ── Builder view-model ───────────────────────────────────────────────────────
export interface PaletteItem {
  iconKey: IconName;
  label: string;
  /** node_type from the feasible catalog (`NODE_CATALOG`, `@zosmed/types`). */
  nodeType: string;
  /** false = palette-only, badge "segera" (R6 — not runnable this iteration). */
  runnable: boolean;
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

// ── Inspector (deep editor demo screen, `/workflows/inspector`) ─────────────
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
