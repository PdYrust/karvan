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

/** `enabled` is always sent: the server reads a missing one as "off". */
export function putSafeMode(next: SafeMode, confirm = ''): Promise<SafeMode> {
  return api<SafeMode>('PUT', 'settings/safe-mode', {
    enabled: next.enabled,
    rollback: next.rollback,
    confirm,
  });
}
