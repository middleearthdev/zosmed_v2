import { I } from '@zosmed/ui';

export interface ConnectedBannerProps {
  /** Where "Tutup" navigates — Settings (`/settings`) or Onboarding (`/onboarding`). */
  closeHref?: string;
}

/**
 * Feedback sukses setelah redirect balik dari OAuth Instagram
 * (`/settings?connected=1` atau `/onboarding?connected=1`, ADR-002 §3.1 /
 * ADR-003 §6). Dipakai Settings & Onboarding — satu komponen, DRY (§12a-1).
 *
 * Server-rendered, tanpa state client: tombol "Tutup" cukup navigasi ke URL
 * tanpa query param (§12a-4 — hindari client JS/toast lib untuk satu banner).
 */
export function ConnectedBanner({ closeHref = '/settings' }: ConnectedBannerProps) {
  return (
    <div
      className="mb-5 flex items-center gap-2.5 rounded-xl border px-4 py-3"
      style={{ background: 'var(--zz-lime-soft)', borderColor: 'oklch(0.9 0.2 130 / 0.3)' }}
    >
      <span className="text-lime inline-flex h-5 w-5 flex-shrink-0 items-center justify-center">
        <I.check />
      </span>
      <span className="text-text flex-1 text-sm">
        Instagram berhasil terhubung <span aria-hidden>✓</span> Workflow siap dijalankan otomatis.
      </span>
      <a href={closeHref} className="mono text-text-3 flex-shrink-0 text-[11px]">
        Tutup
      </a>
    </div>
  );
}
