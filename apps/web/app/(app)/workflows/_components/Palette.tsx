import { I } from '@zosmed/ui';
import type { PaletteSection } from '@/lib/mock/workflows';

/**
 * Draggable-in-spirit node palette (left rail of the builder). Click a
 * runnable item to append it to the canvas; non-runnable items (R6, ADR-004)
 * show a "segera" badge and are disabled — they still surface the full §7
 * catalog without letting users build something that can't go `live`.
 */
export function Palette({ sections, onAdd }: { sections: PaletteSection[]; onAdd?: (nodeType: string) => void }) {
  return (
    <div className="border-bg-3 w-[240px] overflow-y-auto border-r px-3.5 py-5">
      <div className="bg-bg-2 border-line mb-[18px] flex items-center gap-2 rounded-lg border px-2.5 py-1.5">
        <I.search />
        <span className="text-text-3 text-xs">Cari node…</span>
      </div>
      {sections.map((sec) => (
        <div key={sec.title} className="mb-[18px]">
          <div className="flex items-center gap-2 px-1.5 pb-2">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: sec.color }} />
            <span className="mono tracked text-text-3 text-[9.5px]">{sec.title}</span>
          </div>
          {sec.items.map((it, i) => (
            <button
              key={`${it.label}-${i}`}
              type="button"
              disabled={!it.runnable}
              onClick={() => onAdd?.(it.nodeType)}
              className="text-text-2 flex w-full items-center gap-2.5 rounded-md px-2 py-[7px] text-left text-[12.5px] disabled:cursor-not-allowed disabled:opacity-50"
              style={{ cursor: it.runnable ? 'pointer' : 'not-allowed' }}
              title={it.runnable ? `Tambah node "${it.label}"` : 'Segera hadir — belum bisa dijalankan otomatis'}
            >
              <span
                className="bg-bg-2 border-line inline-flex h-6 w-6 items-center justify-center rounded-[5px] border"
                style={{ color: sec.color }}
              >
                {I[it.iconKey]()}
              </span>
              <span className="flex-1">{it.label}</span>
              {it.runnable ? (
                <span className="text-line-2 text-sm">+</span>
              ) : (
                <span className="mono text-text-3 rounded-full px-1.5 py-[1px] text-[9px]" style={{ background: 'var(--zz-bg-3)' }}>
                  segera
                </span>
              )}
            </button>
          ))}
        </div>
      ))}
    </div>
  );
}
