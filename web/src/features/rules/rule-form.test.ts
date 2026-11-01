import { describe, expect, it } from 'vitest';
import type { BatchGroup } from '../../lib/api/batches';
import { ApiError } from '../../lib/api/client';
import type { Domain } from '../../lib/api/domains';
import type { LinkSite, Redirect } from '../../lib/api/rules';
import {
  canContinueDomain,
  canSubmit,
  cleanPaths,
  domainState,
  groupDefaults,
  httpsAllowed,
  isPlainHttp,
  newRuleValues,
  normalizeTargetUrl,
  redirectInputOf,
  refusedField,
  ruleKinds,
  rulePrefill,
  siteInputOf,
  stepForRefused,
  stepsForRule,
  unregistered,
  valuesOfRedirect,
  valuesOfSite,
  withDomains,
  withGroup,
  type RuleValues,
} from './rule-form';

const facts = { domains: ['a.example.com'], issuer: 'R10', warnings: [] };
function domain(id: number, name: string, extra: Partial<Domain> = {}): Domain {
  return {
    id,
    name,
    certificate: { id: 9, name: 'a-cert', origin: '', facts },
    certificates: [],
    pins: [],
    acme: null,
    rules: [],
    warnings: [],
    ...extra,
  };
}
const domains = [
  domain(1, 'a.example.com'),
  domain(2, 'b.example.com', { certificate: null }),
  domain(3, 'c.example.com', {
    certificate: null,
    acme: {
      method: 'dns-01',
      nextRenewal: null,
      tries: 0,
      lastError: null,
      order: {
        id: 4,
        method: 'dns-01',
        renewal: false,
        state: 'waiting_dns',
        dns: null,
        error: null,
        createdAt: '',
        updatedAt: '',
        expiresAt: null,
      },
    },
  }),
];

function group(id: number, extra: Partial<BatchGroup> = {}): BatchGroup {
  return {
    id,
    latestId: id + 20,
    panel: { id: 2, name: 'new', kind: 'pasarguard' },
    source: { kind: 'marzban', origin: 'https://old.example.com:8443' },
    counts: { runs: 3, active: 3, inactive: 0, users: 30, verified: 0, closed: 0 },
    oldLinks: { path: 'sub', hosts: ['old.example.com', 'a.example.com'], port: 8443 },
    rules: [],
    firstAt: '',
    lastAt: '',
    runs: [],
    ...extra,
  };
}

const site: LinkSite = {
  id: 3,
  name: 'old-443',
  mode: 'batches',
  batches: [{ id: 7, name: 'b', state: 'active' }],
  batchGroup: 4,
  panelId: 2,
  sourceKind: 'pasarguard',
  sourceUrl: 'https://old.example.com:8443',
  oldPath: 'sub',
  domains: ['a.example.com'],
  port: 443,
  lazy: true,
  plainHttpOk: false,
  https: true,
  certificate: null,
  warnings: [],
};

const values = (patch: Partial<RuleValues>): RuleValues => ({ ...newRuleValues({}, 1), ...patch });

describe('a new rule', () => {
  it('offers only from an old panel and custom', () => {
    expect(ruleKinds).toEqual(['batches', 'custom']);
  });

  it('starts from batches, on 443 with lazy checks on and sub as initial custom path', () => {
    const v = newRuleValues({}, 1);
    expect(v).toMatchObject({
      kind: 'batches',
      panelId: 1,
      port: '443',
      lazy: true,
      fromPaths: ['sub'],
    });
    expect(v.domains).toEqual([]);
  });

  it('starts from the old panel of the batch a finished migration names', () => {
    const p = rulePrefill(new URLSearchParams('batch=12'));
    expect(p.batchId).toBe(12);
    expect(newRuleValues(p, 1)).toMatchObject({ kind: 'batches', groupId: 0 });
    expect(newRuleValues({ ...p, groupId: 4 }, 1)).toMatchObject({ kind: 'batches', groupId: 4 });
  });

  it('takes custom or redirect from the address and drops unknown kinds', () => {
    expect(newRuleValues(rulePrefill(new URLSearchParams('kind=custom')), 1).kind).toBe('custom');
    expect(newRuleValues(rulePrefill(new URLSearchParams('kind=redirect')), 1).kind).toBe('custom');
    expect(newRuleValues(rulePrefill(new URLSearchParams('kind=batches')), 1).kind).toBe('batches');
    expect(rulePrefill(new URLSearchParams('kind=other&batch=x')).kind).toBeUndefined();
    expect(rulePrefill(new URLSearchParams('batch=-1')).batchId).toBeUndefined();
  });
});

