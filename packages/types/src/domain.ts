/**
 * Core domain contracts (CLAUDE.md §6). Frontend-facing shapes consumed by
 * screens via typed mock data; kept aligned with the backend track.
 */
import type { Id, ISODateTime, Segment, WindowState } from './common';

/** Connected IG Business/Creator account. */
export interface Account {
  id: Id;
  handle: string;
  displayName: string;
  avatarColor?: string;
  status: 'connected' | 'expired' | 'disconnected';
  kit: Segment;
  connectedAt: ISODateTime;
  followerCount?: number;
}

// ── Auth / onboarding (ADR-003) ──────────────────────────────────────────────

/**
 * Authenticated Zosmed user — distinct from the IG `Account` (ADR-003 §0.2).
 * `segment` is `null` until chosen in onboarding step 1; `onboardingCompleted`
 * is stamped only once segment is set AND the IG account is `connected`.
 */
export interface AppUser {
  id: Id;
  email: string;
  segment: Segment | null;
  onboardingCompleted: boolean;
}

/**
 * IG account status surfaced by `/auth/me` — never includes token/secret
 * fields (ADR-003 §4.3). Same shape the FE already passes to
 * `InstagramConnectStatus`.
 */
export type AccountStatus = Pick<Account, 'status' | 'handle' | 'displayName'>;

/** `GET /api/v1/auth/me` response shape (ADR-003 §4.3). */
export interface MeResponse {
  user: AppUser;
  account: AccountStatus | null;
}

// ── Workflow ────────────────────────────────────────────────────────────────

export type WorkflowStatus = 'draft' | 'live' | 'paused' | 'error';

/** Node kinds — feasible-only catalog (CLAUDE.md §7). */
export type TriggerKind =
  | 'comment-received'
  | 'dm-received'
  | 'story-reply'
  | 'story-mention'
  | 'comment-to-order'
  | 'click-to-dm-ad';

export type FilterKind =
  | 'keyword-match'
  | 'conversation-state'
  | 'intent'
  | 'post-selection'
  | 'time-window';

export type ActionKind =
  | 'reply-comment'
  | 'send-dm'
  | 'ai-reply'
  | 'send-whatsapp-link'
  | 'send-trust-kit'
  | 'reserve-stock'
  | 'notify-optin'
  | 'handoff-human'
  | 'tag-contact'
  | 'outbound-webhook';

export type NodeKind =
  | { category: 'trigger'; kind: TriggerKind }
  | { category: 'filter'; kind: FilterKind }
  | { category: 'action'; kind: ActionKind };

export interface WorkflowNode {
  id: Id;
  label: string;
  node: NodeKind;
  /** Free-form node configuration (typed per kind in later phases). */
  config: Record<string, unknown>;
  position: { x: number; y: number };
}

export interface WorkflowEdge {
  id: Id;
  from: Id;
  to: Id;
}

export interface Workflow {
  id: Id;
  name: string;
  status: WorkflowStatus;
  segment: Segment;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  updatedAt: ISODateTime;
}

// ── Contacts & conversations ─────────────────────────────────────────────────

export interface Contact {
  id: Id;
  igUserId: string;
  name: string;
  handle?: string;
  avatarColor?: string;
  tags: string[];
  windowState: WindowState;
  leadScore?: number;
  source: ConversationSource;
  lastSeen: ISODateTime;
}

export type ConversationSource = 'comment' | 'dm' | 'story';

export interface Message {
  id: Id;
  author: 'contact' | 'ai' | 'human';
  text: string;
  at: ISODateTime;
}

export interface Conversation {
  id: Id;
  contactId: Id;
  source: ConversationSource;
  windowState: WindowState;
  messages: Message[];
  handledBy: 'ai' | 'human';
}

// ── Comment-to-Order ─────────────────────────────────────────────────────────

/** Reservation lifecycle (CLAUDE.md §6 / §8.1). */
export type ReservationStatus =
  | 'reserved'
  | 'waiting-pay'
  | 'closed-wa'
  | 'expired-released';

export interface Reservation {
  id: Id;
  code: string;
  product: string;
  contactId: Id;
  status: ReservationStatus;
  reservedAt: ISODateTime;
  expiresAt: ISODateTime;
}

// ── Opt-in / notifications / runs / trust ────────────────────────────────────

export interface OptIn {
  id: Id;
  contactId: Id;
  topic: string;
  optedInAt: ISODateTime;
}

export type RunStatus = 'success' | 'failed' | 'skipped' | 'queued' | 'running';

export interface RunLog {
  id: Id;
  workflowId: Id;
  nodeId: Id;
  status: RunStatus;
  message: string;
  at: ISODateTime;
}

export interface TrustAsset {
  id: Id;
  kind: 'testimoni' | 'real-pict' | 'resi';
  label: string;
  url: string;
}
