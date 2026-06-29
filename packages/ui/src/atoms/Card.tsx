import type { ReactNode } from 'react';
import { cn } from '../lib/cn';

export interface CardProps {
  children: ReactNode;
  className?: string;
}

/** Dominant container surface: bg-2 panel, hairline border, rounded. */
export function Card({ children, className }: CardProps) {
  return <div className={cn('bg-bg-2 border-line rounded-xl border p-5', className)}>{children}</div>;
}
