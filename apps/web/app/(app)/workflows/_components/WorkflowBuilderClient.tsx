'use client';

/**
 * Builder screen state root (ADR-004 F4). Server Component (`[id]/page.tsx`)
 * fetches the initial `Workflow`; this component owns interactive state via
 * `useWorkflowGraph` and composes the existing presentational canvas parts
 * (`Palette`, `FlowCanvas`, `NodeInspector`) — none of them fetch data
 * themselves (§12a-3 SoC).
 */
import { useMemo, useState } from 'react';
import { Button, I, Pill } from '@zosmed/ui';
import { NODE_CATALOG, findCatalogEntry, type Workflow } from '@zosmed/types';
import { PageHeader } from '../../_components/PageHeader';
import { PageHeaderBreadcrumb } from '../../_components/PageHeaderBreadcrumb';
import { Palette } from './Palette';
import { FlowCanvas } from './FlowCanvas';
import { NodeInspector } from './NodeInspector';
import { useWorkflowGraph } from './useWorkflowGraph';
import { buildPaletteSections, nodeToFlowNode, statusPillLabel, statusPillTone } from '@/lib/workflow-catalog';

const paletteSections = buildPaletteSections(NODE_CATALOG);

export function WorkflowBuilderClient({ initialWorkflow }: { initialWorkflow: Workflow }) {
  const g = useWorkflowGraph(initialWorkflow);
  const [nameDraft, setNameDraft] = useState(g.workflow.name);
  const [editingName, setEditingName] = useState(false);

  const flowNodes = useMemo(
    () =>
      g.workflow.nodes.map((n) =>
        nodeToFlowNode(n, { selected: n.id === g.selectedNodeId, runnable: findCatalogEntry(n.node.kind)?.runnable ?? false }),
      ),
    [g.workflow.nodes, g.selectedNodeId],
  );
  const selectedCatalogEntry = g.selectedNode ? findCatalogEntry(g.selectedNode.node.kind) : undefined;
  const canPause = g.workflow.status === 'live';

  return (
    <>
      <PageHeader>
        <div className="flex items-center gap-2.5">
          <PageHeaderBreadcrumb crumbs={[{ label: 'Workflows', href: '/workflows' }, { label: g.workflow.name }]} />
          <button
            type="button"
            className="text-text-3 hover:text-text-2 text-xs transition-colors"
            title="Ganti nama workflow"
            onClick={() => {
              setNameDraft(g.workflow.name);
              setEditingName(true);
            }}
          >
            ✎
          </button>
          <Pill tone={statusPillTone(g.workflow.status)}>{statusPillLabel(g.workflow.status)}</Pill>
          <span className="mono text-text-3 text-[11px]">
            {g.workflow.nodes.length} node · terakhir diperbarui {new Date(g.workflow.updatedAt).toLocaleString('id-ID')}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {g.savedAt ? <span className="mono text-text-3 text-[11px]">Tersimpan {new Date(g.savedAt).toLocaleTimeString('id-ID')}</span> : null}
          {canPause ? (
            <Button variant="ghost" className="px-3 py-[7px] text-xs" disabled={g.pausing} onClick={() => void g.pause()}>
              {g.pausing ? 'Menjeda…' : '⏸ Jeda'}
            </Button>
          ) : null}
          <Button variant="ghost" className="px-3 py-[7px] text-xs" disabled={g.saving} onClick={() => void g.save()}>
            {g.saving ? 'Menyimpan…' : 'Simpan draft'}
          </Button>
          <Button variant="lime" className="px-3 py-[7px] text-xs" disabled={g.publishing} onClick={() => void g.publish()}>
            {g.publishing ? 'Mempublish…' : 'Publish'} <I.arrow />
          </Button>
        </div>
      </PageHeader>

      {g.activateError || g.saveError ? (
        <div
          className="mx-6 mt-3 flex items-center gap-2 rounded-lg px-3 py-2 text-xs"
          style={{ background: 'oklch(0.78 0.2 0 / 0.1)', border: '1px solid oklch(0.78 0.2 0 / 0.3)', color: 'var(--zz-pink)' }}
        >
          <span>⚠</span>
          <span>{g.activateError?.message ?? g.saveError?.message}</span>
        </div>
      ) : null}

      <div className="flex flex-1 overflow-hidden">
        <Palette sections={paletteSections} onAdd={(nodeType) => {
          const entry = findCatalogEntry(nodeType);
          if (entry?.runnable) g.addNode(entry);
        }} />

        {/* Canvas */}
        <div className="bg-bg relative flex-1 overflow-hidden">
          {g.workflow.nodes.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="max-w-xs text-center">
                <p className="text-text-2 text-sm">Canvas masih kosong.</p>
                <p className="text-text-3 mt-1 text-xs">
                  Klik salah satu node di palette kiri (trigger dulu, lalu filter/action) untuk mulai membangun workflow. Seret
                  untuk menata, tarik dari titik kanan node untuk menyambungkan, pilih lalu tekan Delete untuk menghapus.
                </p>
              </div>
            </div>
          ) : (
            <FlowCanvas
              nodes={flowNodes}
              edges={g.workflow.edges}
              onSelectNode={g.selectNode}
              onMoveNode={g.moveNode}
              onConnectNodes={g.addEdge}
              onRemoveEdge={g.removeEdge}
              onRemoveNode={g.removeNode}
            />
          )}
        </div>

        {/* Inspector */}
        <NodeInspector
          node={g.selectedNode}
          catalogEntry={selectedCatalogEntry}
          onChangeConfig={(config) => {
            if (g.selectedNodeId) g.updateNodeConfig(g.selectedNodeId, config);
          }}
          onRemove={() => {
            if (g.selectedNodeId) g.removeNode(g.selectedNodeId);
          }}
        />
      </div>

      {editingName ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditingName(false)}>
          <div className="bg-bg-2 border-line w-[360px] rounded-xl border p-4" onClick={(e) => e.stopPropagation()}>
            <div className="mono tracked text-text-3 mb-2 text-[9.5px]">GANTI NAMA WORKFLOW</div>
            <input
              autoFocus
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              className="bg-bg-3 border-line text-text w-full rounded-lg border px-3 py-2 text-sm outline-none focus:border-lime"
            />
            <div className="mt-3 flex justify-end gap-2">
              <Button variant="ghost" className="px-3 py-[7px] text-xs" onClick={() => setEditingName(false)}>
                Batal
              </Button>
              <Button
                variant="lime"
                className="px-3 py-[7px] text-xs"
                onClick={() => {
                  g.renameWorkflow(nameDraft);
                  setEditingName(false);
                }}
              >
                Simpan
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
