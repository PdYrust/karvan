import { isRunning } from '../../lib/api/acme';
import type { BatchGroup } from '../../lib/api/batches';
import { ApiError } from '../../lib/api/client';
import type { Domain } from '../../lib/api/domains';
import type { PanelKind } from '../../lib/api/panels';
import {
  sitePrefill,
  type LinkSite,
  type Redirect,
  type RedirectInput,
  type SiteInput,
  type SitePrefill,
} from '../../lib/api/rules';

/** What a rule is made from: an old panel's batches, an old panel typed by hand, a redirect, or one old link. */
export type RuleKind = 'batches' | 'manual' | 'redirect' | 'link';
export const ruleKinds: readonly RuleKind[] = ['batches', 'manual', 'redirect', 'link'];

/** The rule form as typed. Domains are names picked from the registered ones. */
export interface RuleValues {
  kind: RuleKind;
  name: string;
  /** The old panel a rule of batches serves, by its group of batches; 0 for none yet. */
  groupId: number;
  panelId: number;
  sourceKind: PanelKind;
  sourceUrl: string;
  sourceIp: string;
  sourcePin: string;
  oldPath: string;
  domains: string[];
  port: string;
  https: boolean;
  lazy: boolean;
  plainHttpOk: boolean;
  fromPath: string;
  match: 'exact' | 'prefix';
  toUrl: string;
  /** The pasted old link of a single link rule. */
  link: string;
}

/** The fields the server can name in a site_invalid or redirect_invalid answer. */
const ruleFields = [
  'name',
  'batchIds',
  'panelId',
  'sourceKind',
  'sourceUrl',
  'sourceIp',
  'sourcePin',
  'oldPath',
  'domains',
  'port',
  'https',
  'fromPath',
  'match',
  'toUrl',
] as const;
export type RuleField = (typeof ruleFields)[number];

/** What the address of a new rule can fill in: a migration's old panel, a batch, a kind. */
export interface RulePrefill extends SitePrefill {
  kind?: RuleKind;
  /** A batch a finished migration names (`?batch=12`); the form serves its old panel. */
  batchId?: number;
  /** That batch's old panel, once read. */
  groupId?: number;
}

export function rulePrefill(q: URLSearchParams): RulePrefill {
  const out: RulePrefill = sitePrefill(q);
  const kind = ruleKinds.find((k) => k === q.get('kind'));
  if (kind) out.kind = kind;
  const batch = Number(q.get('batch'));
  if (Number.isInteger(batch) && batch > 0) out.batchId = batch;
  return out;
}

/** A new rule: from batches unless the address names an old panel, on 443 with lazy checks on. */
export function newRuleValues(p: RulePrefill, panelId: number): RuleValues {
  return {
    kind: p.kind ?? (p.sourceUrl ? 'manual' : 'batches'),
    name: '',
    groupId: p.groupId ?? 0,
    panelId: p.panelId ?? panelId,
    sourceKind: p.sourceKind ?? 'marzban',
    sourceUrl: p.sourceUrl ?? '',
    sourceIp: '',
    sourcePin: '',
    oldPath: 'sub',
    domains: [],
    port: '443',
    https: false,
    lazy: true,
    plainHttpOk: false,
    fromPath: '',
    match: 'prefix',
    toUrl: '',
    link: '',
  };
}

/** A saved panel rule. A rule of batches got its old panel's IP from its batches, so it starts empty. */
export function valuesOfSite(s: LinkSite): RuleValues {
  const batches = s.mode === 'batches';
  return {
    ...newRuleValues({}, s.panelId),
    kind: batches ? 'batches' : 'manual',
    name: s.name,
    groupId: s.batchGroup ?? 0,
    sourceKind: s.sourceKind,
    sourceUrl: s.sourceUrl,
    sourceIp: batches ? '' : (s.sourceIp ?? ''),
    sourcePin: s.sourcePin ?? '',
    oldPath: s.oldPath,
    domains: s.domains ?? [],
    port: String(s.port),
    https: s.https,
    lazy: s.lazy,
    plainHttpOk: s.plainHttpOk,
  };
}

export function valuesOfRedirect(r: Redirect): RuleValues {
  return {
    ...newRuleValues({}, 0),
    kind: 'redirect',
    name: r.name,
    domains: r.domains ?? [],
    port: String(r.port),
    https: r.https,
    fromPath: r.fromPath,
    match: r.match,
    toUrl: r.toUrl,
  };
}

/** A domain's certificate as the picker shows it. */
export function domainState(d: Domain): 'https' | 'plain' | 'pending' {
  if (d.certificate) return 'https';
  return isRunning(d.acme?.order) ? 'pending' : 'plain';
}

/** HTTPS needs domains that all have a certificate. */
export function httpsAllowed(names: readonly string[], domains: readonly Domain[]): boolean {
  return (
    names.length > 0 &&
    names.every((n) => domains.some((d) => d.name === n && d.certificate !== null))
  );
}

/** New picked domains: HTTPS is on when they all have a certificate, else off. */
export function withDomains(
  v: RuleValues,
  names: string[],
  domains: readonly Domain[],
): RuleValues {
  return { ...v, domains: names, https: httpsAllowed(names, domains) };
}

