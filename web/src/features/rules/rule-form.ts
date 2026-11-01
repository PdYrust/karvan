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

/** What a rule is made from: an old panel's batches, or a custom redirect rule. */
export type RuleKind = 'batches' | 'custom';
export const ruleKinds: readonly RuleKind[] = ['batches', 'custom'];

/** The steps of the rule wizard: kind (on create), source domain, HTTPS, then details. */
export type RuleStep = 'kind' | 'domain' | 'https' | 'details';

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
  fromPaths: string[];
  match: 'exact' | 'prefix';
  toUrl: string;
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
  const rawKind = q.get('kind');
  if (rawKind === 'batches') out.kind = 'batches';
  else if (rawKind === 'custom' || rawKind === 'redirect') out.kind = 'custom';
  const batch = Number(q.get('batch'));
  if (Number.isInteger(batch) && batch > 0) out.batchId = batch;
  return out;
}

/** A new rule: from batches unless custom is picked, on 443 with lazy checks on. */
export function newRuleValues(p: RulePrefill, panelId: number): RuleValues {
  return {
    kind: p.kind ?? 'batches',
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
    fromPath: '/sub',
    fromPaths: ['sub'],
    match: 'prefix',
    toUrl: '',
  };
}

/** A saved panel rule. A rule of batches got its old panel's IP from its batches, so it starts empty. */
export function valuesOfSite(s: LinkSite): RuleValues {
  const batches = s.mode === 'batches';
  return {
    ...newRuleValues({}, s.panelId),
    kind: 'batches',
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
  const raw = r.fromPaths && r.fromPaths.length > 0 ? r.fromPaths : [r.fromPath];
  const fromPaths = raw.map((p) => p.replace(/^\/+/, '')).filter(Boolean);
  return {
    ...newRuleValues({}, 0),
    kind: 'custom',
    name: r.name,
    domains: r.domains ?? [],
    port: String(r.port),
    https: r.https,
    fromPath: r.fromPath,
    fromPaths: fromPaths.length > 0 ? fromPaths : [''],
    match: r.match,
    toUrl: r.toUrl,
  };
}

/** The steps the rule form walks: editing skips the kind step. */
export function stepsForRule(isNew: boolean): RuleStep[] {
  return isNew ? ['kind', 'domain', 'https', 'details'] : ['domain', 'https', 'details'];
}

/** Whether the source domain step has what it needs to continue to HTTPS. */
export function canContinueDomain(v: RuleValues): boolean {
  if (!v.name.trim()) return false;
  return v.kind === 'custom' || v.groupId > 0 || v.sourceUrl.trim() !== '';
}

/** The step that owns a refused field, so the wizard jumps back to it when needed. */
export function stepForRefused(field: RuleField | undefined): RuleStep | undefined {
  if (!field) return undefined;
  if (field === 'name' || field === 'domains' || field === 'batchIds') return 'domain';
  if (field === 'https') return 'https';
  return 'details';
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

/** A picked old panel: a new rule takes its registered old hosts, its old port and its old path. */
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
  return withDomains(
    { ...v, groupId: id, port: d.port || v.port, oldPath: d.oldPath || v.oldPath },
    d.domains,
    domains,
  );
}

/** An old panel on plain http: the switch that accepts it shows only then. */
export function isPlainHttp(url: string): boolean {
  return /^http:\/\//i.test(url.trim());
}

/** Normalizes one source path to start with a single leading slash, or empty string. */
export function normalizePath(raw: string): string {
  const trimmed = raw.trim().replace(/^\/+|\/+$/g, '');
  return trimmed ? `/${trimmed}` : '';
}

/** Deduplicates and normalizes all non-empty source paths. */
export function cleanPaths(paths: readonly string[]): string[] {
  const out: string[] = [];
  for (const raw of paths) {
    const p = normalizePath(raw);
    if (p && !out.includes(p)) out.push(p);
  }
  return out;
}

const localHostRe = /^(localhost|127\.|10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.|\[::1\])/i;

/** Ensures a target address has http:// or https:// when a bare domain or IP with port is typed. */
export function normalizeTargetUrl(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed || /^[a-z][a-z0-9+.-]*:\/\//i.test(trimmed) || trimmed.startsWith('/')) {
    return trimmed;
  }
  const scheme = localHostRe.test(trimmed) ? 'http://' : 'https://';
  return `${scheme}${trimmed}`;
}

/** HTTPS goes only with domains; with none the port's certificate decides. */
const httpsOf = (v: RuleValues) => (v.domains.length > 0 ? { https: v.https } : {});

/**
 * A panel rule as sent. A rule of batches names its old panel by the newest batch of it, and
 * sends its old path, which wins over the batches', and the address only when typed.
 */
export function siteInputOf(v: RuleValues, group: BatchGroup | undefined): SiteInput {
  const common = {
    name: v.name.trim(),
    domains: v.domains,
    port: Number(v.port),
    lazy: v.lazy,
    ...httpsOf(v),
  };
  if (v.groupId > 0 || group) {
    const ip = v.sourceIp.trim();
    return {
      ...common,
      mode: 'batches',
      batchIds: group ? [group.latestId] : [],
      plainHttpOk: !!group && isPlainHttp(group.source.origin) && v.plainHttpOk,
      oldPath: v.oldPath.trim(),
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

/** A custom redirect rule as sent, with all allowed source paths and normalized target URL. */
export function redirectInputOf(v: RuleValues): RedirectInput {
  const paths = cleanPaths(v.fromPaths.length > 0 ? v.fromPaths : [v.fromPath]);
  const first = paths[0] ?? normalizePath(v.fromPath);
  return {
    name: v.name.trim(),
    ...httpsOf(v),
    domains: v.domains,
    port: Number(v.port),
    fromPath: first,
    fromPaths: paths,
    match: v.match,
    toUrl: normalizeTargetUrl(v.toUrl),
  };
}

/** The form can save once the kind has what it needs; the server checks the rest. */
export function canSubmit(v: RuleValues): boolean {
  if (!v.name.trim()) return false;
  switch (v.kind) {
    case 'batches':
      return v.groupId > 0 || v.sourceUrl.trim() !== '';
    case 'custom': {
      const paths = cleanPaths(v.fromPaths.length > 0 ? v.fromPaths : [v.fromPath]);
      return paths.length > 0 && v.toUrl.trim() !== '';
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
