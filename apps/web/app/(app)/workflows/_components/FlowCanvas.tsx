import { I } from '@zosmed/ui';
import { NODE_COLORS, type FlowLink, type FlowNode } from '@/lib/mock/workflows';

const NW = 220;
const NH = 84;

/** Static port of the design BigFlow node graph (SVG bezier links + node cards). */
export function FlowCanvas({ nodes, links }: { nodes: FlowNode[]; links: FlowLink[] }) {
  const byId = (id: string) => nodes.find((n) => n.id === id);

  return (
    <div className="absolute inset-0">
      <svg width="100%" height="100%" className="absolute inset-0">
        <defs>
          <marker id="arr" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M0,0 L10,5 L0,10 z" fill="var(--zz-line-2)" />
          </marker>
          <marker id="arr-l" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="8" markerHeight="8" orient="auto">
            <path d="M0,0 L10,5 L0,10 z" fill="var(--zz-lime)" />
          </marker>
        </defs>
        {links.map((lnk) => {
          const A = byId(lnk.from);
          const B = byId(lnk.to);
          if (!A || !B) return null;
          const x1 = A.x + NW;
          const y1 = A.y + NH / 2;
          const x2 = B.x;
          const y2 = B.y + NH / 2;
          const mx = (x1 + x2) / 2;
          return (
            <path
              key={`${lnk.from}-${lnk.to}`}
              d={`M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`}
              stroke={lnk.active ? 'var(--zz-lime)' : 'var(--zz-line-2)'}
              strokeWidth={lnk.active ? 2 : 1.5}
              strokeDasharray={lnk.active ? '4 4' : undefined}
              fill="none"
              markerEnd={`url(#${lnk.active ? 'arr-l' : 'arr'})`}
            >
              {lnk.active ? (
                <animate attributeName="stroke-dashoffset" from="0" to="-16" dur="0.8s" repeatCount="indefinite" />
              ) : null}
            </path>
          );
        })}
      </svg>

      {nodes.map((n) => {
        const color = NODE_COLORS[n.type];
        const border = n.focus ? color : n.selected ? 'var(--zz-line-2)' : 'var(--zz-line)';
        return (
          <div
            key={n.id}
            className="bg-bg-2 absolute overflow-hidden rounded-[10px]"
            style={{
              left: n.x,
              top: n.y,
              width: NW,
              border: `1px solid ${border}`,
              boxShadow: n.focus ? `0 0 0 3px color-mix(in oklch, ${color} 20%, transparent)` : 'none',
            }}
          >
            <div
              className="border-line flex items-center justify-between border-b px-2.5 py-1.5"
              style={{ background: `color-mix(in oklch, ${color} 12%, transparent)`, color }}
            >
              <span className="inline-flex items-center gap-1.5">
                {I[n.iconKey]()}
                <span className="mono tracked text-[10px]">{n.type}</span>
              </span>
              <span className="mono text-[10px]" style={{ color, opacity: 0.7 }}>
                {n.id}
              </span>
            </div>
            <div className="px-3 py-2.5">
              <div className="text-text text-[13.5px] font-medium">{n.title}</div>
              <div className="mono text-text-2 mt-[3px] text-[11px]">{n.sub}</div>
              <div className="mt-2.5 flex items-center gap-1.5">
                <span className="h-[5px] w-[5px] rounded-full" style={{ background: 'var(--zz-lime)' }} />
                <span className="mono text-text-3 text-[10.5px]">{n.badge}</span>
              </div>
            </div>
            <span
              className="bg-bg border-line-2 absolute rounded-full border"
              style={{ left: -4, top: NH / 2 - 4, width: 8, height: 8 }}
            />
            <span className="absolute rounded-full" style={{ right: -4, top: NH / 2 - 4, width: 8, height: 8, background: color }} />
          </div>
        );
      })}
    </div>
  );
}
