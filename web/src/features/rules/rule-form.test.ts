import { describe, expect, it } from 'vitest';
import type { BatchGroup } from '../../lib/api/batches';
import { ApiError } from '../../lib/api/client';
import type { Domain } from '../../lib/api/domains';
import type { LinkSite, Redirect } from '../../lib/api/rules';
import {
  groupDefaults,
  canSubmit,
  domainState,
  httpsAllowed,
  isPlainHttp,
  newRuleValues,
  parseOldLink,
  redirectInputOf,
  refusedField,
  rulePrefill,
  siteInputOf,
  unregistered,
  valuesOfRedirect,
  valuesOfSite,
  withGroup,
  withDomains,
  withLink,
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
  mode: 'manual',
  batches: [],
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
  it('starts from batches, on 443 with lazy checks on', () => {
    const v = newRuleValues({}, 1);
    expect(v).toMatchObject({ kind: 'batches', panelId: 1, port: '443', lazy: true });
    expect(v.domains).toEqual([]);
  });

  it('starts from the old panel of the batch a finished migration names', () => {
    const p = rulePrefill(new URLSearchParams('batch=12'));
    expect(p.batchId).toBe(12);
    expect(newRuleValues(p, 1)).toMatchObject({ kind: 'batches', groupId: 0 });
    expect(newRuleValues({ ...p, groupId: 4 }, 1)).toMatchObject({ kind: 'batches', groupId: 4 });
  });

  it('starts manual when a migration fills in the old panel', () => {
    const p = rulePrefill(new URLSearchParams('panel=2&kind=marzban&from=https://old'));
    expect(newRuleValues(p, 1)).toMatchObject({
      kind: 'manual',
      panelId: 2,
      sourceUrl: 'https://old',
    });
  });

  it('takes a kind from the address and drops one it does not know', () => {
    expect(newRuleValues(rulePrefill(new URLSearchParams('kind=redirect')), 1).kind).toBe(
      'redirect',
    );
    expect(rulePrefill(new URLSearchParams('kind=other&batch=x')).kind).toBeUndefined();
    expect(rulePrefill(new URLSearchParams('batch=-1')).batchId).toBeUndefined();
  });
});

