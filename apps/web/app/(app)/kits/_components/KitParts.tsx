import type { ReactNode } from 'react';

export function KitIntro({ kit, accent, title, sub }: { kit: string; accent: string; title: string; sub: string }) {
  return (
    <div className="mb-6">
      <span className="mono tracked text-[10px]" style={{ color: accent }}>{`// ${kit.toLowerCase()}`}</span>
      <h1 className="m-0 mt-1.5 text-3xl font-medium tracking-tight">{title}</h1>
      <p className="text-text-2 m-0 mt-1.5 max-w-[640px] text-sm">{sub}</p>
    </div>
  );
}

export function KitCard({
  icon,
  accent,
  title,
  node,
  on = true,
  children,
}: {
  icon: ReactNode;
  accent: string;
  title: string;
  node?: string;
  on?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="bg-bg-2 border-line rounded-xl border p-5">
      <div className="mb-3.5 flex items-center gap-2.5">
        <span
          className="inline-flex h-8 w-8 items-center justify-center rounded-lg"
          style={{ background: `color-mix(in oklch, ${accent} 15%, transparent)`, color: accent }}
        >
          {icon}
        </span>
        <div>
          <h3 className="m-0 text-[15px] font-medium">{title}</h3>
          {node ? <span className="mono text-text-3 text-[10.5px]">{node}</span> : null}
        </div>
        <span className="relative ml-auto rounded-full" style={{ width: 32, height: 18, background: on ? accent : '#2a2a2e' }}>
          <span className="bg-bg absolute rounded-full" style={{ top: 2, left: on ? 16 : 2, width: 14, height: 14 }} />
        </span>
      </div>
      {children}
    </div>
  );
}
