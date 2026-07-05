/**
 * Thin async accessors over mock fixtures (or live API when available).
 * Screens consume these functions — swapping mock→real is a local change here
 * (CLAUDE.md §12a SoC). Layar tidak perlu berubah saat backend terhubung.
 */
import type { Account, ApiEnvelope, CommentOrderResponse, CommentOrderStatDTO, MeResponse, ReservationStatus } from '@zosmed/types';
import { getMe } from '@/lib/get-me';
import { mockAccount } from './account';
import { mockDashboard, type DashboardData } from './dashboard';
import { mockInbox, type InboxData } from './inbox';
import {
  mockContactsData,
  getContactProfileByHandle,
  getContactFallbackProfile,
  type ContactsData,
  type ContactProfileData,
} from './contacts';
import {
  mockAnalyticsData,
  getDrilldownByWorkflowId,
  type AnalyticsData,
  type DrilldownData,
} from './analytics';
import {
  mockBuilder,
  mockRuns,
  mockInspector,
  mockCommentOrder,
  type BuilderData,
  type RunsData,
  type InspectorData,
  type CommentOrderData,
  type IncomingComment,
  type ReservationRow,
} from './workflows';
import {
  mockTemplates,
  mockSettings,
  mockBilling,
  mockTeam,
  mockNotifications,
  mockSafety,
  mockAIStudio,
  type TemplatesData,
  type SettingsData,
  type BillingData,
  type TeamData,
  type NotificationsData,
  type SafetyData,
  type AIStudioData,
} from './system';

// ── API config (satu tempat, jangan hardcode tersebar — §12a-1) ──────────────

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
const DEFAULT_ACCOUNT_ID = process.env.NEXT_PUBLIC_DEFAULT_ACCOUNT_ID ?? '';
const DEFAULT_POST_ID = process.env.NEXT_PUBLIC_DEFAULT_POST_ID ?? '';

/**
 * URL redirect OAuth "Hubungkan Instagram" (`GET /connect/instagram`, backend
 * `apps/api` — migrate-instagram-login.md §3.1/§3.3). Satu sumber dipakai
 * Settings & Onboarding (§12a-1 DRY).
 *
 * Same-origin path (proxied by the Next rewrite, `next.config.ts`) rather than
 * an absolute URL to `API_BASE` — `/connect/instagram` now requires a logged-in
 * session (ADR-003 AC-9); the browser only sends the `zsid` cookie when the
 * navigation stays first-party.
 */
export function getInstagramConnectUrl(): string {
  return '/connect/instagram';
}

// ── Comment-to-Order adapter (presentational mapping — §12a-3 SoC) ───────────

/**
 * Derive warna stat dari key menggunakan design token §11.
 * Backend mengirim key netral; warna adalah urusan presentasi FE.
 */
function deriveStatColor(key: CommentOrderStatDTO['key']): string {
  switch (key) {
    case 'code-detected':
    case 'closed-wa':
      return 'var(--zz-lime)';
    case 'reserved-now':
      return 'var(--zz-warn)';
    case 'expired':
      return 'var(--zz-pink)';
  }
}

/**
 * Derive warna reservasi dari status menggunakan design token §11.
 * Tidak pernah datang dari backend (presentational, §12a-3).
 */
function deriveReservationColor(status: ReservationStatus): string {
  switch (status) {
    case 'reserved':
    case 'waiting-pay':
      return 'var(--zz-warn)';
    case 'closed-wa':
      return 'var(--zz-lime)';
    case 'expired-released':
      return 'var(--zz-pink)';
  }
}

/**
 * Petakan status canonical backend ('expired-released') ke label pendek FE
 * ('expired'). Sesuai catatan di domain.ts §6 dan docs/specs §2.
 */
function mapReservationStatus(status: ReservationStatus): ReservationRow['st'] {
  if (status === 'expired-released') return 'expired';
  return status;
}

/**
 * Adapter tunggal: CommentOrderResponse (backend DTO) → CommentOrderData (shape FE).
 * Komponen layar tidak berubah; hanya file ini yang berubah saat kontrak backend berevolusi.
 */
function adaptCommentOrderResponse(resp: CommentOrderResponse): CommentOrderData {
  return {
    postComments: resp.postCommentsLabel,
    comments: resp.comments.map((c): IncomingComment => {
      // exactOptionalPropertyTypes: optional booleans tidak boleh di-set ke undefined.
      // Hanya sisipkan ok/dupe saat nilainya true.
      const comment: IncomingComment = {
        u: c.user,
        t: c.text,
        tm: c.ago,
        match: c.matchedCode,
      };
      if (c.reserved) comment.ok = true;
      if (c.duplicate) comment.dupe = true;
      return comment;
    }),
    stats: resp.stats.map((s) => ({
      label: s.label,
      value: s.value,
      color: deriveStatColor(s.key),
    })),
    reservations: resp.reservations.map((r) => ({
      code: r.code,
      u: r.buyerHandle,
      p: r.product,
      price: r.priceLabel,
      st: mapReservationStatus(r.status),
      cd: r.countdownLabel,
      color: deriveReservationColor(r.status),
    })),
    products: resp.products.map((p) => ({
      code: p.code,
      name: p.name,
      left: p.stockLeft,
      total: p.stockTotal,
    })),
  };
}

