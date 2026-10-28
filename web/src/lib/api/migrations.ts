import type { Plan } from '../plan/plan';
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

/** The batch a job became when it ended done (phase 4.1 D1). */
export interface BatchRef {
  id: number;
  name: string;
  state: 'active' | 'inactive' | 'deleted';
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
  /** The batch the job became; null when it has none. */
  batch: BatchRef | null;
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
  /** One job's row. */
  id?: string;
}

/** Jobs newest first, a page at a time. */
export function listJobs(f: JobFilter = {}): Promise<JobPage> {
  const q = new URLSearchParams();
  if (f.limit) q.set('limit', String(f.limit));
  if (f.panel) q.set('panel', String(f.panel));
  if (f.states?.length) q.set('state', f.states.join(','));
  if (f.id) q.set('id', f.id);
  if (f.cursor) q.set('cursor', f.cursor);
  const query = q.toString();
  return api<JobPage>('GET', query ? `migrations?${query}` : 'migrations');
}

/**
 * One job as the list shows it, read from the database: it answers while
 * the job runs and after a restart, when the job's own reads cannot. Null
 * for an id Karvan does not know.
 */
export async function getJobRow(id: string): Promise<JobRow | null> {
  return (await listJobs({ id })).jobs[0] ?? null;
}

/** Query keys of everything about migrations, so one invalidation reaches them all (D2). */
export const migrationKeys = {
  all: ['migrations'] as const,
  /** The job and, under it, everything read from its last build. */
  job: (id: string) => ['migrations', 'job', id] as const,
  summary: (id: string) => ['migrations', 'job', id, 'summary'] as const,
  groups: (id: string) => ['migrations', 'job', id, 'groups'] as const,
  users: (id: string, q: UserQuery) => ['migrations', 'job', id, 'users', q] as const,
  plan: (id: string) => ['migrations', 'job', id, 'plan'] as const,
  roles: (id: string) => ['migrations', 'job', id, 'roles'] as const,
  row: (id: string) => ['migrations', 'job', id, 'row'] as const,
  report: (id: string) => ['migrations', 'job', id, 'report'] as const,
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

/** The old panel's users by when they expire. */
export interface Expiries {
  expired: number;
  /** Within 7 days. */
  week: number;
  /** Within 30 days, after the first 7. */
  month: number;
  later: number;
  /** On hold without an end date: the time starts at the first connection. */
  onHold: number;
  never: number;
}

/** The admin a plan makes on the destination: its name, and whether it merges into one there. */
export interface AdminSummary {
  username: string;
  merged: boolean;
  roleId: number;
}

/** One admin of the old panel and how it lands (D34); the login's admin comes first. */
export interface AdminInfo {
  /** Its name on the old panel, the key of plan.admins. */
  username: string;
  isLogin: boolean;
  /** Its users the run moves. */
  users: number;
  /** False for an admin other than the login's with no user to move: the run leaves it out. */
  moves: boolean;
  roleId: number;
  /** A merge keeps the existing admin's own role (D33). */
  roleFixed: boolean;
  newAdmin: AdminSummary;
  userTargets: string[] | null;
}

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
  /** The destination panel. */
  panelId: number;
  expiries: Expiries;
  /** Users per protocol they have on the old panel. */
  protocols: Record<string, number>;
  newAdmin: AdminSummary;
  /** A run without a typed password keeps the reseller's own one; otherwise Karvan makes one. */
  canKeepPassword: boolean;
  /** Every admin of the old panel's users; null in the zero summary of a running job. */
  admins: AdminInfo[] | null;
}

export interface Job {
  id: string;
  state: JobState;
  /** Present once a compare has read the old panel. */
  summary?: JobSummary;
  /** The batch the job became; null when it has none. */
  batch: BatchRef | null;
}

export function getJob(id: string): Promise<Job> {
  return api('GET', jobPath(id));
}

export function getSummary(id: string): Promise<JobSummary> {
  return api('GET', jobPath(id, '/summary'));
}

export type Severity = 'blocker' | 'warning' | 'info';

export interface Conflict {
  type: string;
  severity: Severity;
  detail?: Record<string, string>;
  fix?: string;
}

/** A fix Karvan offers for a conflict type, best first. */
export interface FixOption {
  id: string;
  params?: string[];
  changesConfig: boolean;
  changesLink: boolean;
}

