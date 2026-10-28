import { ApiError } from '../../lib/api/client';
import type { CertInput, LinkSite, SiteInput, SitePrefill } from '../../lib/api/rules';
import type { PanelKind } from '../../lib/api/panels';

/** How the site gets its certificate. Upload and paste both send PEM text. */
export type CertMode = 'files' | 'upload' | 'paste' | 'none';

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
  certMode: CertMode;
  certFile: string;
  keyFile: string;
  certPem: string;
  keyPem: string;
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
  'certFile',
  'keyFile',
  'certPem',
  'keyPem',
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
    certMode: 'files',
    certFile: '',
    keyFile: '',
    certPem: '',
    keyPem: '',
    lazy: true,
    plainHttpOk: false,
  };
}

/** A saved site in the form. The key never comes back, so its field starts empty. */
export function valuesOf(s: LinkSite): SiteValues {
  let certMode: CertMode = 'none';
  if (s.certFile) certMode = 'files';
  else if (s.certPem) certMode = 'paste';
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
    certMode,
    certFile: s.certFile ?? '',
    keyFile: s.keyFile ?? '',
    certPem: s.certPem ?? '',
    keyPem: '',
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

/** The certificate part of a site or a check: only the chosen way's fields. */
function certPart(v: SiteValues): Pick<CertInput, 'certFile' | 'keyFile' | 'certPem' | 'keyPem'> {
  switch (v.certMode) {
    case 'files':
      return { certFile: v.certFile.trim(), keyFile: v.keyFile.trim() };
    case 'upload':
    case 'paste':
      return v.keyPem ? { certPem: v.certPem, keyPem: v.keyPem } : { certPem: v.certPem };
    case 'none':
      return {};
  }
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
    ...certPart(v),
  };
}

/** The certificate check of what the form holds; a saved site lends its stored key. */
export function certInputOf(v: SiteValues, id?: number): CertInput {
  return { ...(id ? { id } : {}), domains: parseDomains(v.domains), ...certPart(v) };
}

/** The form can check a certificate once the chosen way has one to read. */
export function canCheckCert(v: SiteValues): boolean {
  if (v.certMode === 'files') return v.certFile.trim() !== '' && v.keyFile.trim() !== '';
  return v.certMode !== 'none' && v.certPem.trim() !== '';
}

/** The field a refused save names, when the server named one this form shows. */
export function refusedField(err: unknown): SiteField | undefined {
  if (!(err instanceof ApiError) || err.code !== 'site_invalid') return undefined;
  const f = err.detail.field;
  return siteFields.find((x) => x === f);
}
