/**
 * Kit presets — neutral engine + per-segment specialist config (CLAUDE.md §8).
 * Full Seller/Creator/Booking preset content is built in Phase 6; this file
 * defines the shared descriptor shape so screens can type against it now.
 */
import type { Segment } from '@zosmed/types';

export interface KitDescriptor {
  segment: Segment;
  /** Display name, Bahasa Indonesia. */
  name: string;
  tagline: string;
  /** Default trigger keywords surfaced in onboarding/templates. */
  keywords: readonly string[];
}
