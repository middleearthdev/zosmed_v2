import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { cn } from '../lib/cn';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'lime' | 'ghost';
  icon?: ReactNode;
}

/** Primary actions. `.btn-lime` / `.btn-ghost` live in the shared globals. */
export function Button({ variant = 'lime', icon, children, className, type = 'button', ...rest }: ButtonProps) {
  return (
    <button type={type} className={cn(variant === 'lime' ? 'btn-lime' : 'btn-ghost', className)} {...rest}>
      {icon}
      {children}
    </button>
  );
}
