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
