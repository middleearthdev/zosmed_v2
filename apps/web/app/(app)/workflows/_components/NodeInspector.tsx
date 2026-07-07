'use client';

/**
 * Right-hand config panel (ADR-004 F5 → ADR-005 §F4). Fully schema-driven:
 * renders `SchemaForm` from the selected node's `configSchema` (`@zosmed/types`,
 * mirrors `libs/workflow/nodes/catalog.go`). Adding a new runnable node needs
 * NO new inspector code — the catalog schema drives the form (§12a DRY).
 * Config always flows back into canvas state via `onChangeConfig` (never
 * fetched mid-JSX, §12a-3 SoC).
 */
import { I, type IconName } from '@zosmed/ui';
import type { NodeCatalogEntry, WorkflowNode } from '@zosmed/types';
import { iconForNodeType, wfTypeForCategory } from '@/lib/workflow-catalog';
import { NODE_COLORS } from '@/lib/mock/workflows';
import { SchemaForm } from './inspector/SchemaForm';

export function NodeInspector({
  node,
  catalogEntry,
  onChangeConfig,
  onRemove,
}: {
  node: WorkflowNode | null;
  catalogEntry: NodeCatalogEntry | undefined;
  onChangeConfig: (config: Record<string, unknown>) => void;
  onRemove: () => void;
}) {
  if (!node) {
    return (
      <></>
      // <div className="border-bg-3 flex w-[320px] flex-col items-center justify-center gap-2 border-l px-6 py-10 text-center">
      //   <span className="text-text-3 text-xs">Pilih node di canvas untuk mengatur konfigurasinya.</span>
      //   <span className="text-text-3 mono text-[10.5px]">Atau klik salah satu node di palette untuk menambah node baru.</span>
      // </div>
    );
  }

  const color = NODE_COLORS[wfTypeForCategory(node.node.category)];
  const iconKey: IconName = iconForNodeType(node.node.kind);
  const schema = catalogEntry?.configSchema;

  return (
    <div className="border-bg-3 w-[320px] overflow-y-auto border-l px-[18px] py-5">
      <div className="mb-1 flex items-center gap-2">
        <span
          className="inline-flex h-7 w-7 items-center justify-center rounded-md"
          style={{ background: `color-mix(in oklch, ${color} 14%, transparent)`, color }}
        >
          {I[iconKey]()}
        </span>
        <span className="mono tracked text-[10px]" style={{ color }}>
          {node.node.category.toUpperCase()} · {node.node.kind}
        </span>
      </div>
      <h3 className="mb-1 mt-1.5 text-lg font-medium">{node.label}</h3>
      <p className="text-text-3 m-0 text-xs">{catalogEntry?.description ?? 'Node dari katalog feasible (CLAUDE.md §7).'}</p>

      <div className="bg-bg-3 my-5 h-px" />

      {!catalogEntry?.runnable ? (
        <div
          className="mb-[18px] flex items-start gap-2 rounded-[7px] px-2.5 py-2"
          style={{ background: 'oklch(0.85 0.16 75 / 0.1)', border: '1px solid oklch(0.85 0.16 75 / 0.3)' }}
        >
          <span className="mt-px text-xs" style={{ color: 'var(--zz-warn)' }}>
            ⓘ
          </span>
          <span className="text-text-2 text-[11px] leading-normal">
            Node ini belum bisa dijalankan otomatis (segera hadir). Workflow tidak bisa dipublish kalau node ini jadi satu-satunya
            trigger/action.
          </span>
        </div>
      ) : null}

      {schema && schema.length > 0 ? (
        // key=node.id so uncontrolled inputs remount when a different node is selected.
        <SchemaForm key={node.id} schema={schema} config={node.config as Record<string, unknown>} onChange={onChangeConfig} />
      ) : (
        <div className="mb-[18px]">
          <div className="mono tracked text-text-3 mb-2 text-[9.5px]">KONFIGURASI</div>
          <div className="bg-bg-2 border-line text-text-2 rounded-lg border p-3 text-[12.5px] leading-normal">
            {catalogEntry?.runnable
              ? 'Node ini tidak butuh konfigurasi tambahan — siap dipakai langsung di workflow.'
              : 'Konfigurasi untuk node ini menyusul di iterasi berikutnya.'}
          </div>
        </div>
      )}

      <div className="bg-bg-3 my-5 h-px" />

      <button
        type="button"
        onClick={onRemove}
        className="mono w-full rounded-md border px-3 py-2 text-center text-xs transition-colors"
        style={{ borderColor: 'oklch(0.78 0.2 0 / 0.35)', color: 'var(--zz-pink)' }}
      >
        Hapus node
      </button>
    </div>
  );
}
