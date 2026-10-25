import { api } from './client';
import type { JobState } from './migrations';
import type { PanelKind } from './panels';

export type Rollback = 'undo' | 'backup';

export interface SafeMode {
  enabled: boolean;
  rollback: Rollback;
}

/** The typed confirmation the server wants before Safe Mode goes off. */
export const turnOffConfirm = 'turn-off-safe-mode';

export function getSafeMode(): Promise<SafeMode> {
  return api<SafeMode>('GET', 'settings/safe-mode');
}

/** A backup Safe Mode took before a run wrote anything. Kept until the operator deletes it. */
export interface Backup {
  id: string;
  jobId: string;
  /** The database it holds, in the panel's address format without the password. */
  dest: string;
  /** Bytes of the encrypted file. */
  size: number;
  method: 'vacuum-into' | 'mysqldump' | 'mariadb-dump' | 'pg_dump';
  createdAt: string;
  /** The job's destination panel; 0 when the job is gone. */
  panelId: number;
  /** The job's state now; empty when the job is gone. */
  jobState: JobState | '';
  sourceKind: PanelKind | '';
  sourceOrigin: string;
  sourceAdmin: string;
}

export const backupKeys = { all: ['safe-mode', 'backups'] as const };

/** Every backup, newest first. */
export async function getBackups(): Promise<Backup[]> {
  return (await api<{ backups: Backup[] }>('GET', 'safe-mode/backups')).backups;
}

/** A restore the server started (B12): the job whose stream follows it, and its last event before it. */
export interface Restoring {
  jobId: string;
  after: number;
}

/** Puts a backup back in the background; `confirm` is the backup id, typed. */
export function restoreBackup(id: string, confirm: string): Promise<Restoring> {
  return api<Restoring>('POST', `safe-mode/backups/${encodeURIComponent(id)}/restore`, {
    confirm,
  });
}

export function deleteBackup(id: string): Promise<void> {
  return api('DELETE', `safe-mode/backups/${encodeURIComponent(id)}`);
}

/** The database a backup holds, from how it was made. */
export const backupDatabase: Readonly<Record<Backup['method'], string>> = {
  'vacuum-into': 'SQLite',
  mysqldump: 'MySQL',
  'mariadb-dump': 'MariaDB',
  pg_dump: 'PostgreSQL',
};

/** `enabled` is always sent: the server reads a missing one as "off". */
export function putSafeMode(next: SafeMode, confirm = ''): Promise<SafeMode> {
  return api<SafeMode>('PUT', 'settings/safe-mode', {
    enabled: next.enabled,
    rollback: next.rollback,
    confirm,
  });
}
