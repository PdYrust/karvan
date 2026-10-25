import { api } from './client';
import type { Conflict, Counts, JobState } from './migrations';
import type { PanelKind } from './panels';
import { download } from './preflight';

/** What the reload did (api-migration.md 10.1). `error` and `failed` are server text: never shown. */
export interface ReloadResult {
  /** "" when the run ended before its reload. */
  method: string;
  ok: number;
  /** "username: error" lines, at most a few, then "N more". */
  failed?: string[];
  error?: string;
  command?: string;
}

export interface StageTime {
  stage: string;
  started: string;
  ended: string;
  ok: boolean;
}

/** One user's outcome in the report. */
export interface ReportUser {
  key: string;
  sourceUsername: string;
  username: string;
  moved: boolean;
  destId?: number;
  applied?: string[];
  fixed?: string[];
  changesConfig: boolean;
  changesLink: boolean;
}

/** The record of a run, karvan.report/v1. It holds no secret. */
export interface Report {
  schema: string;
  jobId: string;
  state: JobState;
  safeMode: boolean;
  /** What Safe Mode does on a failure: undo or backup. */
  rollback?: string;
  backupId?: string;
  startedAt: string;
  endedAt: string;
  source: { kind: PanelKind | ''; origin: string; admin: string };
  destination: { kind: PanelKind | ''; name: string; db: string };
  admin: {
    username: string;
    id?: number;
    merged: boolean;
    roleId?: number;
    password?: 'kept' | 'typed' | 'generated';
  };
  counts: Counts;
  users: ReportUser[] | null;
  conflicts?: Conflict[];
  notUndoable: string[] | null;
  reload: ReloadResult;
  stages: StageTime[] | null;
  /** The code the run or the rollback failed with. */
  error?: string;
  /** Destination rows a rollback could not remove. */
  left?: number[];
}

const jobPath = (id: string, rest = '') => `migrations/${encodeURIComponent(id)}${rest}`;

/** The stored report; 404 before a run ends and for an interrupted job, 409 busy while it runs. */
export function getReport(id: string): Promise<Report> {
  return api('GET', jobPath(id, '/report'));
}

/** Takes back what a failed or interrupted job wrote; the undo follows on the job's stream (B5). */
export function rollbackJob(id: string): Promise<{ id: string }> {
  return api('POST', jobPath(id, '/rollback'));
}

/** A finished job's files, each through fetch and a blob (D12). */
export const reportFiles = {
  json: (id: string) => download(jobPath(id, '/report'), `karvan-report-${id}.json`),
  csv: (id: string) => download(jobPath(id, '/report.csv'), `karvan-report-${id}.csv`),
  links: (id: string) => download(jobPath(id, '/links.csv'), `karvan-links-${id}.csv`),
  /** Once only: the first download empties it (B9). Only on a click, never retried. */
  passwords: (id: string) =>
    download(jobPath(id, '/passwords.csv'), `karvan-passwords-${id}.csv`, 'POST'),
};

/** How the reload went, without the server's own text. */
export type ReloadOutcome =
  | { kind: 'none' }
  | { kind: 'manual'; command: string }
  | { kind: 'error'; command: string }
  | { kind: 'partial'; ok: number; failed: number; command: string }
  | { kind: 'ok'; method: string; ok: number };

const more = /^(\d+) more$/;

/** Users the panel did not pick up: one per listed line, plus the "N more" the list ends with. */
function failedCount(lines: readonly string[]): number {
  return lines.reduce((n, line) => {
    const m = more.exec(line);
    return n + (m ? Number(m[1]) : 1);
  }, 0);
}

/** The panel's own restart command, as the server names it (run.RestartCommand). */
export function restartCommand(kind: PanelKind | ''): string {
  return kind === 'pasarguard' ? 'pasarguard restart' : 'marzban restart';
}

/** How the reload on a panel of this kind went; a failure always comes with the command to run. */
export function reloadOutcome(r: ReloadResult | undefined, kind: PanelKind | ''): ReloadOutcome {
  if (!r?.method) return { kind: 'none' };
  const command = r.command || restartCommand(kind);
  if (r.method === 'manual') return { kind: 'manual', command };
  if (r.error) return { kind: 'error', command };
  const failed = failedCount(r.failed ?? []);
  if (failed > 0) return { kind: 'partial', ok: r.ok, failed, command };
  return { kind: 'ok', method: r.method, ok: r.ok };
}
