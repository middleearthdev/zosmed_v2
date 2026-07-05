import type { Metadata } from 'next';
import Link from 'next/link';
import { Logo } from '@zosmed/ui';
import { RegisterForm } from './RegisterForm';

export const metadata: Metadata = { title: 'Daftar — Zosmed' };

export default function RegisterPage() {
  return (
    <div className="bg-bg text-text flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-[380px]">
        <div className="mb-8 flex justify-center">
          <Logo size={26} />
        </div>

        <div className="bg-bg-2 border-line rounded-2xl border p-7">
          <span className="mono tracked text-lime text-[11px]">{'// daftar gratis'}</span>
          <h1 className="m-0 mb-1 mt-2 text-2xl font-medium tracking-tight">Mulai dalam 4 menit</h1>
          <p className="text-text-2 m-0 mb-6 text-[13px]">
            Setelah ini kamu pilih jalur (jualan/edukasi/jasa) dan hubungkan Instagram — nol kartu kredit.
          </p>
          <RegisterForm />
        </div>

        <p className="text-text-2 mt-5 text-center text-[13px]">
          Sudah punya akun?{' '}
          <Link href="/login" className="text-lime font-medium">
            Masuk
          </Link>
        </p>
      </div>
    </div>
  );
}
