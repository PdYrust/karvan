import { api, request } from './client';
import {
  usersPerPage,
  type Conflict,
  type Severity,
  type UserPage,
  type UserQuery,
} from './migrations';

/** The reads of the last two steps: the preview, what changed, the reload choice and the run. */
const jobPath = (id: string, rest: string) => `migrations/${encodeURIComponent(id)}${rest}`;

export const preflightKeys = {
  preview: (id: string, q: UserQuery) => ['migrations', 'job', id, 'preview', q] as const,
  changes: (id: string, offset: number) => ['migrations', 'job', id, 'changes', offset] as const,
  checkReport: (id: string) => ['migrations', 'job', id, 'check-report'] as const,
  reload: (id: string) => ['migrations', 'job', id, 'reload-options'] as const,
};

/** The preview's filters beside everyone (api-migration.md 2.6). */
export const previewFilters = ['', 'renamed', 'configChanged', 'skipped'] as const;
export type PreviewFilter = (typeof previewFilters)[number];

/** Users as they will land, a page at a time (D5), searched by name on the server. */
export function getPreview(id: string, query: UserQuery = {}): Promise<UserPage> {
  const q = new URLSearchParams();
  if (query.group) q.set('group', query.group);
  if (query.q) q.set('q', query.q);
  if (query.offset) q.set('cursor', String(query.offset));
  q.set('limit', String(query.limit ?? usersPerPage));
  return api('GET', jobPath(id, `/preview?${q.toString()}`));
}

/** One user's outcome in a diff: no credential leaves Karvan, only which protocols it has. */
export interface Outcome {
  sourceUsername: string;
  username: string;
  include: boolean;
  adopted: boolean;
  targets: string[] | null;
  protocols: string[] | null;
  conflicts: { type: string; severity: Severity; fix?: string }[] | null;
}

export type ChangeField =
  'include' | 'adopted' | 'username' | 'targets' | 'credentials' | 'conflicts';

export interface UserChange {
  key: string;
  change: 'new' | 'gone' | 'changed';
  fields?: ChangeField[];
  before?: Outcome;
  after?: Outcome;
}

export interface AdminOutcome {
  username: string;
  merged: boolean;
  roleId: number;
}

/** What the last read of the panels changed in the outcome (B3). */
export interface Changes {
  planVersion: number;
  checkedAt: string;
  changed: boolean;
  admin: { fields: string[]; before: AdminOutcome; after: AdminOutcome } | null;
  conflicts: { before: Conflict[] | null; after: Conflict[] | null } | null;
  counts: { new: number; gone: number; changed: number };
  users: UserChange[];
  total: number;
  next?: string;
}

export function getChanges(id: string, offset = 0, limit = usersPerPage): Promise<Changes> {
  const q = new URLSearchParams({ limit: String(limit) });
  if (offset) q.set('cursor', String(offset));
  return api('GET', jobPath(id, `/changes?${q.toString()}`));
}

/** One user in the pre-flight report. */
export interface CheckUser {
  key: string;
  sourceUsername: string;
  username: string;
  moving: boolean;
  adopted: boolean;
  targets: string[];
  applied: string[];
  /** Fixed conflicts as type:fix. */
  fixed: string[];
  conflicts: { type: string; severity: Severity; fix?: string }[];
  changesConfig: boolean;
  changesLink: boolean;
}

/** The pre-flight report, karvan.check/v1 (B7). */
export interface CheckReport {
  schema: string;
  jobId: string;
  planVersion: number;
  checkedAt: string;
  source: { kind: string; origin: string; admin: string };
  destination: { kind: string; name: string };
  admin: AdminOutcome;
  totals: {
    users: number;
    moving: number;
    skipped: number;
    /** Left out because an earlier migration moved them (C16), also in skipped. */
    alreadyMoved: number;
    adopted: number;
    renamed: number;
    configChanged: number;
    changed: number;
  };
  blockers: number;
  stable: boolean;
  conflicts: Conflict[];
  users: CheckUser[];
}

export function getCheckReport(id: string): Promise<CheckReport> {
  return api('GET', jobPath(id, '/check-report'));
}

export type ReloadMethod = 'manual' | 'touch' | 'core_restart';

/** A reload method as the final step shows it (api-migration.md 9.1). */
export interface ReloadOption {
  method: ReloadMethod;
  available: boolean;
  reason?: string;
  effects: string[];
  /** Manual only: the command the operator runs. */
  command?: string;
}

export async function getReloadOptions(id: string): Promise<ReloadOption[]> {
  return (await api<{ options: ReloadOption[] }>('GET', jobPath(id, '/reload-options'))).options;
}

export interface RunRequest {
  planVersion: number;
  reload: ReloadMethod;
  /** Left out, the reseller's own password is kept, or Karvan makes one. */
  adminPassword?: string;
}

/** Starts the run; its progress comes on the job's event stream. */
export function runJob(id: string, body: RunRequest): Promise<{ id: string; state: string }> {
  return api('POST', jobPath(id, '/run'), body);
}

/** The file name a download's Content-Disposition names, or the fallback. */
export function fileNameOf(disposition: string | null, fallback: string): string {
  const m = /filename="([^"]+)"/.exec(disposition ?? '');
  const name = m?.[1]?.replace(/[/\\]/g, '') ?? '';
  return name || fallback;
}

/**
 * Downloads a file through fetch and a blob (D12), so a failure is an
 * ApiError the page can say in words, and nothing leaves the session.
 */
export async function download(
  path: string,
  fallback: string,
  method: 'GET' | 'POST' = 'GET',
): Promise<void> {
  const res = await request(method, path);
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement('a');
  a.href = url;
  a.download = fileNameOf(res.headers.get('Content-Disposition'), fallback);
  document.body.append(a);
  a.click();
  a.remove();
  // Some browsers read the blob after click returns.
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/** The pre-flight report as JSON and as CSV. */
export const checkReportFiles = {
  json: (id: string) => download(jobPath(id, '/check-report'), `karvan-check-${id}.json`),
  csv: (id: string) => download(jobPath(id, '/check-report.csv'), `karvan-check-${id}.csv`),
};
