/**
 * Client-safe data layer untuk Workflow REST endpoints (ADR-004 §3, F2).
 *
 * Dipakai dari Client Component (builder interaktif, tombol Save/Publish/
 * Pause/Hapus, list "buat workflow baru") — fetch relative `/api/v1/...`
 * yang diproxy same-origin oleh `next.config.ts` sehingga cookie sesi `zsid`
 * ikut otomatis (`credentials: 'include'`), sama seperti `lib/auth.ts`.
 *
 * TIDAK mengimpor `next/headers` — file ini boleh dibundel ke Client
 * Component. Untuk pembacaan awal di Server Component (list/detail/runs
 * page), pakai `lib/api/workflows.server.ts` (perlu forward cookie manual,
 * pola sama `lib/get-me.ts`).
 *
 * SoC (§12a-3): layar (page.tsx / komponen) tidak fetch langsung di JSX;
 * semua mutasi/pembacaan client-side lewat fungsi di file ini.
 */
import type {
  ApiEnvelope,
  ApiErrorShape,
  CreateWorkflowRequest,
  DeleteWorkflowResponse,
  RunSummary,
  SaveWorkflowRequest,
  Workflow,
  WorkflowSummary,
} from '@zosmed/types';

export type ApiError = ApiErrorShape;

export interface ActionResult<T> {
  ok: boolean;
  data?: T;
  error?: ApiError;
}

const NETWORK_ERROR: ApiError = {
  code: 'network_error',
  message: 'Tidak bisa terhubung ke server. Coba lagi.',
};

const UNKNOWN_ERROR: ApiError = {
  code: 'unknown_error',
  message: 'Terjadi kesalahan. Coba lagi.',
};

/**
 * Fetch same-origin `/api/v1/...` dan bungkus jadi `ActionResult`. Satu
 * helper untuk semua method (GET/POST/PUT/DELETE) dipakai fungsi di bawah
 * (§12a-1 DRY) — mirror `lib/auth.ts` `request()` tapi mendukung GET/DELETE
 * dan query string, karena layar workflow butuh keduanya.
 */
async function request<T>(path: string, init?: RequestInit): Promise<ActionResult<T>> {
  try {
    const res = await fetch(path, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      ...init,
    });
    const envelope: ApiEnvelope<T> = await res.json();
    if (!res.ok || envelope.error) {
      return { ok: false, error: envelope.error ?? UNKNOWN_ERROR };
    }
    return envelope.data !== null ? { ok: true, data: envelope.data } : { ok: true };
  } catch {
    return { ok: false, error: NETWORK_ERROR };
  }
}

/** `GET /api/v1/workflows` — daftar workflow milik akun (F3). */
export function listWorkflows(): Promise<ActionResult<WorkflowSummary[]>> {
  return request<WorkflowSummary[]>('/api/v1/workflows');
}

/** `POST /api/v1/workflows` — buat workflow baru (graf kosong / preset segmen). */
export function createWorkflow(body: CreateWorkflowRequest): Promise<ActionResult<Workflow>> {
  return request<Workflow>('/api/v1/workflows', { method: 'POST', body: JSON.stringify(body) });
}

/** `GET /api/v1/workflows/{id}` — graf lengkap (nodes+edges+config) untuk builder. */
export function getWorkflow(id: string): Promise<ActionResult<Workflow>> {
  return request<Workflow>(`/api/v1/workflows/${id}`);
}

/** `PUT /api/v1/workflows/{id}` — simpan draft (replace penuh node/edge, ADR-004 §8 R4). */
export function saveWorkflow(id: string, body: SaveWorkflowRequest): Promise<ActionResult<Workflow>> {
  return request<Workflow>(`/api/v1/workflows/${id}`, { method: 'PUT', body: JSON.stringify(body) });
}

/** `DELETE /api/v1/workflows/{id}`. */
export function deleteWorkflow(id: string): Promise<ActionResult<DeleteWorkflowResponse>> {
  return request<DeleteWorkflowResponse>(`/api/v1/workflows/${id}`, { method: 'DELETE' });
}

/**
 * `POST /api/v1/workflows/{id}/activate` — publish jadi `live`. Bisa gagal
 * 422 `validation_failed` dengan `error.reason` (lihat `VALIDATION_FAILURE_MESSAGES`
 * di `@zosmed/types` untuk copy Bahasa Indonesia per alasan).
 */
export function activateWorkflow(id: string): Promise<ActionResult<Workflow>> {
  return request<Workflow>(`/api/v1/workflows/${id}/activate`, { method: 'POST' });
}

/** `POST /api/v1/workflows/{id}/pause` — kembalikan ke `paused`. */
export function pauseWorkflow(id: string): Promise<ActionResult<Workflow>> {
  return request<Workflow>(`/api/v1/workflows/${id}/pause`, { method: 'POST' });
}

/** `GET /api/v1/workflows/{id}/runs?limit=` — riwayat run satu workflow. */
export function listWorkflowRuns(id: string, limit = 50): Promise<ActionResult<RunSummary[]>> {
  return request<RunSummary[]>(`/api/v1/workflows/${id}/runs?limit=${limit}`);
}

/** `GET /api/v1/runs?limit=` — riwayat run semua workflow milik akun (layar Runs global). */
export function listRuns(limit = 50): Promise<ActionResult<RunSummary[]>> {
  return request<RunSummary[]>(`/api/v1/runs?limit=${limit}`);
}