/** The hosts that are not registered, so the form offers to add them first. */
export function unregistered(hosts: readonly string[], domains: readonly Domain[]): string[] {
  return hosts.filter((h) => !domains.some((d) => d.name === h));
}

/** What an old panel fills in: its old hosts that are registered, its old port and old path. */
export function groupDefaults(g: BatchGroup | undefined, domains: readonly Domain[]) {
  const hosts = [...(g?.oldLinks.hosts ?? [])].sort();
  const port = g?.oldLinks.port;
  return {
    hosts,
    domains: hosts.filter((h) => domains.some((d) => d.name === h)),
    port: port ? String(port) : '',
    oldPath: g?.oldLinks.path ?? '',
  };
}

/** A picked old panel: a new rule takes its registered old hosts and its old port. */
export function withGroup(
  v: RuleValues,
  id: number,
  all: readonly BatchGroup[],
  domains: readonly Domain[],
): RuleValues {
  const d = groupDefaults(
    all.find((g) => g.id === id),
    domains,
  );
  return withDomains({ ...v, groupId: id, port: d.port || v.port }, d.domains, domains);
}

/** A typed single link: its host is the domain once it is registered. */
export function withLink(v: RuleValues, text: string, domains: readonly Domain[]): RuleValues {
  const host = parseOldLink(text)?.host;
  const names = host && unregistered([host], domains).length === 0 ? [host] : [];
  return withDomains({ ...v, link: text }, names, domains);
}

/** An old panel on plain http: the switch that accepts it shows only then. */
export function isPlainHttp(url: string): boolean {
  return /^http:\/\//i.test(url.trim());
}

/** HTTPS goes only with domains; with none the port's certificate decides. */
const httpsOf = (v: RuleValues) => (v.domains.length > 0 ? { https: v.https } : {});

/**
 * A panel rule as sent. A rule of batches names its old panel by the newest batch of it, and
 * sends only what that old panel's batches do not know.
 */
export function siteInputOf(v: RuleValues, group: BatchGroup | undefined): SiteInput {
  const common = {
    name: v.name.trim(),
    domains: v.domains,
    port: Number(v.port),
    lazy: v.lazy,
    ...httpsOf(v),
  };
  if (v.kind === 'batches') {
    const ip = v.sourceIp.trim();
    return {
      ...common,
      mode: 'batches',
      batchIds: group ? [group.latestId] : [],
      plainHttpOk: !!group && isPlainHttp(group.source.origin) && v.plainHttpOk,
      ...(group?.oldLinks.path ? {} : { oldPath: v.oldPath.trim() }),
      ...(ip ? { sourceIp: ip } : {}),
    };
  }
  return {
    ...common,
    mode: 'manual',
    panelId: v.panelId,
    sourceKind: v.sourceKind,
    sourceUrl: v.sourceUrl.trim(),
    sourceIp: v.sourceIp.trim(),
    sourcePin: v.sourcePin.trim(),
    oldPath: v.oldPath.trim(),
    plainHttpOk: isPlainHttp(v.sourceUrl) && v.plainHttpOk,
  };
}

/** The parts of an old subscription link a single link rule answers. */
export function parseOldLink(
  text: string,
): { host: string; port: number; path: string } | undefined {
  let u: URL;
  try {
    u = new URL(text.trim());
  } catch {
    return undefined;
  }
  if (u.protocol !== 'https:' && u.protocol !== 'http:') return undefined;
  const path = u.pathname.replace(/\/+$/, '');
  if (!path) return undefined;
  const port = u.port ? Number(u.port) : u.protocol === 'https:' ? 443 : 80;
  return { host: u.hostname.replace(/^\[(.*)\]$/, '$1').toLowerCase(), port, path };
}

/** A redirect as sent; a single link is a redirect of its path and below. */
export function redirectInputOf(v: RuleValues): RedirectInput {
  const common = { name: v.name.trim(), toUrl: v.toUrl.trim() };
  const link = v.kind === 'link' ? parseOldLink(v.link) : undefined;
  if (link) {
    return {
      ...common,
      domains: [link.host],
      port: link.port,
      fromPath: link.path,
      match: 'prefix',
      https: v.https,
    };
  }
  return {
    ...common,
    ...httpsOf(v),
    domains: v.domains,
    port: Number(v.port),
    fromPath: v.fromPath.trim(),
    match: v.match,
  };
}

/** The form can save once the kind has what it needs; the server checks the rest. */
export function canSubmit(v: RuleValues, domains: readonly Domain[]): boolean {
  if (!v.name.trim()) return false;
  switch (v.kind) {
    case 'batches':
      return v.groupId > 0;
    case 'manual':
      return v.sourceUrl.trim() !== '';
    case 'redirect':
      return v.fromPath.trim() !== '' && v.toUrl.trim() !== '';
    case 'link': {
      const link = parseOldLink(v.link);
      return !!link && unregistered([link.host], domains).length === 0 && v.toUrl.trim() !== '';
    }
  }
}

/** The field a refused save names, when the server named one this form shows. */
export function refusedField(err: unknown): RuleField | undefined {
  if (!(err instanceof ApiError)) return undefined;
  if (err.code !== 'site_invalid' && err.code !== 'redirect_invalid') return undefined;
  const f = err.detail.field;
  return ruleFields.find((x) => x === f);
}
