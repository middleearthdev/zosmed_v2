import type { Account } from '@zosmed/types';
import { I, Pill, cn } from '@zosmed/ui';
import { ACCOUNT_STATUS_LABEL, ACCOUNT_STATUS_TONE } from '@/lib/account-status';

export interface InstagramConnectStatusProps {
  /** `null` = belum pernah connect sama sekali — diperlakukan sebagai `disconnected`. */
  account: Pick<Account, 'status' | 'handle' | 'displayName'> | null;
  /** URL redirect OAuth backend (`GET /connect/instagram`). */
  connectUrl: string;
  className?: string;
}

/**
 * Status koneksi akun Instagram + CTA hubungkan/hubungkan-ulang
 * (migrate-instagram-login.md §3, §4.2 — item Fase F).
 *
 * Presentational murni: data (`account`, `connectUrl`) disuplai pemanggil,
 * komponen ini tidak fetch apa pun (SoC §12a-3). Dipakai di Settings & di
 * Onboarding agar copy dan pemetaan status konsisten (§12a-1 DRY).
 */
export function InstagramConnectStatus({ account, connectUrl, className }: InstagramConnectStatusProps) {
  const status = account?.status ?? 'disconnected';
  const tone = ACCOUNT_STATUS_TONE[status];
  const label = ACCOUNT_STATUS_LABEL[status];

  return (
    <div className={cn('flex flex-wrap items-center gap-3', className)}>
      <Pill tone={tone}>{label.toUpperCase()}</Pill>

      {status === 'connected' && account ? (
        <span className="text-sm">
          <span className="font-medium">{account.displayName}</span>{' '}
          <span className="mono text-text-3 text-[12px]">@{account.handle}</span>
        </span>
      ) : (
        <span className="text-text-2 max-w-[360px] text-[13px]">
          {status === 'expired'
            ? 'Sesi Instagram kadaluarsa — hubungkan ulang biar workflow lanjut jalan otomatis.'
            : 'Hubungkan akun Instagram Business/Creator kamu biar komentar & DM bisa dibalas otomatis.'}
        </span>
      )}

      {status !== 'connected' ? (
        <a href={connectUrl} className="btn-lime ml-auto text-xs">
          {status === 'expired' ? 'Hubungkan ulang' : 'Hubungkan Instagram'} <I.arrow />
        </a>
      ) : null}
    </div>
  );
}
