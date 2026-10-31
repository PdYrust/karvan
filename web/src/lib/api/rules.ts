import { api } from './client';
import { factsOf, list } from './lists';
import type { PanelKind } from './panels';

/** What is wrong with a certificate, as the server names it. */
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

/** A batch a panel rule serves. */
export interface RuleBatch {
  id: number;
  name: string;
  state: 'active' | 'inactive' | 'deleted';
}
/** The certificate a rule serves a handshake without a known name with; read only. */
export interface RuleCertificate {
  id: number;
  name: string;
  facts: CertFacts;
}
/** A panel rule: an old panel's subscription links answered from the new panel. */
export interface LinkSite {
  id: number;
  name: string;
  /** Made from batches, or typed by hand; fixed once saved. */
  mode: 'manual' | 'batches';
  batches: RuleBatch[];
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
  lazy: boolean;
  plainHttpOk: boolean;
  /** Whether the rule answers over HTTPS; its certificate comes from its domains. */
  https: boolean;
  /** Only with HTTPS. */
  cert?: CertFacts;
  certificate: RuleCertificate | null;
  /** Its domains' warnings, and certificate_not_adopted. */
  warnings: string[];
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

/** A rule's certificate with its lists. */
function certOf(c: RuleCertificate | null): RuleCertificate | null {
  return c && { ...c, facts: factsOf(c.facts) };
}

/** A panel rule with its lists, whatever the server wrote for an empty one. */
function siteOf(s: LinkSite): LinkSite {
  return {
    ...s,
    batches: list(s.batches),
    domains: list(s.domains),
    cert: s.cert && factsOf(s.cert),
    certificate: certOf(s.certificate),
    warnings: list(s.warnings),
  };
}

export async function getLinkSites(): Promise<LinkSites> {
  const r = await api<{ sites: LinkSite[] | null; portErrors: LinkSites['portErrors'] | null }>(
    'GET',
    'legacy-sites',
  );
  return { sites: list(r.sites).map(siteOf), portErrors: r.portErrors ?? {} };
}

/**
 * A site as sent. Without `https`, a new rule answers over HTTPS when every domain has a certificate, and a
 * change keeps what it was.
 */
export interface SiteInput {
  name: string;
  mode: LinkSite['mode'];
  /** A rule of batches takes the panel, the old panel and its pin from them. */
  batchIds?: number[];
  panelId?: number;
  sourceKind?: PanelKind;
  sourceUrl?: string;
  sourceIp?: string;
  sourcePin?: string;
  oldPath?: string;
  domains: string[];
  port: number;
  lazy: boolean;
  plainHttpOk: boolean;
  https?: boolean;
}

/** Adds a site (no id) or replaces one; 400 site_invalid names the field in `detail.field`. */
export async function saveSite(input: SiteInput, id?: number): Promise<LinkSite> {
  const s = await (id
    ? api<LinkSite>('PATCH', `legacy-sites/${String(id)}`, input)
    : api<LinkSite>('POST', 'legacy-sites', input));
  return siteOf(s);
}

export function deleteSite(id: number): Promise<void> {
  return api('DELETE', `legacy-sites/${String(id)}`);
}

/** A rule that answers matching old links with a 302 to a fixed address. */
export interface Redirect {
  id: number;
  name: string;
  domains: string[] | null;
  port: number;
  /** With one leading slash. */
  fromPath: string;
  match: 'exact' | 'prefix';
  toUrl: string;
  https: boolean;
  certificate: RuleCertificate | null;
  warnings: string[];
}
export type RedirectInput = Omit<
  Redirect,
  'id' | 'domains' | 'https' | 'certificate' | 'warnings'
> & {
  domains: string[];
  https?: boolean;
};
export const redirectKeys = {
  all: ['link-redirects'] as const,
};
/** A redirect rule with its lists, whatever the server wrote for an empty one. */
function redirectOf(r: Redirect): Redirect {
  return {
    ...r,
    domains: list(r.domains),
    certificate: certOf(r.certificate),
    warnings: list(r.warnings),
  };
}

export async function getRedirects(): Promise<{ redirects: Redirect[] }> {
  const r = await api<{ redirects: Redirect[] | null }>('GET', 'link-redirects');
  return { redirects: list(r.redirects).map(redirectOf) };
}
/** Adds a redirect (no id) or replaces one; 400 redirect_invalid names the field. */
export async function saveRedirect(input: RedirectInput, id?: number): Promise<Redirect> {
  const r = await (id
    ? api<Redirect>('PATCH', `link-redirects/${String(id)}`, input)
    : api<Redirect>('POST', 'link-redirects', input));
  return redirectOf(r);
}
export function deleteRedirect(id: number): Promise<void> {
  return api('DELETE', `link-redirects/${String(id)}`);
}
/** Where a rule is changed: a panel rule by its id, a redirect under its own path. */
export function rulePath(r: { kind: 'panel' | 'redirect'; id: number }): string {
  return r.kind === 'redirect' ? `/rules/redirects/${String(r.id)}` : `/rules/${String(r.id)}`;
}
/** A new rule from a batch, as a finished migration offers it. */
export function newBatchRulePath(batchId: number): string {
  return `/rules/new?batch=${String(batchId)}`;
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

/** The new site form, prefilled from a migration: `/rules/new?panel=2&kind=marzban&from=...`. */
export function newSitePath(p: SitePrefill): string {
  const query = prefillQuery(p);
  return query ? `/rules/new?${query}` : '/rules/new';
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
