import type { IconName } from '@zosmed/ui';
import type { PillTone } from '@zosmed/ui';

const LIME = 'var(--zz-lime)';
const BLUE = 'var(--zz-blue)';
const PINK = 'var(--zz-pink)';
const WARN = 'var(--zz-warn)';
const MUTE = '#3a3a40';

// ── Templates ────────────────────────────────────────────────────────────────
export interface TemplateCard {
  t: string;
  d: string;
  tag: string;
  color: string;
  runs: string;
  iconKey: IconName;
}
export interface TemplatesData {
  filters: string[];
  templates: TemplateCard[];
}
export const mockTemplates: TemplatesData = {
  filters: ['All', 'Sales', 'Engagement', 'Lead-gen', 'AI', 'Support'],
  templates: [
    { t: 'Launch Produk', d: 'Comment "info" → reply + DM checkout link + AI follow-up', tag: 'POPULAR', color: LIME, runs: '2.4k', iconKey: 'bolt' },
    { t: 'Giveaway Entry', d: 'Auto-tag entry, validasi follow + share, draw winner', tag: 'POPULAR', color: PINK, runs: '1.8k', iconKey: 'heart' },
    { t: 'Lead Magnet PDF', d: 'Komentar → DM kirim ebook/PDF + email capture', tag: '', color: BLUE, runs: '1.2k', iconKey: 'send' },
    { t: 'FAQ Bot Default', d: 'AI jawab harga, stok, ukuran, shipping pakai katalog', tag: 'AI', color: BLUE, runs: '912', iconKey: 'ai' },
    { t: 'Win-back 30d', d: 'Re-engage user yang DM 30 hari lalu tapi belum closing', tag: '', color: WARN, runs: '742', iconKey: 'user' },
    { t: 'Quiz Funnel', d: 'Komentar trigger quiz → DM hasil + recommendation', tag: 'NEW', color: LIME, runs: '512', iconKey: 'sparkle' },
    { t: 'Story Mention Welcome', d: 'User mention akun di Story → auto-reply DM thanks + intro', tag: '', color: MUTE, runs: '418', iconKey: 'sparkle' },
    { t: 'Story Reply Funnel', d: 'Reply ke story tertentu → kirim link/CTA', tag: '', color: MUTE, runs: '287', iconKey: 'chat' },
    { t: 'Customer Survey', d: 'Post-purchase survey via DM, collect rating', tag: '', color: MUTE, runs: '142', iconKey: 'check' },
  ],
};

// ── Settings ──────────────────────────────────────────────────────────────────
export interface ConnectedAccount {
  ig: string;
  n: string;
  f: string;
  primary?: boolean;
  perm: string;
  last: string;
}
export interface Integration {
  n: string;
  s: string;
  d: string;
}
export interface SettingsData {
  nav: string[];
  activeNav: string;
  accounts: ConnectedAccount[];
  integrations: Integration[];
  usage: [string, number, number][];
}
export const mockSettings: SettingsData = {
  nav: ['Workspace', 'Members', 'Connected accounts', 'Billing', 'Plans', 'API & Webhooks', 'Notifications', 'Security', 'Danger zone'],
  activeNav: 'Connected accounts',
  accounts: [
    { ig: '@ataka.studio', n: 'Ataka Studio', f: '24,800', primary: true, perm: 'full', last: 'just now' },
    { ig: '@ataka.archive', n: 'Ataka Archive', f: '8,200', perm: 'read-only', last: '2h ago' },
    { ig: '@maya.personal', n: 'Maya Rahma', f: '1,400', perm: 'full', last: '1d ago' },
  ],
  integrations: [
    { n: 'Google Sheets', s: 'connected', d: 'Sync leads to your sheet' },
    { n: 'Notion', s: 'connected', d: 'Mirror contacts as DB' },
    { n: 'Meta Pixel', s: 'connected', d: 'Track conversion events' },
    { n: 'Webhook', s: 'configured', d: '2 endpoints active' },
    { n: 'Slack', s: 'available', d: 'Notify lead alerts to channel' },
    { n: 'WhatsApp Business', s: 'soon', d: 'Cross-channel funnel' },
  ],
  usage: [
    ['Auto-DMs', 892, 2500],
    ['AI tokens', 187000, 1000000],
    ['IG accounts', 3, 5],
    ['Workflows', 6, 20],
  ],
};