describe('rule wizard steps', () => {
  it('walks kind, domain, https and details for a new rule, and skips kind on edit', () => {
    expect(stepsForRule(true)).toEqual(['kind', 'domain', 'https', 'details']);
    expect(stepsForRule(false)).toEqual(['domain', 'https', 'details']);
  });

  it('continues from the domain step once named and, for batches, an old panel is picked', () => {
    expect(canContinueDomain(values({ kind: 'batches', name: 'r', groupId: 0 }))).toBe(false);
    expect(canContinueDomain(values({ kind: 'batches', name: 'r', groupId: 4 }))).toBe(true);
    expect(canContinueDomain(values({ kind: 'custom', name: '' }))).toBe(false);
    expect(canContinueDomain(values({ kind: 'custom', name: 'r' }))).toBe(true);
  });

  it('maps a refused field to the step that holds it', () => {
    expect(stepForRefused(undefined)).toBeUndefined();
    expect(stepForRefused('name')).toBe('domain');
    expect(stepForRefused('domains')).toBe('domain');
    expect(stepForRefused('batchIds')).toBe('domain');
    expect(stepForRefused('https')).toBe('https');
    expect(stepForRefused('port')).toBe('details');
    expect(stepForRefused('fromPath')).toBe('details');
    expect(stepForRefused('toUrl')).toBe('details');
  });
});

describe('a saved rule', () => {
  it('opens a rule of batches with its old panel and domains picked', () => {
    const v = valuesOfSite(site);
    expect(v).toMatchObject({ kind: 'batches', groupId: 4, domains: ['a.example.com'] });
    expect(valuesOfSite({ ...site, domains: null }).domains).toEqual([]);
    expect(valuesOfSite({ ...site, mode: 'batches', sourceIp: '203.0.113.9' }).sourceIp).toBe('');
    expect(valuesOfSite({ ...site, mode: 'manual', sourceIp: '203.0.113.9' }).sourceIp).toBe(
      '203.0.113.9',
    );
  });

  it('sends a changed old path and port of a rule of batches', () => {
    const v = valuesOfSite(site);
    expect(siteInputOf({ ...v, oldPath: 'dl', port: '4433' }, group(4))).toMatchObject({
      mode: 'batches',
      oldPath: 'dl',
      port: 4433,
    });
  });

  it('opens a custom redirect with all its allowed paths and target', () => {
    const r: Redirect = {
      id: 5,
      name: 'r',
      domains: null,
      port: 80,
      fromPath: '/sub',
      fromPaths: ['/sub', '/dashboard'],
      match: 'prefix',
      toUrl: 'http://127.0.0.1:8080',
      https: false,
      certificate: null,
      warnings: [],
    };
    expect(valuesOfRedirect(r)).toMatchObject({
      kind: 'custom',
      domains: [],
      port: '80',
      fromPath: '/sub',
      fromPaths: ['sub', 'dashboard'],
      match: 'prefix',
      toUrl: 'http://127.0.0.1:8080',
      https: false,
    });
  });
});

describe('registered domains', () => {
  it('shows each with its certificate state', () => {
    expect(domains.map(domainState)).toEqual(['https', 'plain', 'pending']);
  });

  it('allows HTTPS only when every picked domain has a certificate', () => {
    expect(httpsAllowed(['a.example.com'], domains)).toBe(true);
    expect(httpsAllowed(['a.example.com', 'b.example.com'], domains)).toBe(false);
    expect(httpsAllowed(['c.example.com'], domains)).toBe(false);
    expect(httpsAllowed(['gone.example.com'], domains)).toBe(false);
    expect(httpsAllowed([], domains)).toBe(false);
  });

  it('turns HTTPS on with domains that all have one, and off when one has none', () => {
    const v = values({ https: false });
    expect(withDomains(v, ['a.example.com'], domains).https).toBe(true);
    expect(withDomains(v, ['a.example.com', 'b.example.com'], domains).https).toBe(false);
  });

  it('names hosts that are not registered yet', () => {
    expect(unregistered(['old.example.com', 'a.example.com'], domains)).toEqual([
      'old.example.com',
    ]);
  });
});

