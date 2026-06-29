'use client';

import { createContext, useContext, type ReactNode } from 'react';
import type { Segment } from '@zosmed/types';

const KitContext = createContext<Segment>('seller');

/** Provides the active segment/Kit (CLAUDE.md §8) to client components. */
export function KitProvider({ segment, children }: { segment: Segment; children: ReactNode }) {
  return <KitContext.Provider value={segment}>{children}</KitContext.Provider>;
}

export function useKit(): Segment {
  return useContext(KitContext);
}
