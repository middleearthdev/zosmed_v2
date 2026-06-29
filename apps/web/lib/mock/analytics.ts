/**
 * Mock fixtures for Analytics overview + drilldown screens (Phase 4).
 * All numeric series use deterministic math (sin/cos only — no Math.random()) to
 * avoid SSR hydration mismatches (CLAUDE.md §6 note in instructions).
 */

// ── Analytics overview ───────────────────────────────────────────────────────

export interface BigMetric {
  label: string;
  value: string;
  delta: string;
  /** CSS color string */
  color: string;
}

export interface FunnelStep {
  label: string;
  n: number;
  p: number; // 0..1
  color: string;
  drop?: string; // e.g. '−18.4%'
}

export interface WorkflowRevRow {
  name: string;
  runs: number;
  cvr: string;
  revenue: string;
  /** 0..1 bar fill ratio */
  barRatio: number;
}

export interface IntentRow {
  label: string;
  count: number;
  color: string;
}

export interface LeaderboardEntry {
  handle: string;
  revenue: string;
  initials: string;
  color: string;
}

/** Precomputed heatmap cell (7 rows × 24 cols). */
export interface HeatCell {
  day: number;
  hour: number;
  /** 0..1 opacity/intensity value */
  v: number;
}

export interface AnalyticsData {
  dateLabel: string;
  bigMetrics: BigMetric[];
  /** Precomputed chart series — 28 data points each. */
  chartSeries: {
    color: string;
    points: number[];
  }[];
  funnel: FunnelStep[];
  workflowsByRevenue: WorkflowRevRow[];
  heatmap: HeatCell[];
  topIntents: IntentRow[];
  leaderboard: LeaderboardEntry[];
}

// ── Analytics drilldown ──────────────────────────────────────────────────────

export interface StepConversion {
  label: string;
  n: number;
  pct: number;
}

export interface PostRevRow {
  title: string;
  comments: number;
  revenue: string;
}

export interface KeywordIntentBar {
  label: string;
  n: number;
  pct: number;
  color: string;
}

/** Precomputed "by time of day" bar intensity (24 bars, 0..1). */
export type ByTimeBar = { hour: number; v: number };

export interface DrilldownData {
  workflowId: string;
  workflowLabel: string;
  revenue: string;
  revenueDelta: string;
  revenueVsPrior: string;
  /** Daily revenue points — 7 days (Mon→Sun). */
  revenuePoints: { day: string; value: number }[];
  stepConversion: StepConversion[];
  byTime: ByTimeBar[];
  byPost: PostRevRow[];
  byIntent: KeywordIntentBar[];
}

// ── Deterministic helpers (no Math.random()) ─────────────────────────────────

function det(i: number, freq = 0.6, off = 0): number {
  return Math.sin(i * freq + off);
}

function makeChartPoints(base: number, vary: number, off: number, days = 28): number[] {
  return Array.from({ length: days }, (_, i) =>
    base + det(i, 0.6, off * 7) * 12 + (i / days) * vary + det(i, 1.7) * 6,
  );
}

function makeHeatmap(): HeatCell[] {
  const cells: HeatCell[] = [];
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      const v = Math.max(
        0,
        det(h - 8, (1 / 24) * Math.PI * 2) * 0.5 +
          0.5 +
          det(d, 1.3) * 0.2 +
          det(h * 0.5 + d * 1.7) * 0.1,
      );
      cells.push({ day: d, hour: h, v: Math.min(1, v) });
    }
  }
  return cells;
}

function makeByTime(): ByTimeBar[] {
  return Array.from({ length: 24 }, (_, h) => ({
    hour: h,
    v: Math.max(0.04, det(h - 4, 0.3) * 0.5 + 0.4 + det(h * 1.3 + 0.7) * 0.1),
  }));
}

// ── Mock analytics data ───────────────────────────────────────────────────────

