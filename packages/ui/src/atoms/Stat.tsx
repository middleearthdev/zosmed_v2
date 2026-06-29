export interface StatProps {
  value: string;
  label: string;
  highlight?: boolean;
}

/** Compact right-aligned metric (value over tracked label). */
export function Stat({ value, label, highlight }: StatProps) {
  return (
    <div className="text-right">
      <div className="mono tnum text-sm" style={{ color: highlight ? 'var(--zz-lime)' : 'var(--zz-text)' }}>
        {value}
      </div>
      <div className="mono tracked text-text-3" style={{ fontSize: 9 }}>
        {label}
      </div>
    </div>
  );
}
