'use client';

import { useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input } from '@zosmed/ui';
import { authErrorMessage, register } from '@/lib/auth';

const MIN_PASSWORD_LENGTH = 8;

/**
 * Segmen (jualan/edukasi/jasa) sengaja TIDAK ditanya di sini — itu langkah
 * pertama onboarding (CLAUDE.md §9). Register hanya bikin identitas Zosmed;
 * `user.segment` masih `null` sampai onboarding step 1.
 */
export function RegisterForm() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`Password minimal ${MIN_PASSWORD_LENGTH} karakter.`);
      return;
    }
    if (password !== confirmPassword) {
      setError('Konfirmasi password belum sama.');
      return;
    }

    setSubmitting(true);
    const result = await register(email, password);
    setSubmitting(false);

    if (!result.ok || !result.data) {
      setError(authErrorMessage(result.error));
      return;
    }

    router.push(result.data.user.onboardingCompleted ? '/dashboard' : '/onboarding');
    router.refresh();
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <Input
        label="Email"
        type="email"
        name="email"
        autoComplete="email"
        placeholder="kamu@olshop.id"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        required
      />
      <Input
        label="Password"
        type="password"
        name="password"
        autoComplete="new-password"
        placeholder={`Minimal ${MIN_PASSWORD_LENGTH} karakter`}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
        minLength={MIN_PASSWORD_LENGTH}
      />
      <Input
        label="Konfirmasi password"
        type="password"
        name="confirmPassword"
        autoComplete="new-password"
        placeholder="Ulangi password"
        value={confirmPassword}
        onChange={(e) => setConfirmPassword(e.target.value)}
        required
        minLength={MIN_PASSWORD_LENGTH}
      />

      {error ? (
        <p role="alert" className="text-pink -mt-1 text-[13px]">
          {error}
        </p>
      ) : null}

      <Button type="submit" disabled={submitting} className="mt-1 w-full justify-center">
        {submitting ? 'Membuat akun…' : 'Daftar gratis'}
      </Button>
    </form>
  );
}
