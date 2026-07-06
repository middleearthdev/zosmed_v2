/**
 * Presentational adapters for the workflow builder (ADR-004 F4/F7, §12a-3
 * SoC): map the neutral `NODE_CATALOG` (`@zosmed/types`, mirrors
 * `libs/workflow/nodes/catalog.go`) and real `Workflow` graph shapes into
 * the view-model types the existing canvas components already render
 * (`FlowNode`, `FlowLink`, `PaletteSection` from `lib/mock/workflows`).
 *
 * Colors/icons are NEVER sent by the backend — derived here once (DRY).
 */
import type { IconName } from '@zosmed/ui';
import type { AnyNodeType, NodeCatalogEntry, Workflow, WorkflowEdge, WorkflowNode } from '@zosmed/types';
import { NODE_COLORS, type FlowLink, type FlowNode, type PaletteItem, type PaletteSection, type WfNodeType } from './mock/workflows';

/** node_type → icon key (design token §11 icon set, `@zosmed/ui`). */
const NODE_ICONS: Record<AnyNodeType, IconName> = {
  'comment-received': 'bolt',
  'dm-received': 'inbox',
  'story-reply': 'heart',
  'story-mention': 'sparkle',
  'comment-to-order': 'box',
  'click-to-dm-ad': 'bolt',
  'keyword-match': 'filter',
  'conversation-state': 'chat',
  intent: 'shield',
  'post-selection': 'filter',
  'time-window': 'cog',
  'reply-comment': 'chat',
  'send-dm': 'send',
  'ai-reply': 'ai',
  'send-whatsapp-link': 'whatsapp',
  'send-trust-kit': 'shield',
  'reserve-stock': 'box',
  'notify-optin': 'bell',
  'handoff-human': 'user',
  'tag-contact': 'tag',
  'outbound-webhook': 'cog',
};

const CATEGORY_TO_WF_TYPE: Record<NodeCatalogEntry['category'], WfNodeType> = {
  trigger: 'TRIGGER',
  filter: 'FILTER',
  action: 'ACTION',
};

const CATEGORY_LABEL: Record<NodeCatalogEntry['category'], string> = {
  trigger: 'TRIGGERS',
  filter: 'FILTERS',
  action: 'ACTIONS',
};

export function iconForNodeType(nodeType: string): IconName {
  return NODE_ICONS[nodeType as AnyNodeType] ?? 'bolt';
}

export function wfTypeForCategory(category: NodeCatalogEntry['category']): WfNodeType {
  return CATEGORY_TO_WF_TYPE[category];
}

/** Group the static feasible-node catalog into palette sections (F7). */
export function buildPaletteSections(catalog: readonly NodeCatalogEntry[]): PaletteSection[] {
  const order: NodeCatalogEntry['category'][] = ['trigger', 'filter', 'action'];
  return order.map((category) => ({
    title: CATEGORY_LABEL[category],
    color: NODE_COLORS[CATEGORY_TO_WF_TYPE[category]],
    items: catalog
      .filter((entry) => entry.category === category)
      .map(
        (entry): PaletteItem => ({
          iconKey: iconForNodeType(entry.nodeType),
          label: entry.label,
          nodeType: entry.nodeType,
          runnable: entry.runnable,
        }),
      ),
  }));
}

/** Short, honest sub-line per node — never fabricated counters (CLAUDE.md §4b). */
function summarizeConfig(node: WorkflowNode): string {
  const cfg = node.config as Record<string, unknown>;
  switch (node.node.kind) {
    case 'keyword-match': {
      const keywords = Array.isArray(cfg.keywords) ? (cfg.keywords as string[]) : [];
      return keywords.length ? keywords.map((k) => `"${k}"`).join(' · ') : 'belum ada kata kunci';
    }
    case 'send-whatsapp-link': {
      const phone = typeof cfg.waPhone === 'string' ? cfg.waPhone : '';
      return phone ? `wa.me/${phone}` : 'nomor WA belum diisi';
    }
    default:
      return node.node.category === 'trigger' ? 'trigger' : node.node.category === 'filter' ? 'filter' : 'action';
  }
}

const AUTO_LAYOUT_COL_X: Record<NodeCatalogEntry['category'], number> = {
  trigger: 60,
  filter: 360,
  action: 680,
};
const AUTO_LAYOUT_ROW_H = 140;
const AUTO_LAYOUT_TOP = 60;

/** Compute a position for a freshly-added node (no manual drag positioning in this iteration). */
export function autoLayoutPosition(category: NodeCatalogEntry['category'], indexInCategory: number): { x: number; y: number } {
  return { x: AUTO_LAYOUT_COL_X[category], y: AUTO_LAYOUT_TOP + indexInCategory * AUTO_LAYOUT_ROW_H };
}

/** Map a real `WorkflowNode` to the canvas view-model (`FlowNode`). */
export function nodeToFlowNode(node: WorkflowNode, opts: { selected?: boolean; runnable: boolean }): FlowNode {
  const flow: FlowNode = {
    id: node.id,
    x: node.position.x,
    y: node.position.y,
    type: wfTypeForCategory(node.node.category),
    title: node.label,
    sub: summarizeConfig(node),
    badge: opts.runnable ? 'siap dijalankan' : 'segera hadir',
    iconKey: iconForNodeType(node.node.kind),
  };
  if (opts.selected) flow.selected = true;
  return flow;
}

/** Map `WorkflowEdge[]` to the canvas link view-model (`FlowLink`). */
export function edgesToFlowLinks(edges: WorkflowEdge[]): FlowLink[] {
  return edges.map((e) => ({ from: e.from, to: e.to }));
}

/** Human label for a workflow status pill (§9 — "● LIVE" = workflow active, never IG Live). */
export function statusPillLabel(status: Workflow['status']): string {
  switch (status) {
    case 'live':
      return '● LIVE';
    case 'paused':
      return 'PAUSED';
    case 'error':
      return 'ERROR';
    case 'draft':
    default:
      return 'DRAFT';
  }
}

export function statusPillTone(status: Workflow['status']): 'lime' | 'neutral' | 'pink' | 'warn' {
  switch (status) {
    case 'live':
      return 'lime';
    case 'paused':
      return 'neutral';
    case 'error':
      return 'pink';
    case 'draft':
    default:
      return 'warn';
  }
}
