'use client';

/**
 * Canvas state for the workflow builder (ADR-004 F4, §12a-3 SoC).
 * Owns nodes/edges/selection/dirty-tracking + calls the data layer
 * (`lib/api/workflows.ts`) for save/publish/pause. Presentational
 * components (`Palette`, `FlowCanvas`, `NodeInspector`) stay pure — they
 * only read state and call the actions this hook returns.
 */
import { useCallback, useMemo, useState } from 'react';
import type {
  ApiErrorShape,
  NodeCatalogEntry,
  Workflow,
  WorkflowEdge,
  WorkflowNode,
} from '@zosmed/types';
import { VALIDATION_FAILURE_MESSAGES, type ValidationFailureReason } from '@zosmed/types';
import { activateWorkflow, pauseWorkflow, saveWorkflow } from '@/lib/api/workflows';
import { autoLayoutPosition } from '@/lib/workflow-catalog';

function defaultConfigFor(nodeType: string): Record<string, unknown> {
  switch (nodeType) {
    case 'keyword-match':
      return { keywords: [], caseInsensitive: true };
    case 'send-whatsapp-link':
      return { template: 'Halo {{nama}}, makasih udah komen di {{post}}! Yuk lanjut chat di WhatsApp ya 💬', waPhone: '' };
    default:
      return {};
  }
}

function newNodeId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : `tmp-${Date.now()}-${Math.random()}`;
}

/** Auto-wire a freshly-added node into the existing chain (trigger→filter→action). */
function autoWireEdges(nodes: WorkflowNode[], newNode: WorkflowNode): WorkflowEdge[] {
  const triggers = nodes.filter((n) => n.node.category === 'trigger' && n.id !== newNode.id);
  const filters = nodes.filter((n) => n.node.category === 'filter' && n.id !== newNode.id);
  const actions = nodes.filter((n) => n.node.category === 'action' && n.id !== newNode.id);
  const mk = (from: string, to: string): WorkflowEdge => ({ id: newNodeId(), from, to });
  const created: WorkflowEdge[] = [];

  const lastFilter = filters[filters.length - 1];
  const lastAction = actions[actions.length - 1];

  if (newNode.node.category === 'filter') {
    if (lastFilter) {
      created.push(mk(lastFilter.id, newNode.id));
    } else {
      for (const t of triggers) created.push(mk(t.id, newNode.id));
    }
  } else if (newNode.node.category === 'action') {
    if (lastAction) {
      created.push(mk(lastAction.id, newNode.id));
    } else if (lastFilter) {
      created.push(mk(lastFilter.id, newNode.id));
    } else {
      for (const t of triggers) created.push(mk(t.id, newNode.id));
    }
  } else if (newNode.node.category === 'trigger') {
    const target = filters[0] ?? actions[0];
    if (target) created.push(mk(newNode.id, target.id));
  }
  return created;
}

export interface WorkflowGraphState {
  workflow: Workflow;
  selectedNodeId: string | null;
  selectedNode: WorkflowNode | null;
  dirty: boolean;
  saving: boolean;
  publishing: boolean;
  pausing: boolean;
  savedAt: number | null;
  activateError: { reason?: ValidationFailureReason; message: string } | null;
  saveError: ApiErrorShape | null;
  selectNode: (id: string | null) => void;
  addNode: (entry: NodeCatalogEntry) => void;
  removeNode: (id: string) => void;
  updateNodeConfig: (id: string, config: Record<string, unknown>) => void;
  renameWorkflow: (name: string) => void;
  save: () => Promise<boolean>;
  publish: () => Promise<boolean>;
  pause: () => Promise<boolean>;
}