/**
 * Bentuk `Account` (dipakai layar non-auth: dashboard, sidebar, settings) dari
 * `MeResponse` yang sudah nyata (ADR-003) + fallback mock untuk field yang
 * backend belum expose (`id`, `avatarColor`, `connectedAt`, `followerCount`).
 * Auth guard-nya sendiri ada di `(app)/layout.tsx` lewat `getMe()` langsung —
 * fungsi ini murni adaptasi data, tidak pernah redirect (§12a-3 SoC).
 */
export function adaptMeToAccount(me: MeResponse): Account {
  const { user, account } = me;
  return {
    ...mockAccount,
    kit: user.segment ?? mockAccount.kit,
    status: account?.status ?? 'disconnected',
    handle: account?.handle ?? mockAccount.handle,
    displayName: account?.displayName ?? user.email,
  };
}

/**
 * Akun untuk layar non-auth. Coba ambil sesi nyata (`/auth/me`); pakai mock
 * hanya saat sesi tidak ada/gagal (dev tanpa backend) — layar tetap render
 * (pola sama `getCommentOrder`).
 */
export async function getAccount(): Promise<Account> {
  const me = await getMe();
  return me ? adaptMeToAccount(me) : mockAccount;
}

export async function getDashboard(): Promise<DashboardData> {
  return mockDashboard;
}

export async function getInbox(): Promise<InboxData> {
  return mockInbox;
}

export async function getContacts(): Promise<ContactsData> {
  return mockContactsData;
}

export async function getContact(id: string): Promise<ContactProfileData | undefined> {
  return getContactProfileByHandle(id) ?? getContactFallbackProfile(id);
}

export async function getAnalytics(): Promise<AnalyticsData> {
  return mockAnalyticsData;
}

export async function getAnalyticsDrilldown(workflowId: string): Promise<DrilldownData | undefined> {
  return getDrilldownByWorkflowId(workflowId);
}

export async function getWorkflowBuilder(): Promise<BuilderData> {
  return mockBuilder;
}

export async function getWorkflowRuns(): Promise<RunsData> {
  return mockRuns;
}

export async function getWorkflowInspector(): Promise<InspectorData> {
  return mockInspector;
}

/**
 * Ambil data Comment-to-Order dari backend.
 * Fallback ke mock bila backend tidak tersedia (dev/offline) — layar tetap render.
 * Signature dipertahankan agar layar tidak perlu berubah.
 */
export async function getCommentOrder(): Promise<CommentOrderData> {
  const url = new URL('/api/v1/comment-order', API_BASE);
  if (DEFAULT_ACCOUNT_ID) url.searchParams.set('accountId', DEFAULT_ACCOUNT_ID);
  if (DEFAULT_POST_ID) url.searchParams.set('postId', DEFAULT_POST_ID);

  try {
    const res = await fetch(url.toString(), {
      // Revalidate tiap 30 detik supaya antrian reservasi tidak terlalu stale.
      next: { revalidate: 30 },
    });
    if (!res.ok) {
      console.warn(`[getCommentOrder] respons non-2xx (${res.status}), pakai mock data`);
      return mockCommentOrder;
    }
    const envelope: ApiEnvelope<CommentOrderResponse> = await res.json();
    if (envelope.error || !envelope.data) {
      console.warn('[getCommentOrder] API mengembalikan error:', envelope.error);
      return mockCommentOrder;
    }
    return adaptCommentOrderResponse(envelope.data);
  } catch (err) {
    console.warn('[getCommentOrder] fetch gagal, pakai mock data:', err);
    return mockCommentOrder;
  }
}

export async function getTemplates(): Promise<TemplatesData> {
  return mockTemplates;
}

export async function getSettings(): Promise<SettingsData> {
  return mockSettings;
}

export async function getBilling(): Promise<BillingData> {
  return mockBilling;
}

export async function getTeam(): Promise<TeamData> {
  return mockTeam;
}

export async function getNotifications(): Promise<NotificationsData> {
  return mockNotifications;
}

export async function getSafety(): Promise<SafetyData> {
  return mockSafety;
}

export async function getAIStudio(): Promise<AIStudioData> {
  return mockAIStudio;
}