/** Users with one conflict type, fixed or not: how many, the fixes, and how many still need one. */
export interface ProblemGroup {
  type: string;
  count: number;
  options: FixOption[];
  open: number;
}

export interface Groups {
  planVersion: number;
  groups: ProblemGroup[];
  /** Conflicts of the whole job, such as a taken admin name. */
  conflicts: Conflict[];
  /** Users per category id; categories without users are left out. */
  categories: Record<string, number>;
  blockers: number;
  stable: boolean;
}

export function getGroups(id: string): Promise<Groups> {
  return api('GET', jobPath(id, '/groups'));
}

/** One user as read (or, in the preview, as it will land) with the plan's outcome. */
export interface UserRow {
  key: string;
  sourceUsername: string;
  username: string;
  include: boolean;
  status: UserStatus;
  usedTraffic: number;
  dataLimit: number;
  expire: string | null;
  note?: string;
  categories: string[];
  targets?: string[];
  conflicts: { type: string; severity: Severity; fix?: string }[];
  /** What the plan does to the user, such as "fix:C02:suffix" or "op:traffic.add". */
  applied: string[];
  changesConfig: boolean;
  changesLink: boolean;
  movedBefore?: boolean;
}

export interface UserPage {
  users: UserRow[];
  total: number;
  next?: string;
}

export interface UserQuery {
  /** "" for everyone, "ready", "skipped", "category:<id>" or a conflict type. */
  group?: string;
  q?: string;
  offset?: number;
  limit?: number;
}

/** Users per page in the wizard's lists (D5). */
export const usersPerPage = 50;

/** Users a page at a time (D5), searched by name on the server. */
export function getUsers(id: string, query: UserQuery = {}): Promise<UserPage> {
  const q = new URLSearchParams();
  if (query.group) q.set('group', query.group);
  if (query.q) q.set('q', query.q);
  if (query.offset) q.set('cursor', String(query.offset));
  q.set('limit', String(query.limit ?? usersPerPage));
  return api('GET', jobPath(id, `/users?${q.toString()}`));
}

/** What a plan does; every count but users is left out when it is 0. */
export interface Impact {
  users: number;
  moving?: number;
  skipped?: number;
  configChanged?: number;
  renamed?: number;
  changed?: number;
}

export interface PlanState {
  planVersion: number;
  plan: Plan;
  /** Present once a compare has read the old panel. */
  impact?: Impact;
}

export function getPlan(id: string): Promise<PlanState> {
  return api('GET', jobPath(id, '/plan'));
}

export interface PlanSaved {
  planVersion: number;
  blockers: number;
  stable: boolean;
  groups: ProblemGroup[];
  conflicts: Conflict[];
  impact: Impact;
}

/** Saves the whole plan if version is still the current one; a 412 says it is not (D7). */
export function putPlan(id: string, version: number, plan: Plan): Promise<PlanSaved> {
  return api('PUT', jobPath(id, '/plan'), plan, { ifMatch: version });
}

/** A role's limits: bytes, seconds and counts, null for no limit. */
export interface RoleLimits {
  max_users: number | null;
  data_limit_min: number | null;
  data_limit_max: number | null;
  expire_min: number | null;
  expire_max: number | null;
  on_hold_timeout_min?: number | null;
  on_hold_timeout_max?: number | null;
  min_hwid_per_user: number | null;
  max_hwid_per_user: number | null;
}

/** A PasarGuard admin role the moved admin can get. */
export interface Role {
  id: number;
  name: string;
  isOwner: boolean;
  /** The role may use every group. */
  allGroups: boolean;
  groups: TargetInfo[];
  limits: RoleLimits;
  features: { can_use_reset_strategy: boolean; can_use_next_plan: boolean };
  requireTemplate: boolean;
}

export function getRoles(
  id: string,
): Promise<{ roleId: number; roleFixed: boolean; admins: AdminInfo[] | null; roles: Role[] }> {
  return api('GET', jobPath(id, '/roles'));
}

/** Reads both panels again in the background; the job's events say how far it got. */
export function compareJob(id: string): Promise<{ id: string }> {
  return api('POST', jobPath(id, '/compare'));
}

/** Stops a compare that reads, or a run between chunks; does nothing otherwise. */
export function cancelJob(id: string): Promise<void> {
  return api('POST', jobPath(id, '/cancel'));
}

/** Deletes a draft that never ran; the server refuses any other job with 409 not_deletable. */
export function deleteJob(id: string): Promise<void> {
  return api('DELETE', jobPath(id));
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