export function useWorkflowGraph(initial: Workflow): WorkflowGraphState {
  const [workflow, setWorkflow] = useState<Workflow>(initial);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [pausing, setPausing] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [activateError, setActivateError] = useState<WorkflowGraphState['activateError']>(null);
  const [saveError, setSaveError] = useState<ApiErrorShape | null>(null);

  const selectedNode = useMemo(
    () => workflow.nodes.find((n) => n.id === selectedNodeId) ?? null,
    [workflow.nodes, selectedNodeId],
  );

  const selectNode = useCallback((id: string | null) => setSelectedNodeId(id), []);

  const addNode = useCallback((entry: NodeCatalogEntry) => {
    setWorkflow((prev) => {
      const countInCategory = prev.nodes.filter((n) => n.node.category === entry.category).length;
      const node: WorkflowNode = {
        id: newNodeId(),
        label: entry.label,
        node: { category: entry.category, kind: entry.nodeType } as WorkflowNode['node'],
        config: defaultConfigFor(entry.nodeType),
        position: autoLayoutPosition(entry.category, countInCategory),
      };
      const newEdges = autoWireEdges(prev.nodes, node);
      setSelectedNodeId(node.id);
      return { ...prev, nodes: [...prev.nodes, node], edges: [...prev.edges, ...newEdges] };
    });
    setDirty(true);
  }, []);

  const removeNode = useCallback(
    (id: string) => {
      setWorkflow((prev) => ({
        ...prev,
        nodes: prev.nodes.filter((n) => n.id !== id),
        edges: prev.edges.filter((e) => e.from !== id && e.to !== id),
      }));
      setSelectedNodeId((cur) => (cur === id ? null : cur));
      setDirty(true);
    },
    [],
  );

  const updateNodeConfig = useCallback((id: string, config: Record<string, unknown>) => {
    setWorkflow((prev) => ({
      ...prev,
      nodes: prev.nodes.map((n) => (n.id === id ? { ...n, config: { ...n.config, ...config } } : n)),
    }));
    setDirty(true);
  }, []);

  const renameWorkflow = useCallback((name: string) => {
    setWorkflow((prev) => ({ ...prev, name }));
    setDirty(true);
  }, []);

  const save = useCallback(async (): Promise<boolean> => {
    setSaving(true);
    setSaveError(null);
    const res = await saveWorkflow(workflow.id, { name: workflow.name, nodes: workflow.nodes, edges: workflow.edges });
    setSaving(false);
    if (!res.ok || !res.data) {
      setSaveError(res.error ?? { code: 'unknown_error', message: 'Gagal menyimpan draft.' });
      return false;
    }
    setWorkflow(res.data);
    setDirty(false);
    setSavedAt(Date.now());
    return true;
  }, [workflow.id, workflow.name, workflow.nodes, workflow.edges]);

  const publish = useCallback(async (): Promise<boolean> => {
    setPublishing(true);
    setActivateError(null);
    // Publish selalu menyimpan draft dulu supaya validasi backend melihat graf terbaru.
    const saved = await save();
    if (!saved) {
      setPublishing(false);
      return false;
    }
    const res = await activateWorkflow(workflow.id);
    setPublishing(false);
    if (!res.ok || !res.data) {
      const reason = res.error?.reason as ValidationFailureReason | undefined;
      const message = reason ? VALIDATION_FAILURE_MESSAGES[reason] : res.error?.message ?? 'Gagal mengaktifkan workflow.';
      setActivateError({ ...(reason ? { reason } : {}), message });
      return false;
    }
    setWorkflow(res.data);
    return true;
  }, [save, workflow.id]);

  const pause = useCallback(async (): Promise<boolean> => {
    setPausing(true);
    const res = await pauseWorkflow(workflow.id);
    setPausing(false);
    if (!res.ok || !res.data) return false;
    setWorkflow(res.data);
    return true;
  }, [workflow.id]);

  return {
    workflow,
    selectedNodeId,
    selectedNode,
    dirty,
    saving,
    publishing,
    pausing,
    savedAt,
    activateError,
    saveError,
    selectNode,
    addNode,
    removeNode,
    updateNodeConfig,
    renameWorkflow,
    save,
    publish,
    pause,
  };
}
