import type { ReactNode } from 'react';
import { cn } from '@zosmed/ui';

/** Standard 56px page header bar (title/left + actions/right). */
export function PageHeader({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <header
      className={cn(
        'border-bg-3 flex h-14 flex-shrink-0 items-center justify-between border-b px-6',
        className,
      )}
    >
      {children}
    </header>
  );
}
