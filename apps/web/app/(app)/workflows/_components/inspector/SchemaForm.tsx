'use client';

/**
 * Generic, config-schema-driven inspector form (ADR-005 §F4). Renders one
 * control per `FieldSchema.type` from a node's `configSchema` (`@zosmed/types`).
 * Adding a new runnable node = zero new inspector code (§12a DRY): the node's
 * schema drives everything.
 *
 * UX principle (per user): keep the form friendly and hide backend formats.
 * `time` shows a clock picker but stores minutes-since-midnight; `weekdays`
 * shows day toggles but stores weekday numbers; `phone` normalises to E.164;
 * `list` is a chip input. The conversion happens in the control, never in the
 * user's head. Value flows back via `onChange` (never fetched mid-JSX, §12a-3).
 * Callers remount per node (`key={node.id}`) so seeded inputs pick up config.
 */
import { useState, type ReactNode } from 'react';
import type { FieldSchema } from '@zosmed/types';

const inputCls =
  'bg-bg-2 border-line text-text placeholder:text-text-3 w-full rounded-lg border p-3 text-[13px] leading-normal outline-none focus:border-lime';

// Indonesian week order for the toggle; value stored is time.Weekday number.
const WEEKDAYS: { num: string; label: string }[] = [
  { num: '1', label: 'Sen' },
  { num: '2', label: 'Sel' },
  { num: '3', label: 'Rab' },
  { num: '4', label: 'Kam' },
  { num: '5', label: 'Jum' },
  { num: '6', label: 'Sab' },
  { num: '0', label: 'Min' },
];

export function SchemaForm({
  schema,
  config,
  onChange,
}: {
  schema: readonly FieldSchema[];
  config: Record<string, unknown>;
  onChange: (patch: Record<string, unknown>) => void;
}) {
  return (
    <>
      {schema.map((field) => (
        <Field key={field.key} label={field.label} required={field.required} help={field.help}>
          <FieldControl field={field} value={config[field.key]} onChange={(v) => onChange({ [field.key]: v })} />
        </Field>
      ))}
    </>
  );
}

function FieldControl({
  field,
  value,
  onChange,
}: {
  field: FieldSchema;
  value: unknown;
  onChange: (v: unknown) => void;
}) {
  switch (field.type) {
    case 'boolean':
      return (
        <label className="bg-bg-2 border-line flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs">
          <input type="checkbox" defaultChecked={Boolean(value)} onChange={(e) => onChange(e.target.checked)} />
          {field.help ?? 'Aktifkan'}
        </label>
      );

    case 'select':
      return (
        <select
          defaultValue={typeof value === 'string' ? value : (field.options?.[0]?.value ?? '')}
          onChange={(e) => onChange(e.target.value)}
          className={inputCls}
          style={{ colorScheme: 'dark' }}
        >
          {(field.options ?? []).map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      );

    case 'time':
      return (
        <input
          type="time"
          defaultValue={minutesToHHMM(value)}
          onChange={(e) => onChange(e.target.value === '' ? undefined : hhmmToMinutes(e.target.value))}
          className={inputCls}
          style={{ colorScheme: 'dark' }}
        />
      );

    case 'weekdays':
      return <WeekdayToggle value={Array.isArray(value) ? (value as string[]) : []} onChange={onChange} />;

    case 'phone':
      return (
        <input
          type="tel"
          inputMode="numeric"
          defaultValue={typeof value === 'string' ? value : ''}
          onBlur={(e) => {
            const norm = normalizePhone(e.target.value);
            e.target.value = norm; // reflect the cleaned number back to the user
            onChange(norm);
          }}
          placeholder={field.placeholder}
          className={inputCls}
        />
      );

    case 'list':
      return <ChipInput value={Array.isArray(value) ? (value as string[]) : []} onChange={onChange} placeholder={field.placeholder} />;

    case 'textarea':
      return (
        <textarea
          rows={4}
          defaultValue={typeof value === 'string' ? value : ''}
          onBlur={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className={inputCls}
        />
      );

    case 'text':
    default:
      return (
        <input
          type="text"
          defaultValue={typeof value === 'string' ? value : ''}
          onBlur={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className={inputCls}
        />
      );
  }
}

// ── friendly controls ───────────────────────────────────────────────────────

function ChipInput({
  value,
  onChange,
  placeholder,
}: {
  value: string[];
  onChange: (v: string[]) => void;
  placeholder?: string | undefined;
}) {
  const [chips, setChips] = useState<string[]>(value);
  const [draft, setDraft] = useState('');

  function commit(next: string[]) {
    setChips(next);
    onChange(next);
  }
  function add(raw: string) {
    const t = raw.trim();
    if (!t || chips.includes(t)) return;
    commit([...chips, t]);
  }

  return (
    <div className="bg-bg-2 border-line flex flex-wrap gap-1.5 rounded-lg border p-2 focus-within:border-lime">
      {chips.map((c) => (
        <span key={c} className="bg-bg-3 text-text flex items-center gap-1 rounded-md px-2 py-1 text-[12px]">
          {c}
          <button type="button" onClick={() => commit(chips.filter((x) => x !== c))} className="text-text-3 hover:text-text leading-none">
            ×
          </button>
        </span>
      ))}
      <input
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault();
            add(draft);
            setDraft('');
          } else if (e.key === 'Backspace' && draft === '' && chips.length) {
            commit(chips.slice(0, -1));
          }
        }}
        onBlur={() => {
          if (draft.trim()) {
            add(draft);
            setDraft('');
          }
        }}
        placeholder={chips.length === 0 ? placeholder : ''}
        className="text-text placeholder:text-text-3 min-w-[8ch] flex-1 bg-transparent p-1 text-[13px] outline-none"
      />
    </div>
  );
}

