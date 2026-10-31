import { api } from './client';

/**
 * The domain enabled for Karvan (0 for none). A handshake for its name gets its certificate;
 * every other one, the server's IP included, gets Karvan's own.
 */
export interface KarvanDomain {
  domainId: number;
  /** While the domain has no certificate: plain HTTP for its name, else Karvan's own certificate. */
  plain: boolean;
  /** The panel is on a shared HTTPS port, where its domain cannot change. */
  shared: boolean;
}

export const karvanKeys = { all: ['settings', 'karvan-domain'] as const };

export function getKarvanDomain(): Promise<KarvanDomain> {
  return api('GET', 'settings/karvan-domain');
}

/** Enables a domain (0 stops), or sets the plain HTTP choice; it applies at once. */
export function putKarvanDomain(change: {
  domainId?: number;
  plain?: boolean;
}): Promise<KarvanDomain> {
  return api('PUT', 'settings/karvan-domain', change);
}

/** How Karvan uses domain d: not, with its certificate, or without one on its own or over plain HTTP. */
export type KarvanUse = 'none' | 'cert' | 'own' | 'plain';

export function karvanUseOf(d: { id: number; certificate: unknown }, k: KarvanDomain): KarvanUse {
  if (k.domainId !== d.id) return 'none';
  if (d.certificate) return 'cert';
  return k.plain ? 'plain' : 'own';
}

/** An IP address, which a browser asks for with no server name: Karvan's own certificate serves it. */
export const isIpName = (name: string) =>
  name.includes(':') || /^\d{1,3}(\.\d{1,3}){3}$/.test(name);