describe('the old panel picker', () => {
  const noFacts = group(10, { oldLinks: { path: '', hosts: [], port: 0 } });

  it('fills in the registered old hosts, the old port and the old path', () => {
    expect(groupDefaults(group(4), domains)).toEqual({
      domains: ['a.example.com'],
      port: '8443',
      hosts: ['a.example.com', 'old.example.com'],
      oldPath: 'sub',
    });
    expect(groupDefaults(noFacts, domains)).toMatchObject({ port: '', oldPath: '', hosts: [] });
    expect(groupDefaults(undefined, domains)).toMatchObject({ port: '', domains: [] });
  });

  it('picks the registered old hosts with HTTPS when they all have certificates', () => {
    const v = withGroup(values({ oldPath: 'typed' }), 4, [group(4), group(5)], domains);
    expect(v).toMatchObject({
      groupId: 4,
      domains: ['a.example.com'],
      port: '8443',
      oldPath: 'sub',
      https: true,
    });
    expect(
      withGroup(values({ port: '2053', oldPath: 'typed' }), 10, [noFacts], domains),
    ).toMatchObject({ port: '2053', oldPath: 'typed' });
  });

  it('names the old panel by its newest batch and sends its old path and port as typed', () => {
    const v = values({
      groupId: 4,
      domains: ['a.example.com'],
      oldPath: ' typed ',
      port: '2096',
      https: true,
    });
    expect(siteInputOf(v, group(4))).toEqual({
      name: '',
      mode: 'batches',
      batchIds: [24],
      domains: ['a.example.com'],
      port: 2096,
      oldPath: 'typed',
      lazy: true,
      plainHttpOk: false,
      https: true,
    });
    expect(siteInputOf({ ...v, sourceIp: ' 203.0.113.9 ' }, noFacts)).toMatchObject({
      batchIds: [30],
      oldPath: 'typed',
      sourceIp: '203.0.113.9',
    });
  });

  it('accepts a plain http old panel only when turned on', () => {
    expect(isPlainHttp(' HTTP://old')).toBe(true);
    expect(isPlainHttp('https://old')).toBe(false);
    const plain = group(4, { source: { kind: 'marzban', origin: 'http://old' } });
    const v = values({ groupId: 4, plainHttpOk: true });
    expect(siteInputOf(v, plain).plainHttpOk).toBe(true);
    expect(siteInputOf(v, group(4)).plainHttpOk).toBe(false);
  });
});

describe('a custom rule', () => {
  it('cleans and deduplicates allowed source paths', () => {
    expect(cleanPaths([' sub ', '/dashboard/', 'sub', '', ' /api/v1 '])).toEqual([
      '/sub',
      '/dashboard',
      '/api/v1',
    ]);
  });

  it('normalizes bare domains and local IPs with ports into target URLs', () => {
    expect(normalizeTargetUrl('new.example.com:2096')).toBe('https://new.example.com:2096');
    expect(normalizeTargetUrl('127.0.0.1:8080')).toBe('http://127.0.0.1:8080');
    expect(normalizeTargetUrl('192.168.1.10:2096')).toBe('http://192.168.1.10:2096');
    expect(normalizeTargetUrl('https://new.example.com:8443')).toBe('https://new.example.com:8443');
    expect(normalizeTargetUrl('http://10.0.0.2:3000')).toBe('http://10.0.0.2:3000');
  });

  it('sends its allowed paths, port, domains and target URL', () => {
    const v = values({
      kind: 'custom',
      name: ' custom-rule ',
      domains: ['a.example.com'],
      port: '8443',
      fromPaths: [' sub ', '/dashboard/'],
      match: 'prefix',
      toUrl: ' 127.0.0.1:8080 ',
      https: true,
    });
    expect(redirectInputOf(v)).toEqual({
      name: 'custom-rule',
      domains: ['a.example.com'],
      port: 8443,
      fromPath: '/sub',
      fromPaths: ['/sub', '/dashboard'],
      match: 'prefix',
      toUrl: 'http://127.0.0.1:8080',
      https: true,
    });
  });
});

describe('the save button', () => {
  it('waits for what each kind needs', () => {
    expect(canSubmit(values({ name: 'x' }))).toBe(false);
    expect(canSubmit(values({ name: 'x', groupId: 7 }))).toBe(true);
    const custom = values({
      kind: 'custom',
      name: 'x',
      fromPaths: ['sub', 'dashboard'],
      toUrl: 'https://new.example.com:2096',
    });
    expect(canSubmit(custom)).toBe(true);
    expect(canSubmit({ ...custom, fromPaths: [' '] })).toBe(false);
    expect(canSubmit({ ...custom, toUrl: ' ' })).toBe(false);
    expect(canSubmit({ ...custom, name: '' })).toBe(false);
  });
});

describe('a refused save', () => {
  it('names the field of a panel rule or a redirect', () => {
    const siteErr = new ApiError(400, 'site_invalid', undefined, { field: 'batchIds' });
    expect(refusedField(siteErr)).toBe('batchIds');
    const redirectErr = new ApiError(400, 'redirect_invalid', undefined, { field: 'toUrl' });
    expect(refusedField(redirectErr)).toBe('toUrl');
    expect(
      refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'other' })),
    ).toBeUndefined();
    expect(refusedField(new ApiError(409, 'name_taken'))).toBeUndefined();
  });
});