// ── Billing ───────────────────────────────────────────────────────────────────
export interface BillingPlan {
  n: string;
  p: string;
  sub: string;
  f: string[];
  cta: string;
  tone: 'ghost' | 'current' | 'lime';
}
export interface UsageRow {
  l: string;
  v: number;
  cap: number;
  sub: string;
}
export interface BillingData {
  plans: BillingPlan[];
  invoices: [string, string, string, string, string][];
  usage: UsageRow[];
}
export const mockBilling: BillingData = {
  plans: [
    { n: 'Starter', p: 'Rp 0', sub: 'Free forever', f: ['1 IG account', '40 auto-DMs / hr', '500 AI tokens / day', '3 workflows', 'Basic templates'], cta: 'Downgrade', tone: 'ghost' },
    { n: 'Pro', p: 'Rp 490rb', sub: '/ bulan · current', f: ['5 IG accounts', '200 auto-DMs / hr', '1M AI tokens / day', '20 workflows', 'All templates', 'AI Studio', 'Priority support'], cta: 'Current plan', tone: 'current' },
    { n: 'Enterprise', p: 'Rp 1.9jt+', sub: '/ bulan · custom', f: ['Unlimited IG accounts', 'Custom rate limits', 'Unlimited AI tokens', 'Unlimited workflows', 'White-label option', 'SSO + audit log', 'Dedicated CSM'], cta: 'Talk to sales', tone: 'lime' },
  ],
  invoices: [
    ['15 Apr 2026', 'INV-202604', 'Pro plan · Apr 2026', 'Rp 490,000', 'paid'],
    ['15 Mar 2026', 'INV-202603', 'Pro plan · Mar 2026', 'Rp 490,000', 'paid'],
    ['15 Feb 2026', 'INV-202602', 'Pro plan · Feb 2026', 'Rp 490,000', 'paid'],
    ['28 Jan 2026', 'INV-202601-X', 'AI tokens overage · 240k', 'Rp 86,000', 'paid'],
    ['15 Jan 2026', 'INV-202601', 'Pro plan · Jan 2026', 'Rp 490,000', 'paid'],
    ['15 Dec 2025', 'INV-202512', 'Pro plan · Dec 2025', 'Rp 490,000', 'paid'],
  ],
  usage: [
    { l: 'Auto-DMs sent', v: 8924, cap: 75000, sub: '12% of monthly cap' },
    { l: 'AI tokens', v: 487000, cap: 30000000, sub: 'projected: 720k by end of cycle' },
    { l: 'IG accounts', v: 3, cap: 5, sub: '2 slots remaining' },
    { l: 'Workflows', v: 6, cap: 20, sub: '14 slots remaining' },
    { l: 'Webhook calls', v: 1284, cap: 50000, sub: 'no overage' },
  ],
};

