import type { Reauth } from './account';
import { api } from './client';

/** How the panel is served (D4): plain HTTP, a self-signed certificate, or a domain's certificate. */
export type TlsMode = 'off' | 'self' | 'domain';

export interface HttpsSite {
  tls: TlsMode;
  domainId: number;
  /** On a bridge HTTPS port, under the hidden path (D12). */
  shared: boolean;
  port: number;
}

export interface PendingHttps extends HttpsSite {
  /** A new hidden path waits with it (an address change). */
  pathChanged: boolean;
  expiresAt: string;
  /** This page came through the new address, where Confirm works. */
  fromNew: boolean;
}

export interface Https extends HttpsSite {
  /** SHA-256 of the self-signed certificate, when the current or the waiting address uses it. */
  fingerprint?: string;
  pending: PendingHttps | null;
  /** Each port where an HTTPS rule holds the domain, for a shared address. */
  sharedPorts: { domain: string; port: number }[];
}

export const httpsKeys = { all: ['settings', 'https'] as const };

export function getHttps(): Promise<Https> {
  return api<Https>('GET', 'settings/https');
}

/** The change to start; port 0 keeps the port. */
export interface HttpsStart {
  tls: TlsMode;
  domainId: number;
  port: number;
  shared: boolean;
}

/** Serves the new mode next to the current address until it is confirmed from there (D7, D10). */
export function startHttps(
  change: HttpsStart,
  r: Reauth,
): Promise<{ port: number; expiresAt: string }> {
  return api('PUT', 'settings/https', { ...change, password: r.password, code: r.code });
}
