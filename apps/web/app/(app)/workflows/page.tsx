import Link from 'next/link';
import { Card, I, Pill } from '@zosmed/ui';
import type { WorkflowStatus } from '@zosmed/types';
import { listWorkflowsServer } from '@/lib/api/workflows.server';
import { PageHeader } from '../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../_components/PageHeaderBreadcrumb';
import { NewWorkflowForm } from './_components/NewWorkflowForm';

const STATUS_TONE: Record<WorkflowStatus, 'lime' | 'neutral' | 'pink' | 'warn'> = {
  live: 'lime',
  paused: 'neutral',
  error: 'pink',
  draft: 'warn',
};

const STATUS_LABEL: Record<WorkflowStatus, string> = {
  live: '● LIVE',
  paused: 'PAUSED',
  error: 'ERROR',
  draft: 'DRAFT',
};

const SEGMENT_LABEL: Record<string, string> = {
  seller: 'Seller',
  creator: 'Creator',
  booking: 'Booking',
};

/** Daftar workflow milik akun (F3) — pintu masuk ke builder per workflow. */
export default async function WorkflowsIndexPage() {
  const workflows = await listWorkflowsServer();

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb crumbs={[{ label: 'Workflows' }]} />
        </div>
        <NewWorkflowForm />
      </PageHeader>

      <div className="zz-scroll flex-1 overflow-y-auto p-6">
        {workflows.length === 0 ? (
          <Card className="flex flex-col items-center gap-3 py-14 text-center">
            <span className="bg-bg-3 text-lime inline-flex h-11 w-11 items-center justify-center rounded-full">
              <I.workflow />
            </span>
            <div>
              <p className="text-text m-0 text-sm font-medium">Belum ada workflow</p>
              <p className="text-text-3 m-0 mt-1 text-xs">Buat workflow pertama untuk mulai ubah komentar/DM jadi hasil.</p>
            </div>
            <NewWorkflowForm />
          </Card>
        ) : (
          <div className="grid grid-cols-3 gap-3.5">
            {workflows.map((w) => (
              <Link key={w.id} href={`/workflows/${w.id}`} className="block">
                <Card className="h-full transition-colors hover:border-line-2">
                  <div className="mb-2.5 flex items-start justify-between gap-2">
                    <span className="text-text truncate text-sm font-medium">{w.name}</span>
                    <Pill tone={STATUS_TONE[w.status]}>{STATUS_LABEL[w.status]}</Pill>
                  </div>
                  <div className="text-text-3 mono flex items-center gap-2 text-[11px]">
                    <span>{SEGMENT_LABEL[w.segment] ?? w.segment}</span>
                    <span>·</span>
                    <span>{w.nodeCount} node</span>
                  </div>
                  <div className="text-text-3 mono mt-2.5 text-[10.5px]">
                    diperbarui {new Date(w.updatedAt).toLocaleString('id-ID')}
                  </div>
                </Card>
              </Link>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
