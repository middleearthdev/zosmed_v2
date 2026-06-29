import { Card } from './Card';
import { Sparkline } from './Sparkline';

export interface StatCardProps {
  label: string;
  value: string;
  delta: string;
  accent?: string;
  spark: number[];
}

/** Dashboard/analytics metric tile: label · delta · big value · sparkline. */
export function StatCard({ label, value, delta, accent = 'var(--zz-lime)', spark }: StatCardProps) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <span className="mono tracked text-text-3" style={{ fontSize: 9.5 }}>
          {label}
        </span>
        <span className="mono" style={{ fontSize: 11, color: accent }}>
          ↑ {delta}
        </span>
      </div>
      <div
        className="mono tnum"
        style={{ fontSize: 32, fontWeight: 500, marginTop: 12, letterSpacing: '-0.02em' }}
      >
        {value}
      </div>
      <Sparkline data={spark} color={accent} />
    </Card>
  );
}