describe('a saved rule', () => {
  it('opens a manual site with its domains picked', () => {
    expect(valuesOfSite(site)).toMatchObject({ kind: 'manual', domains: ['a.example.com'] });
    expect(valuesOfSite({ ...site, domains: null }).domains).toEqual([]);
  });

  it('opens a rule of batches with its old panel picked', () => {
    const v = valuesOfSite({
      ...site,
      mode: 'batches',
      batches: [{ id: 7, name: 'b', state: 'active' }],
      batchGroup: 4,
    });
    expect(v).toMatchObject({ kind: 'batches', groupId: 4 });
    // The batches gave the IP; sending it back would be refused.
    expect(valuesOfSite({ ...site, mode: 'batches', sourceIp: '203.0.113.9' }).sourceIp).toBe('');
    expect(valuesOfSite({ ...site, sourceIp: '203.0.113.9' }).sourceIp).toBe('203.0.113.9');
  });

  it('sends a changed old path and port of a rule of batches', () => {
    const v = valuesOfSite({ ...site, mode: 'batches', batchGroup: 4 });
    expect(siteInputOf({ ...v, oldPath: 'dl', port: '4433' }, group(4))).toMatchObject({
      mode: 'batches',
      oldPath: 'dl',
      port: 4433,
    });
  });

  it('opens a redirect with its path and target', () => {
    const r: Redirect = {
      id: 5,
      name: 'r',
      domains: null,
      port: 80,
      fromPath: '/sub/abc',
      match: 'prefix',
      toUrl: 'https://new.example.com/sub/xyz',
      https: false,
      certificate: null,
      warnings: [],
    };
    expect(valuesOfRedirect(r)).toMatchObject({
      kind: 'redirect',
      domains: [],
      port: '80',
      fromPath: '/sub/abc',
      match: 'prefix',
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
    const plain = group(4, { source: { kind: 'marzban', origin: 'http://old' } });
    const v = values({ groupId: 4, plainHttpOk: true });
    expect(siteInputOf(v, plain).plainHttpOk).toBe(true);
    expect(siteInputOf(v, group(4)).plainHttpOk).toBe(false);
  });
});

describe('a manual rule', () => {
  it('sends the old panel, the domains and HTTPS', () => {
    const input = siteInputOf(valuesOfSite({ ...site, port: 8443 }), undefined);
    expect(input).toMatchObject({
      mode: 'manual',
      panelId: 2,
      sourceUrl: 'https://old.example.com:8443',
      domains: ['a.example.com'],
      port: 8443,
      https: true,
    });
    expect(input).not.toHaveProperty('batchIds');
  });

  it('leaves HTTPS to the port when no domain is picked', () => {
    expect(siteInputOf(values({ kind: 'manual', domains: [] }), undefined)).not.toHaveProperty(
      'https',
    );
  });

  it('accepts a plain http old panel only for an http address', () => {
    expect(isPlainHttp(' HTTP://old')).toBe(true);
    expect(isPlainHttp('https://old')).toBe(false);
    const v = values({ kind: 'manual', plainHttpOk: true, sourceUrl: 'https://old' });
    expect(siteInputOf(v, undefined).plainHttpOk).toBe(false);
    expect(siteInputOf({ ...v, sourceUrl: 'http://old' }, undefined).plainHttpOk).toBe(true);
  });
});

describe('the single link parser', () => {
  it('reads the host, the port and the path of an old link', () => {
    expect(parseOldLink(' https://Old.Example.com:8443/sub/abcDEF/?x=1#y ')).toEqual({
      host: 'old.example.com',
      port: 8443,
      path: '/sub/abcDEF',
    });
    expect(parseOldLink('http://203.0.113.9/sub/abc')).toEqual({
      host: '203.0.113.9',
      port: 80,
      path: '/sub/abc',
    });
    expect(parseOldLink('https://[2001:db8::1]/s/t')?.host).toBe('2001:db8::1');
    expect(parseOldLink('https://old.example.com/sub')?.port).toBe(443);
  });

  it('picks the link host as the domain once it is registered', () => {
    const v = withLink(values({ kind: 'link' }), 'https://a.example.com/sub/t', domains);
    expect(v).toMatchObject({ domains: ['a.example.com'], https: true });
    const other = withLink(v, 'https://old.example.com/sub/t', domains);
    expect(other).toMatchObject({
      link: 'https://old.example.com/sub/t',
      domains: [],
      https: false,
    });
  });

  it('reads nothing from a value that is not an old link', () => {
    expect(parseOldLink('')).toBeUndefined();
    expect(parseOldLink('old.example.com/sub/abc')).toBeUndefined();
    expect(parseOldLink('ftp://old.example.com/sub/abc')).toBeUndefined();
    expect(parseOldLink('https://old.example.com/')).toBeUndefined();
  });
});

describe('a redirect', () => {
  it('sends its path, how it matches and its target', () => {
    const v = values({
      kind: 'redirect',
      name: ' r ',
      domains: ['a.example.com'],
      port: '80',
      fromPath: ' /go ',
      match: 'exact',
      toUrl: ' https://new.example.com/x ',
      https: true,
    });
    expect(redirectInputOf(v)).toEqual({
      name: 'r',
      domains: ['a.example.com'],
      port: 80,
      fromPath: '/go',
      match: 'exact',
      toUrl: 'https://new.example.com/x',
      https: true,
    });
  });

  it('saves a single link as a redirect of its path and below', () => {
    const v = values({
      kind: 'link',
      name: 'one',
      link: 'https://a.example.com:2096/sub/abc',
      toUrl: 'https://new.example.com/sub/xyz',
      https: true,
    });
    expect(redirectInputOf(v)).toEqual({
      name: 'one',
      domains: ['a.example.com'],
      port: 2096,
      fromPath: '/sub/abc',
      match: 'prefix',
      toUrl: 'https://new.example.com/sub/xyz',
      https: true,
    });
  });
});

describe('the save button', () => {
  it('waits for what each kind needs', () => {
    expect(canSubmit(values({ name: 'x' }), domains)).toBe(false);
    expect(canSubmit(values({ name: 'x', groupId: 7 }), domains)).toBe(true);
    expect(canSubmit(values({ kind: 'manual', name: 'x', sourceUrl: 'https://o' }), domains)).toBe(
      true,
    );
    const redirect = values({ kind: 'redirect', name: 'x', fromPath: '/a', toUrl: 'https://n' });
    expect(canSubmit(redirect, domains)).toBe(true);
    expect(canSubmit({ ...redirect, toUrl: ' ' }, domains)).toBe(false);
    expect(canSubmit({ ...redirect, name: '' }, domains)).toBe(false);
  });

  it('waits for a single link whose host is registered', () => {
    const v = values({ kind: 'link', name: 'x', toUrl: 'https://n' });
    expect(canSubmit({ ...v, link: 'https://a.example.com/sub/t' }, domains)).toBe(true);
    expect(canSubmit({ ...v, link: 'https://old.example.com/sub/t' }, domains)).toBe(false);
    expect(canSubmit({ ...v, link: 'not a link' }, domains)).toBe(false);
  });
});

describe('a refused save', () => {
  it('names the field of a panel rule or a redirect', () => {
    const site = new ApiError(400, 'site_invalid', undefined, { field: 'batchIds' });
    expect(refusedField(site)).toBe('batchIds');
    const redirect = new ApiError(400, 'redirect_invalid', undefined, { field: 'toUrl' });
    expect(refusedField(redirect)).toBe('toUrl');
    expect(
      refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'other' })),
    ).toBeUndefined();
    expect(refusedField(new ApiError(409, 'name_taken'))).toBeUndefined();
  });
});
