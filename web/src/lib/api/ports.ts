import { api } from './client';

/** Who holds a listening port (D13). */
export type PortOwner = 'panel' | 'rule' | 'acme' | 'other';

export interface Ports {
  ports: { port: number; owner: PortOwner }[];
  /** The port D14 proposes for a new rule; 0 when none is free. */
  proposal: number;
}

/** `prefer` is the old domain's own port, proposed when 443 is taken. */
export function getPorts(prefer = 0): Promise<Ports> {
  return api<Ports>('GET', prefer > 0 ? `ports?prefer=${String(prefer)}` : 'ports');
}
