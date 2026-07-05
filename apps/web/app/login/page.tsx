import type { Metadata } from 'next';
import Link from 'next/link';
import { Logo } from '@zosmed/ui';
import { LoginForm } from './LoginForm';

export const metadata: Metadata = { title: 'Masuk — Zosmed' };

export default function LoginPage() {
  return (
    <div className="bg-bg text-text flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-[380px]">
        <div className="mb-8 flex justify-center">
          <Logo size={26} />
        </div>

        <div className="bg-bg-2 border-line rounded-2xl border p-7">
          <span className="mono tracked text-lime text-[11px]">{'// masuk'}</span>
          <h1 className="m-0 mb-1 mt-2 text-2xl font-medium tracking-tight">Selamat datang kembali</h1>
          <p className="text-text-2 m-0 mb-6 text-[13px]">Masuk untuk lanjut kelola workflow, inbox, dan komentar kamu.</p>
          <LoginForm />
        </div>

        <p className="text-text-2 mt-5 text-center text-[13px]">
          Belum punya akun?{' '}
          <Link href="/register" className="text-lime font-medium">
            Daftar gratis
          </Link>
        </p>
      </div>
    </div>
  );
}
