'use client';

/** "Buat workflow baru" (F3) — POST create lalu redirect ke builder. */
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, I } from '@zosmed/ui';
import type { Segment } from '@zosmed/types';
import { createWorkflow } from '@/lib/api/workflows';
import { useKit } from '@/lib/kit-context';

const SEGMENT_OPTIONS: { value: Segment; label: string }[] = [
  { value: 'seller', label: 'Jualan (Seller)' },
  { value: 'creator', label: 'Edukasi (Creator)' },
  { value: 'booking', label: 'Jasa (Booking)' },
];

export function NewWorkflowForm() {
  const router = useRouter();
  const defaultSegment = useKit();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [segment, setSegment] = useState<Segment>(defaultSegment);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) {
    return (
      <Button variant="lime" className="px-3 py-[7px] text-xs" onClick={() => setOpen(true)}>
        <I.plus /> Workflow baru
      </Button>
    );
  }

  return (
    <div className="bg-bg-2 border-line flex items-center gap-2 rounded-lg border p-1.5">
      <input
        autoFocus
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Nama workflow, mis. promo-akhir-bulan"
        className="bg-bg-3 border-line text-text placeholder:text-text-3 w-56 rounded-md border px-2.5 py-1.5 text-xs outline-none focus:border-lime"
      />
      <select
        value={segment}
        onChange={(e) => setSegment(e.target.value as Segment)}
        className="bg-bg-3 border-line text-text-2 rounded-md border px-2 py-1.5 text-xs outline-none"
      >
        {SEGMENT_OPTIONS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
      <Button
        variant="lime"
        className="px-3 py-[7px] text-xs"
        disabled={submitting || !name.trim()}
        onClick={async () => {
          setSubmitting(true);
          setError(null);
          const res = await createWorkflow({ name: name.trim(), segment });
          setSubmitting(false);
          if (!res.ok || !res.data) {
            setError(res.error?.message ?? 'Gagal membuat workflow.');
            return;
          }
          router.push(`/workflows/${res.data.id}`);
        }}
      >
        {submitting ? 'Membuat…' : 'Buat'}
      </Button>
      <Button variant="ghost" className="px-2.5 py-[7px] text-xs" onClick={() => setOpen(false)}>
        Batal
      </Button>
      {error ? <span className="text-pink mono text-[10.5px]">{error}</span> : null}
    </div>
  );
}
