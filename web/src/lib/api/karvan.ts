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
  /** "Only open Karvan on its domain". */
  only: boolean;
  /**
   * Who the panel opens for: everyone; everyone until a first sign-in through the domain; the
   * domain alone; or everyone again because the domain's certificate ended or cannot be read.
   */
  access: 'all' | 'waiting' | 'domain' | 'fallback';
  /** Why the IP opened again, with `fallback`. */
  reason?: 'expired' | 'missing';
  /** The end of the domain's certificate, within 14 days, when nothing renews it by itself. */
  endsSoon?: string;
}

export const karvanKeys = { all: ['settings', 'karvan-domain'] as const };

export function getKarvanDomain(): Promise<KarvanDomain> {
  return api('GET', 'settings/karvan-domain');
}

/** Enables a domain (0 stops), or sets plain HTTP or the domain-only switch; it applies at once. */
export function putKarvanDomain(change: {
  domainId?: number;
  plain?: boolean;
  only?: boolean;
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
