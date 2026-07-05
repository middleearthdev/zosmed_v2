import type { InputHTMLAttributes } from 'react';
import { cn } from '../lib/cn';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

/**
 * Text input matching the dark/lime surface (CLAUDE.md §11). Presentational
 * only — validation/state is owned by the calling form (§12a-3).
 */
export function Input({ label, error, id, className, ...rest }: InputProps) {
  const inputId = id ?? rest.name;
  return (
    <div className="flex flex-col gap-1.5">
      {label ? (
        <label htmlFor={inputId} className="text-text-2 text-xs font-medium">
          {label}
        </label>
      ) : null}
      <input
        id={inputId}
        className={cn(
          'bg-bg-3 border-line text-text placeholder:text-text-3 rounded-lg border px-3 py-2.5 text-sm outline-none transition-colors',
          'focus:border-lime',
          error ? 'border-pink' : '',
          className,
        )}
        aria-invalid={error ? true : undefined}
        {...rest}
      />
      {error ? (
        <span className="text-pink text-xs" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
