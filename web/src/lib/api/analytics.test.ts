import { afterEach, describe, expect, it, vi } from 'vitest';
import { getPanelStats, getPanelsStats, getServerStats } from './analytics';

afterEach(() => vi.unstubAllGlobals());

function stub() {
  const f = vi.fn(() => Promise.resolve(new Response('{}')));
  vi.stubGlobal('fetch', f);
  return () => {
    const [url, init] = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    return { url: new URL(url), headers: init.headers as Record<string, string> };
  };
}

describe('analytics calls', () => {
  it('poll as background reads, so the session can still time out', async () => {
    const last = stub();
    await getServerStats('live', 42);
    const live = last();
    expect(live.url.pathname).toMatch(/\/api\/v1\/analytics\/server$/);
    expect(live.url.searchParams.get('range')).toBe('live');
    expect(live.url.searchParams.get('after')).toBe('42');
    expect(live.headers['X-Karvan-Background']).toBe('1');

    await getServerStats('24h');
    expect(last().url.searchParams.has('after')).toBe(false);
  });

  it('ask for panels by id and range', async () => {
    const last = stub();
    await getPanelStats(3, '7d');
    expect(last().url.toString()).toMatch(/\/analytics\/panels\/3\?range=7d$/);
    await getPanelsStats('30d');
    const all = last();
    expect(all.url.toString()).toMatch(/\/analytics\/panels\?range=30d$/);
    expect(all.headers['X-Karvan-Background']).toBe('1');
  });
});
