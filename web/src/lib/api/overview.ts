import { api } from './client';

export type Outcome = 'success' | 'failure';

export interface ActivityEntry {
  at: string;
  actor: string;
  action: string;
  outcome: Outcome;
  ip: string;
}

export interface Overview {
  version: string;
  safeMode: boolean;
  security: {
    totpEnabled: boolean;
    previousSignIn: { at: string; ip: string } | null;
  };
  // Panel discovery arrives in phase 3; the list is honest-empty until then.
  panels: never[];
  activity: ActivityEntry[];
}

export function getOverview(): Promise<Overview> {
  return api<Overview>('GET', 'overview');
}
