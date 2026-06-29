import { I } from '@zosmed/ui';
import type { PaletteSection } from '@/lib/mock/workflows';

/** Draggable node palette (left rail of the builder). Static port. */
export function Palette({ sections }: { sections: PaletteSection[] }) {
  return (
    <div className="border-bg-3 w-[240px] overflow-y-auto border-r px-3.5 py-5">
      <div className="bg-bg-2 border-line mb-[18px] flex items-center gap-2 rounded-lg border px-2.5 py-1.5">
        <I.search />
        <span className="text-text-3 text-xs">Search nodes…</span>
      </div>
      {sections.map((sec) => (
        <div key={sec.title} className="mb-[18px]">
          <div className="flex items-center gap-2 px-1.5 pb-2">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: sec.color }} />
            <span className="mono tracked text-text-3 text-[9.5px]">{sec.title}</span>
          </div>
          {sec.items.map((it, i) => (
            <div
              key={`${it.label}-${i}`}
              className="text-text-2 flex cursor-grab items-center gap-2.5 rounded-md px-2 py-[7px] text-[12.5px]"
            >
              <span
                className="bg-bg-2 border-line inline-flex h-6 w-6 items-center justify-center rounded-[5px] border"
                style={{ color: sec.color }}
              >
                {I[it.iconKey]()}
              </span>
              {it.label}
              <span className="text-line-2 ml-auto text-sm">⋮⋮</span>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
