import { notFound } from 'next/navigation';
import { getWorkflowServer } from '@/lib/api/workflows.server';
import { WorkflowBuilderClient } from '../_components/WorkflowBuilderClient';

/** Builder screen (ADR-004 F4) — loads the persisted graph, hands off to the client canvas. */
export default async function WorkflowBuilderPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const workflow = await getWorkflowServer(id);
  if (!workflow) notFound();

  return <WorkflowBuilderClient initialWorkflow={workflow} />;
}