function WeekdayToggle({ value, onChange }: { value: string[]; onChange: (v: string[]) => void }) {
  const [days, setDays] = useState<string[]>(value);
  function toggle(num: string) {
    const next = days.includes(num) ? days.filter((d) => d !== num) : [...days, num];
    setDays(next);
    onChange(next);
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {WEEKDAYS.map((d) => {
        const on = days.includes(d.num);
        return (
          <button
            key={d.num}
            type="button"
            onClick={() => toggle(d.num)}
            className="mono rounded-md border px-2.5 py-1.5 text-[11px] transition-colors"
            style={
              on
                ? { background: 'color-mix(in oklch, var(--zz-lime) 16%, transparent)', borderColor: 'var(--zz-lime)', color: 'var(--zz-lime)' }
                : { background: 'var(--zz-bg-2)', borderColor: 'var(--zz-line)', color: 'var(--zz-text-2)' }
            }
          >
            {d.label}
          </button>
        );
      })}
    </div>
  );
}

// ── value transforms (backend format ⇄ friendly control) ────────────────────

function minutesToHHMM(value: unknown): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '';
  const h = Math.floor(value / 60) % 24;
  const m = value % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function hhmmToMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number);
  return (h || 0) * 60 + (m || 0);
}

/** Normalise an Indonesian phone number to E.164 digits (no '+'): 0812… → 62812…. */
function normalizePhone(raw: string): string {
  const d = raw.replace(/\D/g, '');
  if (!d) return '';
  if (d.startsWith('62')) return d;
  if (d.startsWith('0')) return `62${d.slice(1)}`;
  if (d.startsWith('8')) return `62${d}`;
  return d;
}

function Field({
  label,
  required,
  help,
  children,
}: {
  label: string;
  required?: boolean | undefined;
  help?: string | undefined;
  children: ReactNode;
}) {
  return (
    <div className="mb-[18px]">
      <div className="mono tracked text-text-3 mb-2 text-[9.5px]">
        {label.toUpperCase()}
        {required ? <span style={{ color: 'var(--zz-pink)' }}> *</span> : null}
      </div>
      {children}
      {help ? <span className="text-text-3 mt-1.5 block text-[10.5px] leading-normal">{help}</span> : null}
    </div>
  );
}

/** Seed config for a freshly-added node from its schema `default`s (ADR-005 §F4, DRY). */
export function defaultConfigFromSchema(schema: readonly FieldSchema[] | undefined): Record<string, unknown> {
  const cfg: Record<string, unknown> = {};
  for (const f of schema ?? []) {
    if (f.default !== undefined) cfg[f.key] = f.default;
    else if (f.type === 'list' || f.type === 'weekdays') cfg[f.key] = [];
    else if (f.type === 'boolean') cfg[f.key] = false;
    else if (f.type === 'select') cfg[f.key] = f.options?.[0]?.value ?? '';
  }
  return cfg;
}
