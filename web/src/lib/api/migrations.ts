import { api } from './client';
import type { PanelKind } from './panels';

export type JobState =
  'created' | 'checked' | 'running' | 'done' | 'failed' | 'rolled_back' | 'interrupted';

/** What a run did, from its report. */
export interface Counts {
  read: number;
  moved: number;
  skipped: number;
  adopted: number;
  undone: number;
  configChanged: number;
  renamed: number;
  protocolsAdded: number;
  verified: number;
}

/** One job as the jobs list shows it. */
export interface JobRow {
  id: string;
  state: JobState;
  /** The destination panel; kind is empty when Karvan does not know it. */
  panel: { id: number; name: string; kind: PanelKind | '' };
  /** The old panel: its kind, its address and its admin (empty before the first compare). */
  source: { kind: PanelKind; origin: string; admin: string };
  safeMode: boolean;
  /** From the last run; null until a run ends. */
  counts: Counts | null;
  createdAt: string;
  updatedAt: string;
  endedAt: string | null;
  /** Karvan still holds the old panel's login. */
  loginHeld: boolean;
}

export interface JobPage {
  jobs: JobRow[];
  /** The cursor of the next page, when there is one. */
  next?: string;
}

export interface JobFilter {
  limit?: number;
  panel?: number;
  states?: readonly JobState[];
  cursor?: string;
}

/** Jobs newest first, a page at a time. */
export function listJobs(f: JobFilter = {}): Promise<JobPage> {
  const q = new URLSearchParams();
  if (f.limit) q.set('limit', String(f.limit));
  if (f.panel) q.set('panel', String(f.panel));
  if (f.states?.length) q.set('state', f.states.join(','));
  if (f.cursor) q.set('cursor', f.cursor);
  const query = q.toString();
  return api<JobPage>('GET', query ? `migrations?${query}` : 'migrations');
}

/**
 * The users a job moved that are still there and the users it left out,
 * counted the way the overview's totals count them. Null before a run ends,
 * and after a rollback took the users back.
 */
export function jobOutcome(
  job: Pick<JobRow, 'state' | 'counts'>,
): { moved: number; skipped: number } | null {
  if (!job.counts || job.state === 'rolled_back') return null;
  return { moved: Math.max(0, job.counts.moved - job.counts.undone), skipped: job.counts.skipped };
}

/** The old panel's host and port, such as "old.example.com:8443". */
export function originHost(origin: string): string {
  try {
    return new URL(origin).host || origin;
  } catch {
    return origin;
  }
}