// ── Team ──────────────────────────────────────────────────────────────────────
export interface TeamRoleCard {
  r: string;
  d: string;
  c: string;
}
export interface TeamMemberRow {
  n: string;
  e: string;
  role: 'Owner' | 'Admin' | 'Editor' | 'Viewer';
  avatar: string;
  color: string;
  last: string;
  acc: string[];
  you?: boolean;
  pending?: boolean;
}
export interface TeamData {
  roles: TeamRoleCard[];
  members: TeamMemberRow[];
  activity: [string, string, string, string, string][];
}
export const ROLE_COLOR: Record<TeamMemberRow['role'], string> = {
  Owner: LIME,
  Admin: BLUE,
  Editor: PINK,
  Viewer: MUTE,
};
export const mockTeam: TeamData = {
  roles: [
    { r: 'Owner', d: 'Full access · billing · delete workspace', c: LIME },
    { r: 'Admin', d: 'Manage members, billing, all workflows', c: BLUE },
    { r: 'Editor', d: 'Build & edit workflows, reply DMs', c: PINK },
    { r: 'Viewer', d: 'Read-only access, view analytics', c: MUTE },
  ],
  members: [
    { n: 'Maya Rahmawati', e: 'maya@ataka.id', role: 'Owner', avatar: 'MR', color: LIME, last: 'online now', acc: ['ataka.studio', 'ataka.archive'], you: true },
    { n: 'Andi Saputra', e: 'andi@ataka.id', role: 'Admin', avatar: 'AS', color: BLUE, last: '12 min ago', acc: ['ataka.studio'] },
    { n: 'Lely Pratiwi', e: 'lely.p@ataka.id', role: 'Editor', avatar: 'LP', color: PINK, last: '2 hours ago', acc: ['ataka.studio'] },
    { n: 'Bayu Mahendra', e: 'bayu.m@ataka.id', role: 'Editor', avatar: 'BM', color: WARN, last: '1 day ago', acc: ['ataka.archive'] },
    { n: 'Sari Wulan', e: 'sari.w@ataka.id', role: 'Viewer', avatar: 'SW', color: MUTE, last: '5 days ago', acc: ['ataka.studio', 'ataka.archive'] },
    { n: 'pending: rifqi@ataka.id', e: '', role: 'Editor', avatar: '?', color: MUTE, last: 'invited 2 days ago', acc: ['ataka.studio'], pending: true },
  ],
  activity: [
    ['14:32', 'Andi', 'edited workflow', 'launch-promo-mei', BLUE],
    ['11:08', 'Lely', 'replied to 14 DMs', 'inbox', PINK],
    ['09:42', 'Maya', 'increased rate limit', 'safety center', LIME],
    ['08:17', 'Bayu', 'cloned template "Quiz Funnel"', 'workflows', WARN],
    ['Yesterday', 'Andi', 'invited rifqi@ataka.id', 'members', BLUE],
    ['Yesterday', 'Maya', 'updated AI prompt v3.2', 'AI Studio', LIME],
  ],
};

// ── Notifications ──────────────────────────────────────────────────────────────
export interface NotifItem {
  k: string;
  icon: string;
  c: string;
  t: string;
  sub: string;
  val?: string;
  t2: string;
}
export interface NotifGroup {
  d: string;
  items: NotifItem[];
}
export interface NotificationsData {
  filters: [string, number][];
  groups: NotifGroup[];
}
export const mockNotifications: NotificationsData = {
  filters: [
    ['All', 11],
    ['Leads', 4],
    ['Workflows', 3],
    ['Safety', 1],
    ['AI', 1],
    ['Team', 1],
    ['System', 1],
  ],
  groups: [
    {
      d: 'TODAY',
      items: [
        { k: 'lead', icon: '🔥', c: PINK, t: 'Hot lead detected', sub: '@arief.daud · score 86 · likely to convert', val: '"Saya mau ambil 2 set, COD bandung bisa kak?"', t2: '14:38' },
        { k: 'workflow', icon: '✓', c: LIME, t: 'launch-promo-mei converted Rp 540rb', sub: '@arief.daud purchased · attribution: comment-to-DM', t2: '14:38' },
        { k: 'safety', icon: '⚠', c: WARN, t: 'Rate limit at 80% capacity', sub: '200 dm/hr — auto-paced cooldown enabled', t2: '13:21' },
        { k: 'ai', icon: '🤖', c: BLUE, t: '8 AI conversations need review', sub: 'low confidence (<0.5) · refund / custom-order topics', t2: '12:48' },
      ],
    },
    {
      d: 'YESTERDAY',
      items: [
        { k: 'milestone', icon: '🎉', c: LIME, t: 'You crossed 1,000 auto-replies this month', sub: 'estimated time saved: 18 hours', t2: '20:14' },
        { k: 'integration', icon: '⌬', c: BLUE, t: 'Google Sheets sync completed', sub: '247 new contacts → ataka-leads.csv', t2: '17:32' },
        { k: 'workflow', icon: '⏸', c: '#a3a39c', t: 'win-back-juni paused by Maya', sub: 'manual pause · 142 contacts in queue', t2: '09:01' },
      ],
    },
    {
      d: 'THIS WEEK',
      items: [
        { k: 'system', icon: 'ℹ', c: '#a3a39c', t: 'IG token refresh successful', sub: 'all 3 accounts re-authorized · valid until Aug 2026', t2: '3 days ago' },
        { k: 'team', icon: '👥', c: '#a3a39c', t: 'Andi joined the workspace', sub: 'invited by Maya · role: Editor', t2: '5 days ago' },
        { k: 'product', icon: '✨', c: LIME, t: 'New: AI Studio v2 with custom evals', sub: 'available on your Pro plan — try it now', t2: '6 days ago' },
      ],
    },
  ],
};

