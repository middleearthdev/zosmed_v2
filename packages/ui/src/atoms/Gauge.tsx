import { Meter } from './Meter';

export interface GaugeProps {
  label: string;
  valueText: string;
  capText?: string;
  /** 0..1 fill ratio. */
  value: number;
  color?: string;
}

/** Labelled quota row (label · value/cap · bar) — Safety center, sidebar usage. */
export function Gauge({ label, valueText, capText, value, color }: GaugeProps) {
  return (
    <div className="mb-3">
      <div className="mb-1 flex justify-between text-xs">
        <span className="text-text-2">{label}</span>
        <span className="mono">
          <span className="text-text">{valueText}</span>
          {capText ? <span className="text-text-3"> / {capText}</span> : null}
        </span>
      </div>
      <Meter value={value} color={color ?? 'var(--zz-lime)'} />
    </div>
  );
}