export const mockAnalyticsData: AnalyticsData = {
  dateLabel: '1 APR — 28 APR 2026',
  bigMetrics: [
    { label: 'COMMENTS', value: '38.247', delta: '+22%', color: 'var(--zz-lime)' },
    { label: 'DM TERKIRIM', value: '26.891', delta: '+18%', color: 'var(--zz-blue)' },
    { label: 'LEADS', value: '6.142', delta: '+34%', color: 'var(--zz-pink)' },
    { label: 'CVR%', value: '22,8%', delta: '+4,2pt', color: 'var(--zz-warn)' },
    { label: 'EST. REVENUE', value: 'Rp 348jt', delta: '+41%', color: 'var(--zz-lime)' },
  ],
  chartSeries: [
    { color: 'var(--zz-lime)', points: makeChartPoints(90, 50, 0) },
    { color: 'var(--zz-blue)', points: makeChartPoints(60, 40, 0.4) },
    { color: 'var(--zz-pink)', points: makeChartPoints(25, 18, 0.9) },
  ],
  funnel: [
    { label: 'Comments received', n: 38247, p: 1.0, color: 'var(--zz-lime)' },
    { label: 'Passed filter', n: 31204, p: 0.815, color: 'var(--zz-lime)', drop: '−18,4%' },
    { label: 'Comment replied', n: 30891, p: 0.807, color: 'var(--zz-pink)', drop: '−1,0%' },
    { label: 'DM delivered', n: 26891, p: 0.703, color: 'var(--zz-pink)', drop: '−12,9%' },
    { label: 'AI conversation', n: 14207, p: 0.371, color: 'var(--zz-blue)', drop: '−47,2%' },
    { label: 'Leads captured', n: 6142, p: 0.16, color: 'var(--zz-lime)', drop: '−56,7%' },
    { label: 'Purchases (tracked)', n: 1402, p: 0.0367, color: 'var(--zz-warn)', drop: '−77,2%' },
  ],
  workflowsByRevenue: [
    { name: 'promo-launch-mei', runs: 12847, cvr: '24,2%', revenue: 'Rp 187jt', barRatio: 1.0 },
    { name: 'giveaway-buku', runs: 6312, cvr: '18,7%', revenue: 'Rp 89jt', barRatio: 0.48 },
    { name: 'faq-bot-default', runs: 8821, cvr: '12,4%', revenue: 'Rp 42jt', barRatio: 0.22 },
    { name: 'lead-magnet-ebook', runs: 3218, cvr: '21,8%', revenue: 'Rp 18jt', barRatio: 0.1 },
    { name: 'quiz-style-guide', runs: 1402, cvr: '9,1%', revenue: 'Rp 8jt', barRatio: 0.04 },
    { name: 'win-back-juni', runs: 892, cvr: '6,2%', revenue: 'Rp 4jt', barRatio: 0.02 },
  ],
  heatmap: makeHeatmap(),
  topIntents: [
    { label: 'ask_price', count: 1842, color: 'var(--zz-lime)' },
    { label: 'ask_availability', count: 1247, color: 'var(--zz-blue)' },
    { label: 'ask_size', count: 892, color: 'var(--zz-pink)' },
    { label: 'ask_shipping', count: 612, color: 'var(--zz-warn)' },
    { label: 'compliment', count: 287, color: 'var(--zz-text-3)' },
    { label: 'complaint', count: 142, color: 'var(--zz-text-3)' },
  ],
  leaderboard: [
    { handle: 'ataka.studio', revenue: 'Rp 187jt', initials: 'AS', color: 'var(--zz-lime)' },
    { handle: 'folkstudio', revenue: 'Rp 89jt', initials: 'F', color: 'var(--zz-pink)' },
    { handle: 'rumah.kebun', revenue: 'Rp 42jt', initials: 'RK', color: 'var(--zz-blue)' },
    { handle: 'ekuator.co', revenue: 'Rp 18jt', initials: 'E', color: 'var(--zz-warn)' },
  ],
};

// ── Mock drilldown data keyed by workflowId ──────────────────────────────────

const mockDrilldowns: Record<string, DrilldownData> = {
  'promo-launch-mei': {
    workflowId: 'promo-launch-mei',
    workflowLabel: 'promo-launch-mei',
    revenue: 'Rp 38,4jt',
    revenueDelta: '+24,6%',
    revenueVsPrior: 'vs Rp 30,8jt · 7d sebelumnya',
    revenuePoints: [
      { day: 'Mon', value: 4.2 },
      { day: 'Tue', value: 5.8 },
      { day: 'Wed', value: 6.4 },
      { day: 'Thu', value: 8.1 },
      { day: 'Fri', value: 6.9 },
      { day: 'Sat', value: 4.1 },
      { day: 'Sun', value: 2.9 },
    ],
    stepConversion: [
      { label: 'Comment received', n: 4280, pct: 100 },
      { label: 'Public reply sent', n: 4124, pct: 96.4 },
      { label: 'DM delivered', n: 3812, pct: 89.1 },
      { label: 'User replied', n: 1924, pct: 44.9 },
      { label: 'Link clicked', n: 982, pct: 22.9 },
      { label: 'Checkout started', n: 412, pct: 9.6 },
      { label: 'Purchase', n: 218, pct: 5.1 },
    ],
    byTime: makeByTime(),
    byPost: [
      { title: 'Bundle Mei 20% off', comments: 1842, revenue: 'Rp 14,2jt' },
      { title: 'Restock sage edition', comments: 1124, revenue: 'Rp 9,8jt' },
      { title: 'Behind the scenes', comments: 718, revenue: 'Rp 6,4jt' },
      { title: 'Limited drop juni', comments: 412, revenue: 'Rp 4,1jt' },
      { title: 'FAQ Q1', comments: 184, revenue: 'Rp 3,9jt' },
    ],
    byIntent: [
      { label: 'ask_price', n: 1284, pct: 30, color: 'var(--zz-lime)' },
      { label: 'ask_link', n: 982, pct: 23, color: 'var(--zz-blue)' },
      { label: 'ask_size', n: 612, pct: 14, color: 'var(--zz-warn)' },
      { label: 'ask_shipping', n: 484, pct: 11, color: 'var(--zz-warn)' },
      { label: 'compliment', n: 412, pct: 9.6, color: 'var(--zz-text-3)' },
      { label: 'other', n: 506, pct: 12.4, color: 'var(--zz-text-3)' },
    ],
  },
};

export function getDrilldownByWorkflowId(id: string): DrilldownData | undefined {
  return mockDrilldowns[id] ?? mockDrilldowns['promo-launch-mei'];
}
