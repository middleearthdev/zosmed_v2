import { listRunsServer } from '@/lib/api/workflows.server';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';
import { RunsClient } from './_components/RunsClient';

/** Runs global — semua workflow milik akun (ADR-004 F6, `GET /api/v1/runs`). */
export default async function WorkflowRunsPage() {
  const runs = await listRunsServer(50);

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: 'Runs' }]} />
        </div>
      </PageHeader>

      <RunsClient runs={runs} />
    </>
  );
}
