import { afterEach, describe, expect, it, vi } from 'vitest';
import { jobOutcome, listJobs, originHost, type Counts } from './migrations';

afterEach(() => vi.unstubAllGlobals());

function reply(body: unknown) {
  const f = vi.fn(() => Promise.resolve(new Response(JSON.stringify(body), { status: 200 })));
  vi.stubGlobal('fetch', f);
  return f;
}

const counts = (c: Partial<Counts>): Counts => ({
  read: 0,
  moved: 0,
  skipped: 0,
  adopted: 0,
  undone: 0,
  configChanged: 0,
  renamed: 0,
  protocolsAdded: 0,
  verified: 0,
  ...c,
});

describe('listJobs', () => {
  it('asks for every job when no filter is set', async () => {
    const f = reply({ jobs: [] });
    await expect(listJobs()).resolves.toEqual({ jobs: [] });
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/api\/v1\/migrations$/);
  });

  it('puts the filter in the query', async () => {
    const f = reply({ jobs: [] });
    await listJobs({ limit: 5, panel: 2, states: ['done', 'failed'], cursor: 'mvc694g0.ab' });
    const url = new URL((f.mock.calls.at(-1) as unknown as [string])[0]);
    expect(Object.fromEntries(url.searchParams)).toEqual({
      limit: '5',
      panel: '2',
      state: 'done,failed',
      cursor: 'mvc694g0.ab',
    });
  });
});

describe('jobOutcome', () => {
  it('counts moved users that are still there', () => {
    expect(jobOutcome({ state: 'done', counts: counts({ moved: 120, skipped: 4 }) })).toEqual({
      moved: 120,
      skipped: 4,
    });
    expect(jobOutcome({ state: 'failed', counts: counts({ moved: 50, undone: 80 }) })).toEqual({
      moved: 0,
      skipped: 0,
    });
  });

  it('has nothing to count before a run ends or after a rollback', () => {
    expect(jobOutcome({ state: 'checked', counts: null })).toBeNull();
    expect(jobOutcome({ state: 'rolled_back', counts: counts({ moved: 120 }) })).toBeNull();
  });
});

describe('originHost', () => {
  it('keeps the host and a port that is not the default', () => {
    expect(originHost('https://old.example.com:8443/panel')).toBe('old.example.com:8443');
    expect(originHost('https://old.example.com')).toBe('old.example.com');
  });

  it('keeps what it cannot read', () => {
    expect(originHost('not a url')).toBe('not a url');
  });
});
