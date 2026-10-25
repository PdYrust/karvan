import { api } from './client';
import type { PanelKind } from './panels';

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
}

export interface LinkSites {
  sites: LinkSite[];
  /** Ports that could not listen, by port number. */
  portErrors: Record<string, string>;
}

export function getLinkSites(): Promise<LinkSites> {
  return api<LinkSites>('GET', 'legacy-sites');
}

/** What a finished migration can fill in on a new site's form. */
export interface SitePrefill {
  panelId?: number;
  sourceKind?: PanelKind;
  sourceUrl?: string;
}

/** The new site form, prefilled from a migration: `/links/new?panel=2&kind=marzban&from=...`. */
export function newSitePath(p: SitePrefill): string {
  const q = new URLSearchParams();
  if (p.panelId) q.set('panel', String(p.panelId));
  if (p.sourceKind) q.set('kind', p.sourceKind);
  if (p.sourceUrl) q.set('from', p.sourceUrl);
  const query = q.toString();
  return query ? `/links/new?${query}` : '/links/new';
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
