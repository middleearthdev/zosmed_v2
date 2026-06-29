/** Safety & rate-limit surfaces (CLAUDE.md §10 / §4c). */
import type { Id, ISODateTime } from './common';

/** A single quota gauge shown in Safety Center / Dashboard. */
export interface QuotaGauge {
  key:
    | 'comment-replies-hr'
    | 'dm-hr'
    | 'dm-day'
    | 'comments-per-post-5min'
    | 'ai-tokens-day';
  label: string;
  used: number;
  cap: number;
  unit: string;
}

export interface SafetyState {
  gauges: QuotaGauge[];
  autoPaused: boolean;
  killSwitchEngaged: boolean;
  queueDepth: number;
}

export interface SafetyEvent {
  id: Id;
  kind: 'auto-pause' | 'cooldown' | 'kill-switch' | 'queue-overflow' | 'dedupe';
  message: string;
  at: ISODateTime;
}
