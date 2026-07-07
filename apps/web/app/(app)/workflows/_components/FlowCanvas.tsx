'use client';

/**
 * Workflow builder canvas (ADR-005 §F1–F3). Wraps React Flow (@xyflow/react)
 * to give real drag-repositioning, edge-by-handle creation, and select+Delete
 * removal — the flexibility the hand-rolled SVG canvas lacked. The domain
 * model (`WorkflowNode`/`WorkflowEdge`) is untouched: this component adapts to
 * React Flow's node/edge shapes and reports every mutation back up through
 * callbacks (§12a-3 SoC — no data fetching, no domain writes here).
 *
 * Connection rules (isValidConnection): honour category order
 * trigger→filter→action (never backward) and reject any edge that would form a
 * cycle, so a saved graph always compiles to a DAG (backend activate rejects
 * `cycle` otherwise).
 */
import { useCallback, useEffect } from 'react';
import {
  Background,
  Controls,
  Handle,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type IsValidConnection,
  type Node,
  type NodeProps,
  type NodeTypes,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { I, type IconName } from '@zosmed/ui';
import type { WorkflowEdge } from '@zosmed/types';
import { NODE_COLORS, type FlowNode, type WfNodeType } from '@/lib/mock/workflows';

type ZNodeData = {
  title: string;
  sub: string;
  badge: string;
  iconKey: IconName;
  wfType: WfNodeType;
  kind: string;
};
type ZNode = Node<ZNodeData, 'z'>;

const CATEGORY_RANK: Record<WfNodeType, number> = { TRIGGER: 0, FILTER: 1, ACTION: 2, AI: 2, OUTPUT: 3 };

function toRfNode(n: FlowNode): ZNode {
  return {
    id: n.id,
    type: 'z',
    position: { x: n.x, y: n.y },
    selected: Boolean(n.selected),
    data: { title: n.title, sub: n.sub, badge: n.badge, iconKey: n.iconKey, wfType: n.type, kind: n.type },
  };
}

function ZFlowNode({ data, selected }: NodeProps<ZNode>) {
  const color = NODE_COLORS[data.wfType];
  const border = selected ? color : 'var(--zz-line)';
  return (
    <div
      className="bg-bg-2 overflow-hidden rounded-lg"
      style={{
        width: 176,
        border: `1px solid ${border}`,
        boxShadow: selected ? `0 0 0 2px color-mix(in oklch, ${color} 22%, transparent)` : 'none',
      }}
    >
      <Handle type="target" position={Position.Left} style={{ width: 7, height: 7, background: 'var(--zz-bg)', border: '1px solid var(--zz-line-2)' }} />
      <div
        className="border-line flex items-center gap-1.5 border-b px-2 py-1"
        style={{ background: `color-mix(in oklch, ${color} 12%, transparent)`, color }}
      >
        <span className="inline-flex items-center gap-1 [&_svg]:h-3 [&_svg]:w-3">
          {I[data.iconKey]()}
          <span className="mono tracked text-[8.5px]">{data.kind}</span>
        </span>
      </div>
      <div className="px-2.5 py-2">
        <div className="text-text text-[12px] font-medium leading-tight">{data.title}</div>
        <div className="mono text-text-2 mt-[2px] truncate text-[9.5px]">{data.sub}</div>
        <div className="mt-2 flex items-center gap-1.5">
          <span className="h-[4px] w-[4px] rounded-full" style={{ background: 'var(--zz-lime)' }} />
          <span className="mono text-text-3 text-[9px]">{data.badge}</span>
        </div>
      </div>
      <Handle type="source" position={Position.Right} style={{ width: 7, height: 7, background: color }} />
    </div>
  );
}

const nodeTypes: NodeTypes = { z: ZFlowNode };

export function FlowCanvas({
  nodes,
  edges,
  onSelectNode,
  onMoveNode,
  onConnectNodes,
  onRemoveEdge,
  onRemoveNode,
}: {
  nodes: FlowNode[];
  edges: WorkflowEdge[];
  onSelectNode: (id: string | null) => void;
  onMoveNode: (id: string, x: number, y: number) => void;
  onConnectNodes: (from: string, to: string) => void;
  onRemoveEdge: (id: string) => void;
  onRemoveNode: (id: string) => void;
}) {
  const [rfNodes, setRfNodes, onNodesChange] = useNodesState<ZNode>(nodes.map(toRfNode));
  const [rfEdges, setRfEdges, onEdgesChange] = useEdgesState<Edge>(
    edges.map((e) => ({ id: e.id, source: e.from, target: e.to })),
  );

  // Reconcile React Flow's local state whenever the domain graph changes
  // (node added/removed, config/label edited, positions committed). During a
  // drag the domain graph is stable, so this does not fire mid-gesture.
  useEffect(() => {
    setRfNodes(nodes.map(toRfNode));
  }, [nodes, setRfNodes]);
  useEffect(() => {
    setRfEdges(edges.map((e) => ({ id: e.id, source: e.from, target: e.to })));
  }, [edges, setRfEdges]);

  const isValidConnection = useCallback<IsValidConnection<Edge>>(
    (c) => {
      const source = 'source' in c ? c.source : null;
      const target = 'target' in c ? c.target : null;
      if (!source || !target || source === target) return false;
      const s = nodes.find((n) => n.id === source);
      const t = nodes.find((n) => n.id === target);
      if (!s || !t) return false;
      if (CATEGORY_RANK[s.type] > CATEGORY_RANK[t.type]) return false; // never backward
      // Reject if target can already reach source (adding source→target ⇒ cycle).
      const adj = new Map<string, string[]>();
      for (const e of edges) (adj.get(e.from) ?? adj.set(e.from, []).get(e.from)!).push(e.to);
      const seen = new Set<string>();
      const stack = [target];
      while (stack.length) {
        const cur = stack.pop()!;
        if (cur === source) return false;
        if (seen.has(cur)) continue;
        seen.add(cur);
        for (const nx of adj.get(cur) ?? []) stack.push(nx);
      }
      return true;
    },
    [nodes, edges],
  );

  const onConnect = useCallback(
    (c: Connection) => {
      if (c.source && c.target) onConnectNodes(c.source, c.target);
    },
    [onConnectNodes],
  );

  return (
    <ReactFlow
      colorMode="dark"
      nodes={rfNodes}
      edges={rfEdges}
      nodeTypes={nodeTypes}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeDragStop={(_, node) => onMoveNode(node.id, node.position.x, node.position.y)}
      onNodeClick={(_, node) => onSelectNode(node.id)}
      onPaneClick={() => onSelectNode(null)}
      onConnect={onConnect}
      isValidConnection={isValidConnection}
      onEdgesDelete={(deleted) => deleted.forEach((e) => onRemoveEdge(e.id))}
      onNodesDelete={(deleted) => deleted.forEach((n) => onRemoveNode(n.id))}
      fitView
      fitViewOptions={{ padding: 0.25, maxZoom: 0.9, minZoom: 0.4 }}
      minZoom={0.3}
      maxZoom={1.5}
      proOptions={{ hideAttribution: true }}
      defaultEdgeOptions={{ style: { stroke: 'var(--zz-line-2)', strokeWidth: 1.5 } }}
    >
      <Background color="var(--zz-line)" gap={20} />
      <Controls showInteractive={false} />
    </ReactFlow>
  );
}