// ── Safety ────────────────────────────────────────────────────────────────────
export interface SafetyLimit {
  l: string;
  v: number;
  cap: number;
  rec: string;
}
export interface SafetyData {
  rateLimits: SafetyLimit[];
  antiSpam: [string, boolean, string][];
  log: [string, string, string, string, string][];
}
export const mockSafety: SafetyData = {
  rateLimits: [
    { l: 'Comment replies / hour', v: 142, cap: 750, rec: 'Meta private-reply ~750/hr' },
    { l: 'DMs sent / hour', v: 89, cap: 200, rec: 'safe ~200/hr · queue overflow' },
    { l: 'DMs sent / day', v: 612, cap: 1000, rec: 'behaviour-based soft limit' },
    { l: 'Comments per post / 5min', v: 12, cap: 30, rec: 'human-paced' },
    { l: 'AI tokens / day', v: 187000, cap: 1000000, rec: 'soft (cost guard)' },
  ],
  antiSpam: [
    ['Random delay 2–8s before reply', true, 'human-paced response'],
    ['Skip duplicate DM to same user', true, 'no spam-burst'],
    ['Pause if error rate > 5%', true, 'auto-circuit-breaker'],
    ['Skip reply if comment < 3 chars', false, 'avoid emoji-only spam'],
    ['Mirror user language (ID/EN)', true, 'natural conversation'],
    ['Quiet hours (22:00 — 06:00)', false, 'optional'],
  ],
  log: [
    ['14:32', 'auto-pause', 'rate near limit (89/100 dm/hr)', 'system', WARN],
    ['14:38', 'resumed', 'cooldown done — back to normal', 'system', LIME],
    ['12:18', 'rule-skip', 'duplicate DM blocked — @rina_susanti', 'anti-spam', LIME],
    ['09:42', 'config', 'rate limit set: 150 → 200 dm/hr', 'maya', '#a3a39c'],
    ['09:01', 'workflow', 'win-back-juni paused by user', 'maya', '#a3a39c'],
    ['Yesterday 23:50', 'health-check', 'all systems normal · IG token valid', 'system', '#a3a39c'],
  ],
};

// ── AI Studio ───────────────────────────────────────────────────────────────
export interface BrandVoiceSlider {
  l: string;
  v: number;
  lo: string;
  hi: string;
}
export interface KnowledgeSource {
  f: string;
  m: string;
  last: string;
  status: 'synced' | 'stale';
}
export interface AIStudioData {
  brandVoice: BrandVoiceSlider[];
  knowledge: KnowledgeSource[];
  evals: [string, string, string][];
  handoffs: [string, string, string][];
}
export const mockAIStudio: AIStudioData = {
  brandVoice: [
    { l: 'Formality', v: 0.32, lo: 'casual', hi: 'formal' },
    { l: 'Warmth', v: 0.78, lo: 'cool', hi: 'warm' },
    { l: 'Energy', v: 0.65, lo: 'calm', hi: 'energetic' },
    { l: 'Humor', v: 0.45, lo: 'serious', hi: 'playful' },
  ],
  knowledge: [
    { f: 'products.csv', m: '128 SKU · 1.4 MB', last: '2h ago', status: 'synced' },
    { f: 'faq.md', m: '47 Q&A · 8 KB', last: '1d ago', status: 'synced' },
    { f: 'shipping-policy.pdf', m: '12 pages · 240 KB', last: '3d ago', status: 'synced' },
    { f: 'instagram-bio.txt', m: '420 chars', last: '1w ago', status: 'stale' },
  ],
  evals: [
    ['Helpfulness', '92%', '+3pt'],
    ['Tone match', '88%', '+5pt'],
    ['CTA included', '76%', '−2pt'],
    ['Hand-off rate', '4.2%', '−1.8pt'],
  ],
  handoffs: [
    ['budi.s', 'asked refund policy', 'low confidence (0.42)'],
    ['nadya.p', 'complaint about shipping', 'sentiment: negative'],
    ['putu.gita', 'asked custom order', 'no kb match'],
  ],
};

export const STATUS_TONE: Record<string, PillTone> = {
  connected: 'lime',
  configured: 'lime',
  synced: 'lime',
  available: 'neutral',
  soon: 'neutral',
  stale: 'warn',
};
