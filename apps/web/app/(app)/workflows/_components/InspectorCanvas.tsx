import { NODE_COLORS, type FlowLink, type InspectorNode } from '@/lib/mock/workflows';

const NW = 200;
const NH = 64;

/** Simplified node canvas for the editor-inspector screen (no badges). */
export function InspectorCanvas({ nodes, links }: { nodes: InspectorNode[]; links: FlowLink[] }) {
  const byId = (id: string) => nodes.find((n) => n.id === id);
  return (
    <div className="absolute inset-0">
      <svg className="absolute inset-0 h-full w-full">
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
              fill="none"
            />
          );
        })}
      </svg>
      {nodes.map((n) => {
        const color = NODE_COLORS[n.type];
        return (
          <div
            key={n.id}
            className="bg-bg-2 absolute rounded-lg p-2.5"
            style={{
              left: n.x,
              top: n.y,
              width: NW,
              border: `${n.selected ? 2 : 1}px solid ${n.selected ? color : 'var(--zz-line)'}`,
              boxShadow: n.selected ? `0 0 0 4px color-mix(in oklch, ${color} 16%, transparent)` : 'none',
            }}
          >
            <div className="mono tracked mb-1 text-[9px]" style={{ color }}>
              {n.type}
            </div>
            <div className="text-[12.5px] font-medium">{n.title}</div>
            <div className="mono text-text-3 mt-0.5 text-[10px]">{n.sub}</div>
          </div>
        );
      })}
    </div>
  );
}
