import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  cancelJob,
  compareJob,
  createJob,
  getJob,
  getJobRow,
  jobOutcome,
  listJobs,
  originHost,
  type Counts,
} from './migrations';

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

describe('getJobRow', () => {
  it("reads one job's row by its id, and null for an id Karvan does not know", async () => {
    const f = reply({ jobs: [{ id: 'ab', state: 'running' }] });
    await expect(getJobRow('ab')).resolves.toMatchObject({ id: 'ab', state: 'running' });
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/migrations\?id=ab$/);
    reply({ jobs: [] });
    await expect(getJobRow('cd')).resolves.toBeNull();
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

describe('job calls', () => {
  it('creates a job with the fields given and nothing else', async () => {
    const f = reply({ id: 'abc', state: 'created' });
    await createJob({
      panelId: 2,
      sourceUrl: 'http://127.0.0.1:18000',
      username: 'root',
      password: 'x',
    });
    const [url, init] = f.mock.calls.at(-1) as unknown as [string, RequestInit];
    expect(url).toMatch(/api\/v1\/migrations$/);
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      panelId: 2,
      sourceUrl: 'http://127.0.0.1:18000',
      username: 'root',
      password: 'x',
    });
  });

  it('starts a compare and cancels on the job path', async () => {
    const f = reply({ id: 'a/b' });
    await compareJob('a/b');
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/migrations\/a%2Fb\/compare$/);
    await cancelJob('abc');
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/migrations\/abc\/cancel$/);
    await getJob('abc');
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/migrations\/abc$/);
  });
});
