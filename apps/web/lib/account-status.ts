import type { Account } from '@zosmed/types';
import type { PillTone } from '@zosmed/ui';

/**
 * Satu sumber pemetaan status akun Instagram → tone Pill & label Bahasa Indonesia
 * (CLAUDE.md §12a-1 DRY). Dipakai Settings & Onboarding — jangan duplikasi mapping
 * ini di masing-masing layar.
 *
 * `connected` = token IG-user-scoped valid (§4.0). `expired` = token kadaluarsa,
 * butuh re-auth (migrate-instagram-login.md §4.2). `disconnected` = belum pernah
 * connect / token dicabut.
 */
export const ACCOUNT_STATUS_TONE: Record<Account['status'], PillTone> = {
  connected: 'lime',
  expired: 'pink',
  disconnected: 'neutral',
};

export const ACCOUNT_STATUS_LABEL: Record<Account['status'], string> = {
  connected: 'Terhubung',
  expired: 'Kadaluarsa',
  disconnected: 'Belum terhubung',
};
