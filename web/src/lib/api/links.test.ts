import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  getLinkSites,
  getUpload,
  newSitePath,
  originOf,
  pendingOf,
  sitePrefill,
  uploadTokens,
} from './links';

afterEach(() => vi.unstubAllGlobals());

describe('old link sites', () => {
  it('lists them with the ports that could not listen', async () => {
    const body = { sites: [{ id: 1, name: 'old-443' }], portErrors: {} };
    const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(body))));
    vi.stubGlobal('fetch', f);
    await expect(getLinkSites()).resolves.toEqual(body);
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/api\/v1\/legacy-sites$/);
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
    expect(path).toBe('/links/new?panel=2&kind=marzban&from=https%3A%2F%2Fold.example.com%3A8443');
    expect(sitePrefill(new URL(path, 'http://x').searchParams)).toEqual(p);
  });

  it('opens an empty form without a prefill', () => {
    expect(newSitePath({})).toBe('/links/new');
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
