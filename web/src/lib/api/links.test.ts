import { afterEach, describe, expect, it, vi } from 'vitest';
import { getLinkSites } from './links';

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
