/**
 * Server-only reads untuk Workflow REST endpoints (ADR-004 §3, F2).
 *
 * Sama alasannya dengan `lib/get-me.ts`: Server Component fetch tidak punya
 * origin browser implisit, jadi request langsung ke `API_BASE` sambil
 * forward cookie sesi secara manual — TIDAK bisa dibundel ke Client
 * Component (pakai `next/headers`). Mutasi (save/publish/pause/hapus/buat)
 * tetap lewat `lib/api/workflows.ts` (client-safe) dari Client Component.
 *
 * Tiap fungsi mengembalikan fallback aman (array kosong / null) saat
 * backend belum hidup atau sesi tidak valid — layar tetap render (pola sama
 * `getCommentOrder`), bukan melempar error ke pengguna.
 */
import { cookies } from 'next/headers';
import type { ApiEnvelope, RunSummary, Workflow, WorkflowSummary } from '@zosmed/types';
import { API_BASE } from '../env';

async function fetchServer<T>(path: string): Promise<T | null> {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  if (!cookieHeader) return null;

  try {
    const res = await fetch(new URL(path, API_BASE), {
      headers: { cookie: cookieHeader },
      cache: 'no-store',
    });
    if (!res.ok) return null;
    const envelope: ApiEnvelope<T> = await res.json();
    return envelope.data ?? null;
  } catch (err) {
    console.warn(`[workflows.server] fetch gagal (${path}):`, err);
    return null;
  }
}

/** `GET /api/v1/workflows` — dipakai layar daftar (F3). Fallback: array kosong. */
export async function listWorkflowsServer(): Promise<WorkflowSummary[]> {
  return (await fetchServer<WorkflowSummary[]>('/api/v1/workflows')) ?? [];
}

/** `GET /api/v1/workflows/{id}` — dipakai layar builder (F4). Fallback: `null`. */
export async function getWorkflowServer(id: string): Promise<Workflow | null> {
  return fetchServer<Workflow>(`/api/v1/workflows/${id}`);
}

/** `GET /api/v1/runs?limit=` — dipakai layar Runs global (F6). Fallback: array kosong. */
export async function listRunsServer(limit = 50): Promise<RunSummary[]> {
  return (await fetchServer<RunSummary[]>(`/api/v1/runs?limit=${limit}`)) ?? [];
}
