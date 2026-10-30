import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  deleteRedirect,
  getLinkSites,
  getRedirects,
  getUpload,
  newBatchRulePath,
  newSitePath,
  originOf,
  pendingOf,
  rulePath,
  saveRedirect,
  sitePrefill,
  uploadTokens,
} from './rules';

afterEach(() => vi.unstubAllGlobals());

describe('old link sites', () => {
  it('lists them with the ports that could not listen', async () => {
    const site = {
      id: 1,
      name: 'old-443',
      batches: [],
      domains: [],
      certificate: null,
      warnings: [],
    };
    const body = { sites: [site], portErrors: {} };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(body))));
    vi.stubGlobal('fetch', f);
    await expect(getLinkSites()).resolves.toEqual(body);
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/api\/v1\/legacy-sites$/);
  });

  it('reads null lists as empty', async () => {
    const facts = { domains: null, issuer: '', warnings: null };
    const site = {
      id: 1,
      batches: null,
      domains: null,
      cert: facts,
      certificate: { id: 2, facts },
      warnings: null,
    };
    const f = vi.fn(() =>
      Promise.resolve(new Response(JSON.stringify({ sites: [site], portErrors: null }))),
    );
    vi.stubGlobal('fetch', f);
    const empty = { domains: [], issuer: '', warnings: [] };
    await expect(getLinkSites()).resolves.toEqual({
      sites: [
        {
          id: 1,
          batches: [],
          domains: [],
          cert: empty,
          certificate: { id: 2, facts: empty },
          warnings: [],
        },
      ],
      portErrors: {},
    });
  });
});
describe('redirect rules', () => {
  it('read null lists as empty', async () => {
    const r = { id: 1, domains: null, certificate: null, warnings: null };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify({ redirects: [r] }))));
    vi.stubGlobal('fetch', f);
    await expect(getRedirects()).resolves.toEqual({
      redirects: [{ id: 1, domains: [], certificate: null, warnings: [] }],
    });
    f.mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ redirects: null }))));
    await expect(getRedirects()).resolves.toEqual({ redirects: [] });
  });
  it('are read, added, changed and deleted at their own route', async () => {
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify({ redirects: [] }))));
    vi.stubGlobal('fetch', f);
    await getRedirects();
    const input = {
      name: 'r',
      domains: [],
      port: 80,
      fromPath: '/a',
      match: 'exact' as const,
      toUrl: 'https://n',
    };
    await saveRedirect(input);
    await saveRedirect(input, 4);
    f.mockResolvedValueOnce(new Response(null, { status: 204 }));
    await deleteRedirect(4);
    const calls = (f.mock.calls as unknown as [string, RequestInit][]).map(
      ([url, init]) => `${init.method ?? 'GET'} ${url.replace(/.*api\/v1\//, '')}`,
    );
    expect(calls).toEqual([
      'GET link-redirects',
      'POST link-redirects',
      'PATCH link-redirects/4',
      'DELETE link-redirects/4',
    ]);
  });

  it('are changed under their own path, panel rules under theirs', () => {
    expect(rulePath({ kind: 'redirect', id: 4 })).toBe('/rules/redirects/4');
    expect(rulePath({ kind: 'panel', id: 4 })).toBe('/rules/4');
    expect(newBatchRulePath(12)).toBe('/rules/new?batch=12');
  });
});

describe('a new site prefilled from a migration', () => {
  it('carries the panel, the kind and the old address there and back', () => {
    const p = {
      panelId: 2,
      sourceKind: 'marzban',
      sourceUrl: 'https://old.example.com:8443',
    } as const;
    const path = newSitePath(p);
    expect(path).toBe('/rules/new?panel=2&kind=marzban&from=https%3A%2F%2Fold.example.com%3A8443');
    expect(sitePrefill(new URL(path, 'http://x').searchParams)).toEqual(p);
  });

  it('opens an empty form without a prefill', () => {
    expect(newSitePath({})).toBe('/rules/new');
    expect(sitePrefill(new URLSearchParams())).toEqual({});
  });

  it('drops values the form cannot use', () => {
    const q = new URLSearchParams({ panel: '-1', kind: 'xui', from: 'javascript:alert(1)' });
    expect(sitePrefill(q)).toEqual({});
    expect(sitePrefill(new URLSearchParams({ panel: '1.5', from: 'old panel' }))).toEqual({});
  });
});

describe('a token list', () => {
  it('goes up as text and is followed in the background', async () => {
    const progress = { state: 'running', found: 3, checked: 0 };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(progress))));
    vi.stubGlobal('fetch', f);
    await uploadTokens(4, new Blob(['a,b']));
    const [url, init] = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(url).toMatch(/legacy-sites\/4\/tokens$/);
    expect(init.method).toBe('POST');
    expect((init.headers as Record<string, string>)['Content-Type']).toMatch(/^text\/plain/);
    await expect(getUpload(4)).resolves.toEqual(progress);
    const [, poll] = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect((poll.headers as Record<string, string>)['X-Karvan-Background']).toBe('1');
  });
});

describe('site numbers', () => {
  it('counts pending users as moved, not proven and not closed', () => {
    expect(pendingOf({ moved: 120, verified: 34, closed: 6 })).toBe(80);
    expect(pendingOf({ moved: 2, verified: 2, closed: 1 })).toBe(0);
  });
});

describe('an old panel address', () => {
  it('reads like the ledger keeps it', () => {
    expect(originOf('https://Old.Example.com:443/Panel/')).toBe('https://old.example.com/panel');
    expect(originOf('http://old.example.com:8000')).toBe('http://old.example.com:8000');
    expect(originOf(' Not an address ')).toBe('not an address');
  });
});
