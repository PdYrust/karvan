import { api } from './client';
import type { PanelKind } from './panels';

/** What is wrong with a certificate, as the server names it (api-resources.md 5.0). */
export type CertWarning =
  | 'unreadable'
  | 'no_key'
  | 'key_mismatch'
  | 'domain_mismatch'
  | 'expired'
  | 'not_yet_valid'
  | 'expires_soon';

/** What Karvan read from a certificate. With `unreadable` the rest is empty. */
export interface CertFacts {
  domains: string[];
  issuer: string;
  notBefore?: string;
  notAfter?: string;
  warnings: CertWarning[];
}

/** The old panel's users moved into the destination: proven, closed for good, the rest pending. */
export interface SiteStats {
  moved: number;
  verified: number;
  closed: number;
}

/** A site that keeps an old panel's subscription links working (Old links). */
export interface LinkSite {
  id: number;
  name: string;
  /** The destination panel. */
  panelId: number;
  sourceKind: PanelKind;
  sourceUrl: string;
  sourceIp?: string;
  sourcePin?: string;
  oldPath: string;
  /** Null or empty: every host on the port. */
  domains: string[] | null;
  port: number;
  certFile?: string;
  keyFile?: string;
  certPem?: string;
  hasKey: boolean;
  lazy: boolean;
  plainHttpOk: boolean;
  /** Only with HTTPS. */
  cert?: CertFacts;
  /** Missing when the destination did not answer. */
  stats?: SiteStats;
}

export interface LinkSites {
  sites: LinkSite[];
  /** Ports that could not listen, by port number. */
  portErrors: Record<string, string>;
}

export const linkKeys = {
  all: ['legacy-sites'] as const,
  upload: (id: number) => ['legacy-sites', 'upload', id] as const,
};

export function getLinkSites(): Promise<LinkSites> {
  return api<LinkSites>('GET', 'legacy-sites');
}

/** A site as sent. `keyPem` goes in and never comes back; empty on a change keeps the saved key. */
export type SiteInput = Omit<LinkSite, 'id' | 'hasKey' | 'cert' | 'stats' | 'domains'> & {
  domains: string[];
  keyPem?: string;
};

/** Adds a site (no id) or replaces one; 400 site_invalid names the field in `detail.field`. */
export function saveSite(input: SiteInput, id?: number): Promise<LinkSite> {
  return id
    ? api('PATCH', `legacy-sites/${String(id)}`, input)
    : api('POST', 'legacy-sites', input);
}

export function deleteSite(id: number): Promise<void> {
  return api('DELETE', `legacy-sites/${String(id)}`);
}

/** A certificate to check before saving; with `id` and no key, the site's saved key is used (api-resources.md 10.6). */
export interface CertInput {
  id?: number;
  certFile?: string;
  keyFile?: string;
  certPem?: string;
  keyPem?: string;
  domains: string[];
}

export function inspectCert(input: CertInput): Promise<CertFacts> {
  return api('POST', 'certificates/inspect', input);
}

/** A token list checked in the background; `checked` of `found` are done. */
export interface UploadProgress {
  state: 'none' | 'running' | 'done' | 'stopped';
  found: number;
  checked: number;
  stored: number;
  known: number;
  noUser: number;
  refused: number;
  failed: number;
  startedAt?: string;
  endedAt?: string;
}

/** Sends a text or CSV file of old links; 413 too_large over 16 MB, 409 busy while one runs. */
export function uploadTokens(id: number, file: Blob): Promise<UploadProgress> {
  return api('POST', `legacy-sites/${String(id)}/tokens`, undefined, { raw: file });
}

/** Read while a list is checked, so it does not keep the session awake. */
export function getUpload(id: number): Promise<UploadProgress> {
  return api('GET', `legacy-sites/${String(id)}/tokens`, undefined, { background: true });
}

/** Pending users: moved, not proven yet and not closed. */
export function pendingOf(s: SiteStats): number {
  return Math.max(0, s.moved - s.verified - s.closed);
}

/**
 * An old panel address as the ledger keeps it (scheme, host and port lower
 * case, default port dropped, no trailing slash), so a site finds the jobs
 * that moved users from the same panel.
 */
export function originOf(address: string): string {
  try {
    const u = new URL(address);
    const path = u.pathname.replace(/\/+$/, '').toLowerCase();
    return `${u.protocol}//${u.host.toLowerCase()}${path}`;
  } catch {
    return address.trim().toLowerCase();
  }
}

/** What a finished migration can fill in on a new site's form. */
export interface SitePrefill {
  panelId?: number;
  sourceKind?: PanelKind;
  sourceUrl?: string;
}

/** The new site form, prefilled from a migration: `/links/new?panel=2&kind=marzban&from=...`. */
export function newSitePath(p: SitePrefill): string {
  const query = prefillQuery(p);
  return query ? `/links/new?${query}` : '/links/new';
}

/** The prefill as a query string; Connect reads the same one to start a migration again. */
export function prefillQuery(p: SitePrefill): string {
  const q = new URLSearchParams();
  if (p.panelId) q.set('panel', String(p.panelId));
  if (p.sourceKind) q.set('kind', p.sourceKind);
  if (p.sourceUrl) q.set('from', p.sourceUrl);
  return q.toString();
}

/** Reads the prefill back from the address, keeping only values the form can use. */
export function sitePrefill(q: URLSearchParams): SitePrefill {
  const out: SitePrefill = {};
  const panel = Number(q.get('panel'));
  if (Number.isInteger(panel) && panel > 0) out.panelId = panel;
  const kind = q.get('kind');
  if (kind === 'marzban' || kind === 'pasarguard') out.sourceKind = kind;
  const from = q.get('from') ?? '';
  try {
    const url = new URL(from);
    if (url.protocol === 'https:' || url.protocol === 'http:') out.sourceUrl = from;
  } catch {
    // Not an address: the operator types it.
  }
  return out;
}
