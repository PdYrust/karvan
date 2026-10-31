import { api } from './client';
import type { JobState } from './migrations';

export type Outcome = 'success' | 'failure';

export interface ActivityEntry {
  id: number;
  at: string;
  actor: string;
  action: string;
  outcome: Outcome;
  ip: string;
}

/** Every migration summed up. */
export interface MigrationTotals {
  /** Jobs in each state; a state no job is in is left out. */
  jobs: Partial<Record<JobState, number>>;
  /** Users moved and still on their new panel. A rolled back job adds none. */
  moved: number;
  /** Users the runs left out. */
  skipped: number;
  lastEnded: string | null;
}

export interface Overview {
  version: string;
  safeMode: boolean;
  security: {
    totpEnabled: boolean;
    previousSignIn: { at: string; ip: string } | null;
    /** How the panel is served. */
    tls: 'off' | 'self' | 'domain';
  };
  migrations: MigrationTotals;
  activity: ActivityEntry[];
}

export function getOverview(): Promise<Overview> {
  return api<Overview>('GET', 'overview');
}
