import { api } from './client';

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
}

/** Every backup, newest first. */
export async function getBackups(): Promise<Backup[]> {
  return (await api<{ backups: Backup[] }>('GET', 'safe-mode/backups')).backups;
}

/** `enabled` is always sent: the server reads a missing one as "off". */
export function putSafeMode(next: SafeMode, confirm = ''): Promise<SafeMode> {
  return api<SafeMode>('PUT', 'settings/safe-mode', {
    enabled: next.enabled,
    rollback: next.rollback,
    confirm,
  });
}
