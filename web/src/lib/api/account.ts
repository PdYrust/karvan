import { api, setCsrfToken } from './client';
import type { Outcome } from './overview';

/** The current password, and the code when two-step sign-in is on. */
export interface Reauth {
  password: string;
  code: string;
}

export interface TotpSetup {
  secret: string;
  /** The otpauth address the QR code holds. */
  uri: string;
}

/** Starts two-step setup; the secret is shown once and is not stored until a code confirms it. */
export function beginTotp(): Promise<TotpSetup> {
  return api<TotpSetup>('POST', 'account/totp');
}

export function confirmTotp(code: string): Promise<void> {
  return api('POST', 'account/totp/confirm', { code });
}

export function disableTotp(r: Reauth): Promise<void> {
  return api('DELETE', 'account/totp', { password: r.password, code: r.code });
}

export interface AccountSession {
  id: string;
  ip: string;
  userAgent: string;
  createdAt: string;
  lastSeenAt: string;
  current: boolean;
}

export const accountKeys = {
  sessions: ['account', 'sessions'] as const,
  address: ['settings', 'address'] as const,
  audit: ['audit'] as const,
};

/**
 * Live sessions of the operator, most recently used first. Read in the
 * background: the list refreshes itself, and that must not keep this
 * session alive.
 */
export async function getSessions(): Promise<AccountSession[]> {
  const res = await api<{ sessions: AccountSession[] }>('GET', 'account/sessions', undefined, {
    background: true,
  });
  return res.sessions;
}

/** Signs out every other session; answers how many ended. */
export async function endOtherSessions(): Promise<number> {
  return (await api<{ ended: number }>('DELETE', 'account/sessions')).ended;
}

export function endSession(id: string): Promise<void> {
  return api('DELETE', `account/sessions/${encodeURIComponent(id)}`);
}

export interface NewLogin {
  username: string;
  password: string;
}

/**
 * A new username and password. Every other session ends and this browser
 * gets a new one, so the new CSRF token is kept at once.
 */
export async function renewLogin(r: Reauth): Promise<NewLogin> {
  const out = await api<NewLogin & { csrfToken: string }>('POST', 'account/credentials', {
    password: r.password,
    code: r.code,
  });
  setCsrfToken(out.csrfToken);
  return { username: out.username, password: out.password };
}

export interface PendingAddress {
  port: number;
  pathChanged: boolean;
  expiresAt: string;
  /** This page came through the new address, where Confirm works. */
  fromNew: boolean;
  /** How the new address is served. */
  tls: 'off' | 'self' | 'domain';
  /** The domain's name in domain mode; absent when the link keeps this host. */
  host?: string;
}

export interface Address {
  port: number;
  pending: PendingAddress | null;
}

export function getAddress(): Promise<Address> {
  return api<Address>('GET', 'settings/address');
}

/** The port to ask for: 0 keeps it, -1 lets the server pick one. */
export const keepPort = 0;
export const randomPort = -1;

export interface AddressStart {
  newPath: boolean;
  port: number;
}

export interface NewAddress {
  port: number;
  /** The hidden path of the new address, without slashes. Shown once. */
  path: string;
  expiresAt: string;
}

export function startAddress(change: AddressStart, r: Reauth): Promise<NewAddress> {
  return api<NewAddress>('POST', 'settings/address', {
    password: r.password,
    code: r.code,
    newPath: change.newPath,
    port: change.port,
  });
}

/** Saves the new address; works only through it. */
export function confirmAddress(): Promise<void> {
  return api('POST', 'settings/address/confirm');
}

/** The address a change serves, for this page's scheme and host, opened at `page`. */
export function newAddressURL(base: string, to: NewAddress, page = ''): string {
  const url = new URL(base);
  url.port = String(to.port);
  url.pathname = to.path ? `/${to.path}/${page}` : `/${page}`;
  url.search = '';
  url.hash = '';
  return url.toString();
}

export interface AuditEvent {
  id: number;
  at: string;
  actor: string;
  ip: string;
  action: string;
  outcome: Outcome;
  detail: Record<string, string>;
}

export interface AuditFilter {
  action: string;
  outcome: Outcome | '';
}

export interface AuditPage {
  events: AuditEvent[];
  next: number | null;
}

/** A page of the audit log, newest first; pass `next` back as `before` for the one after. */
export function getAudit(filter: AuditFilter, before?: number, limit = 50): Promise<AuditPage> {
  const q = new URLSearchParams({ limit: String(limit) });
  if (before !== undefined) q.set('before', String(before));
  if (filter.action) q.set('action', filter.action);
  if (filter.outcome) q.set('outcome', filter.outcome);
  return api<AuditPage>('GET', `audit?${q.toString()}`);
}
