import type { ReactNode } from 'react';
import { I } from '@zosmed/ui';

export interface NavItemDef {
  key: string;
  label: string;
  href: string;
  icon: ReactNode;
  badge?: string;
}

/** Sidebar groups — order follows the design artboard (dashboard-dark.jsx). */
export const WORKSPACE_NAV: NavItemDef[] = [
  { key: 'dashboard', label: 'Dashboard', href: '/dashboard', icon: <I.chart /> },
  { key: 'inbox', label: 'Inbox', href: '/inbox', icon: <I.inbox />, badge: '12' },
  { key: 'workflows', label: 'Workflows', href: '/workflows', icon: <I.workflow />, badge: '3' },
  { key: 'ai', label: 'AI Studio', href: '/ai', icon: <I.ai /> },
  { key: 'contacts', label: 'Contacts', href: '/contacts', icon: <I.user /> },
  { key: 'analytics', label: 'Analytics', href: '/analytics', icon: <I.chart /> },
];

export const SYSTEM_NAV: NavItemDef[] = [
  { key: 'templates', label: 'Templates', href: '/templates', icon: <I.sparkle /> },
  { key: 'safety', label: 'Safety', href: '/safety', icon: <I.shield /> },
  { key: 'notifications', label: 'Notifications', href: '/notifications', icon: <I.bell />, badge: '4' },
  { key: 'team', label: 'Team', href: '/team', icon: <I.users /> },
  { key: 'settings', label: 'Settings', href: '/settings', icon: <I.cog /> },
];
