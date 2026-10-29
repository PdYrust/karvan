import { ApiError } from '../../lib/api/client';
import type { LinkSite, SiteInput, SitePrefill } from '../../lib/api/rules';
import type { PanelKind } from '../../lib/api/panels';

/** The site form as typed: every field a string or a switch. */
export interface SiteValues {
  name: string;
  panelId: number;
  sourceKind: PanelKind;
  sourceUrl: string;
  sourceIp: string;
  sourcePin: string;
  oldPath: string;
  domains: string;
  port: string;
  lazy: boolean;
  plainHttpOk: boolean;
}

/** The fields the server can name in a site_invalid answer. */
export const siteFields = [
  'name',
  'panelId',
  'sourceKind',
  'sourceUrl',
  'sourceIp',
  'sourcePin',
  'oldPath',
  'domains',
  'port',
] as const;
export type SiteField = (typeof siteFields)[number];

/** A new site: the old panel's usual path, HTTPS on 443, lazy checks on, then the prefill. */
export function newValues(p: SitePrefill, panelId: number): SiteValues {
  return {
    name: '',
    panelId: p.panelId ?? panelId,
    sourceKind: p.sourceKind ?? 'marzban',
    sourceUrl: p.sourceUrl ?? '',
    sourceIp: '',
    sourcePin: '',
    oldPath: 'sub',
    domains: '',
    port: '443',
    lazy: true,
    plainHttpOk: false,
  };
}

/** A saved site in the form. */
export function valuesOf(s: LinkSite): SiteValues {
  return {
    name: s.name,
    panelId: s.panelId,
    sourceKind: s.sourceKind,
    sourceUrl: s.sourceUrl,
    sourceIp: s.sourceIp ?? '',
    sourcePin: s.sourcePin ?? '',
    oldPath: s.oldPath,
    domains: (s.domains ?? []).join('\n'),
    port: String(s.port),
    lazy: s.lazy,
    plainHttpOk: s.plainHttpOk,
  };
}

/** Domains typed one per line, or split by spaces or commas. */
export function parseDomains(text: string): string[] {
  return text.split(/[\s,]+/).filter(Boolean);
}

/** An old panel on plain http: the switch that accepts it shows only then. */
export function isPlainHttp(url: string): boolean {
  return /^http:\/\//i.test(url.trim());
}

export function inputOf(v: SiteValues): SiteInput {
  return {
    name: v.name.trim(),
    panelId: v.panelId,
    sourceKind: v.sourceKind,
    sourceUrl: v.sourceUrl.trim(),
    sourceIp: v.sourceIp.trim(),
    sourcePin: v.sourcePin.trim(),
    oldPath: v.oldPath.trim(),
    domains: parseDomains(v.domains),
    port: Number(v.port),
    lazy: v.lazy,
    plainHttpOk: isPlainHttp(v.sourceUrl) && v.plainHttpOk,
  };
}

/** The field a refused save names, when the server named one this form shows. */
export function refusedField(err: unknown): SiteField | undefined {
  if (!(err instanceof ApiError) || err.code !== 'site_invalid') return undefined;
  const f = err.detail.field;
  return siteFields.find((x) => x === f);
}
