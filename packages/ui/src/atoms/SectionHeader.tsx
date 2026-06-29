import type { ReactNode } from 'react';
import { cn } from '../lib/cn';

export interface SectionHeaderProps {
  title: string;
  subtitle?: ReactNode;
  action?: ReactNode;
  size?: 'md' | 'sm';
  className?: string;
}

/** Title + optional subtitle (left) and action (right). Atop most panels. */
export function SectionHeader({ title, subtitle, action, size = 'md', className }: SectionHeaderProps) {
  return (
    <div className={cn('mb-4 flex items-center justify-between', className)}>
      <div>
        <h3 className={cn('m-0 font-medium', size === 'md' ? 'text-base' : 'text-sm')}>{title}</h3>
        {subtitle ? <p className="text-text-3 m-0 mt-1 text-xs">{subtitle}</p> : null}
      </div>
      {action}
    </div>
  );
}
