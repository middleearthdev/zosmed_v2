import type { SVGProps, ReactElement } from 'react';

export type IconProps = SVGProps<SVGSVGElement>;

const base: IconProps = {
  width: 14,
  height: 14,
  viewBox: '0 0 24 24',
  fill: 'none',
};

/**
 * Shared icon set (ported from the design `primitives.jsx` `I` record).
 * Strokes use `currentColor` so callers set color via text utilities.
 */
export const I = {
  arrow: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M5 12h14M13 6l6 6-6 6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  ),
  bolt: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M13 2L4 14h7l-1 8 9-12h-7l1-8z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  ),
  chat: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M21 12a8 8 0 0 1-12 6.9L3 21l2.1-6A8 8 0 1 1 21 12z" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  ),
  filter: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M3 5h18M6 12h12M10 19h4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  ),
  send: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  ),
  ai: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  ),
  chart: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M3 21V3M3 21h18M7 17v-6M12 17V8M17 17v-3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  ),
  workflow: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <rect x="3" y="3" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.6" />
      <rect x="15" y="15" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.6" />
      <path d="M9 6h4a2 2 0 0 1 2 2v7" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  ),
  inbox: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M22 12h-6l-2 3h-4l-2-3H2M5 5h14l3 7v7a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2v-7l3-7z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    </svg>
  ),
  user: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <circle cx="12" cy="8" r="4" stroke="currentColor" strokeWidth="1.6" />
      <path d="M4 21c1.5-4 5-6 8-6s6.5 2 8 6" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  ),
  cog: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.6" />
      <path
        d="M12 1v3M12 20v3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M1 12h3M20 12h3M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  ),
  plus: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  ),
  search: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <circle cx="11" cy="11" r="7" stroke="currentColor" strokeWidth="1.6" />
      <path d="M21 21l-4.3-4.3" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  ),
  heart: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.6l-1-1a5.5 5.5 0 0 0-7.8 7.8l1 1L12 21l7.8-7.8 1-1a5.5 5.5 0 0 0 0-7.6z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  ),
  check: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M5 12l5 5L20 6" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  sparkle: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8L12 3zM19 16l.9 2.1L22 19l-2.1.9L19 22l-.9-2.1L16 19l2.1-.9L19 16z"
        fill="currentColor"
      />
    </svg>
  ),
  shield: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M12 2l9 4v6c0 5-3.5 9-9 10-5.5-1-9-5-9-10V6l9-4z" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  ),
  bell: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path
        d="M18 8a6 6 0 1 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 0 1-3.4 0"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  ),
  users: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <circle cx="9" cy="8" r="3.2" stroke="currentColor" strokeWidth="1.6" />
      <path d="M3 20c1.2-3.2 3.6-4.6 6-4.6s4.8 1.4 6 4.6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      <path
        d="M16 5.2a3.2 3.2 0 0 1 0 6M18 15.4c1.8.5 3.2 1.9 4 4.6"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
    </svg>
  ),
  whatsapp: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M3 21l1.7-5.1A8 8 0 1 1 8.5 19.4L3 21z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
      <path
        d="M9 8.5c0 3.5 3 6.5 6.5 6.5l.6-1.6-2-1-1 .9c-1.2-.5-2.3-1.6-2.8-2.8l.9-1-1-2L9 8.5z"
        fill="currentColor"
      />
    </svg>
  ),
  live: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <circle cx="12" cy="12" r="3" fill="currentColor" />
      <path
        d="M6.5 6.5a8 8 0 0 0 0 11M17.5 6.5a8 8 0 0 1 0 11M4 4a12 12 0 0 0 0 16M20 4a12 12 0 0 1 0 16"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
    </svg>
  ),
  calendar: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <rect x="3" y="5" width="18" height="16" rx="2" stroke="currentColor" strokeWidth="1.6" />
      <path d="M3 9h18M8 3v4M16 3v4" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  ),
  tag: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M3 12V4a1 1 0 0 1 1-1h8l9 9-9 9-9-9z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
      <circle cx="7.5" cy="7.5" r="1.4" fill="currentColor" />
    </svg>
  ),
  box: (p: IconProps = {}) => (
    <svg {...base} {...p}>
      <path d="M21 8l-9-5-9 5 9 5 9-5zM3 8v8l9 5 9-5V8M12 13v8" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
    </svg>
  ),
} satisfies Record<string, (p?: IconProps) => ReactElement>;

export type IconName = keyof typeof I;
