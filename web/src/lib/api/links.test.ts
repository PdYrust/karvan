import { afterEach, describe, expect, it, vi } from 'vitest';
import { getLinkSites, newSitePath, sitePrefill } from './links';

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
