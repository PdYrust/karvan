import { api } from './client';

/** How Let's Encrypt checks the domain: automatic over HTTP, or a TXT record. */
export type AcmeMethod = 'http-01' | 'dns-01';

export type AcmeState =
  'starting' | 'waiting_dns' | 'validating' | 'issuing' | 'done' | 'failed' | 'cancelled';

/** An order error code with the CA's text; retryAt comes with a rate limit. */
export interface AcmeError {
  code: string;
  detail: string;
  retryAt?: string;
}

/** A Let's Encrypt order. */
export interface AcmeOrder {
  id: number;
  method: AcmeMethod;
  renewal: boolean;
  state: AcmeState;
  /** The TXT record a DNS-01 order waits for. */
  dns: { name: string; value: string } | null;
  error: AcmeError | null;
  createdAt: string;
  updatedAt: string;
  expiresAt: string | null;
}

export interface AcmeRenewal {
  method: AcmeMethod;
  nextAt: string | null;
  tries: number;
  lastError: AcmeError | null;
}

export interface AcmeSettings {
  staging: boolean;
  email: string;
  keepPort80: boolean;
}

/** The settings, whether the config's test CA is in use, and whether no agreement is needed any more. */
export interface AcmeSettingsView extends AcmeSettings {
  testDirectory: boolean;
  hasAccount: boolean;
}

const running: readonly AcmeState[] = ['starting', 'waiting_dns', 'validating', 'issuing'];

/** Whether an order still works toward a certificate, so the page polls it. */
export function isRunning(o: AcmeOrder | null | undefined): boolean {
  return !!o && running.includes(o.state);
}

export const acmeKeys = {
  domain: (id: number) => ['domains', id, 'acme'] as const,
  settings: ['settings', 'acme'] as const,
};

const path = (id: number) => `domains/${String(id)}/acme`;

/** The latest order and the renewal; polled every 2 seconds while an order runs. */
export function getAcme(
  id: number,
): Promise<{ order: AcmeOrder | null; renewal: AcmeRenewal | null }> {
  return api('GET', path(id), undefined, { background: true });
}

/** Starts an order now, also to renew ahead of time; 409 acme_busy, acme_port_*; 400 acme_invalid. */
export function startAcme(id: number, method: AcmeMethod, agreeTos = false): Promise<AcmeOrder> {
  return api('POST', path(id), { method, agreeTos });
}

/** Finishes a DNS-01 order; 409 acme_dns_not_found with the values seen, acme_state. */
export function checkAcme(id: number, force = false): Promise<AcmeOrder> {
  return api('POST', `${path(id)}/check`, { force });
}

export function cancelAcme(id: number): Promise<void> {
  return api('DELETE', path(id));
}

export function getAcmeSettings(): Promise<AcmeSettingsView> {
  return api('GET', 'settings/acme');
}

/** 400 acme_invalid for the email; 409 acme_port_* when keepPort80 cannot open the port. */
export function putAcmeSettings(s: AcmeSettings): Promise<AcmeSettingsView> {
  return api('PUT', 'settings/acme', s);
}
