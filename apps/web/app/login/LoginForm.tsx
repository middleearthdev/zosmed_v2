'use client';

import { useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input } from '@zosmed/ui';
import { authErrorMessage, login } from '@/lib/auth';

/**
 * Presentational + local form state only — data-fetching lives in `lib/auth`
 * (§12a-3 SoC). Redirect target after login comes straight from the login
 * response (`user.onboardingCompleted`), no extra `/auth/me` round-trip.
 */
export function LoginForm() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setSubmitting(true);
    const result = await login(email, password);
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
        autoComplete="current-password"
        placeholder="••••••••"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />

      {error ? (
        <p role="alert" className="text-pink -mt-1 text-[13px]">
          {error}
        </p>
      ) : null}

      <Button type="submit" disabled={submitting} className="mt-1 w-full justify-center">
        {submitting ? 'Masuk…' : 'Masuk'}
      </Button>
    </form>
  );
}
