/** Tiny className joiner (no dep). Falsy parts are dropped. */
export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ');
}
