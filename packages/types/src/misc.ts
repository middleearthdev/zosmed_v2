/** Supporting contracts: notifications, team, templates, analytics. */
import type { Id, ISODateTime, Segment } from './common';

export interface Notification {
  id: Id;
  group: 'leads' | 'workflows' | 'safety' | 'ai' | 'team';
  title: string;
  body: string;
  read: boolean;
  at: ISODateTime;
}

export type TeamRole = 'owner' | 'admin' | 'agent' | 'viewer';

export interface TeamMember {
  id: Id;
  name: string;
  email: string;
  role: TeamRole;
  avatarColor?: string;
  accountAccess: Id[];
  lastActive?: ISODateTime;
}

export interface Template {
  id: Id;
  name: string;
  description: string;
  segment: Segment;
  category: string;
  nodeCount: number;
}

export interface AnalyticsMetric {
  key: string;
  label: string;
  value: number;
  unit?: string;
  deltaPct?: number;
}

export interface FunnelStep {
  label: string;
  value: number;
}
