import { api } from './client';

/** How Karvan runs on this server. */
export interface Install {
  runsAsRoot: boolean;
}

export function getInstall(): Promise<Install> {
  return api<Install>('GET', 'settings/install');
}
