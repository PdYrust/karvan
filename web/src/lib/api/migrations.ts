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

/** Query keys of everything about migrations, so one invalidation reaches them all (D2). */
export const migrationKeys = {
  all: ['migrations'] as const,
  job: (id: string) => ['migrations', 'job', id] as const,
};

const jobPath = (id: string, rest = '') => `migrations/${encodeURIComponent(id)}${rest}`;

/** What a new job needs: the destination and the old panel's address and sign-in. */
export interface NewJob {
  panelId: number;
  sourceUrl: string;
  /** Left out, Karvan signs in and finds out by itself. */
  sourceKind?: PanelKind;
  username?: string;
  password?: string;
  /** A PasarGuard API key instead of a username and password. */
  apiKey?: string;
  /** The SHA-256 of the one certificate the operator chose to trust. */
  pinSha256?: string;
}

export function createJob(job: NewJob): Promise<{ id: string; state: JobState }> {
  return api('POST', 'migrations', job);
}

/** A target moved users can get: a PasarGuard group, or a Marzban inbound as "protocol/tag". */
export interface TargetInfo {
  id: string;
  name: string;
  protocols: string[] | null;
  /** Enabled hosts: one link per host per user. */
  hosts: number;
  disabled?: boolean;
}

export type UserStatus = 'active' | 'disabled' | 'limited' | 'expired' | 'on_hold';

/** The old panel as read and the destination, after a compare (api-migration.md 2.3). */
export interface JobSummary {
  state: JobState;
  planVersion: number;
  sourceKind: PanelKind | '';
  origin: string;
  /** Empty once Karvan dropped the old panel's login. */
  admin: string;
  users: number;
  usedTraffic: number;
  statuses: Partial<Record<UserStatus, number>> | null;
  destKind: PanelKind | '';
  targets: TargetInfo[] | null;
  ready: number;
  problems: number;
  skipped: number;
  movedBefore: number;
  blockers: number;
  stable: boolean;
  roleId?: number;
  userTargets: string[] | null;
}

export interface Job {
  id: string;
  state: JobState;
  /** Present once a compare has read the old panel. */
  summary?: JobSummary;
}

export function getJob(id: string): Promise<Job> {
  return api('GET', jobPath(id));
}

/** Reads both panels again in the background; the job's events say how far it got. */
export function compareJob(id: string): Promise<{ id: string }> {
  return api('POST', jobPath(id, '/compare'));
}

/** Stops a compare that reads, or a run between batches; does nothing otherwise. */
export function cancelJob(id: string): Promise<void> {
  return api('POST', jobPath(id, '/cancel'));
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
