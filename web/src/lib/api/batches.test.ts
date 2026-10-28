import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  deleteBatch,
  getBatch,
  listBatches,
  listBatchUsers,
  reactivateBatch,
  renameBatch,
  showBatchLink,
} from './batches';

afterEach(() => vi.unstubAllGlobals());

function stubFetch(body: unknown = {}, status = 200) {
  const f = vi.fn(() =>
    Promise.resolve(
      status === 204 ? new Response(null, { status }) : new Response(JSON.stringify(body)),
    ),
  );
  vi.stubGlobal('fetch', f);
  return () => f.mock.calls.at(-1) as unknown as [string, RequestInit];
}

describe('the batch list', () => {
  it('reads every batch without filters', async () => {
    const last = stubFetch({ batches: [] });
    await expect(listBatches()).resolves.toEqual({ batches: [] });
    expect(last()[0]).toMatch(/api\/v1\/batches$/);
  });
  it('sends the panel, the state, the cursor and the page size', async () => {
    const last = stubFetch({ batches: [], next: '7' });
    await listBatches({ panel: 2, state: 'inactive', cursor: '9', limit: 20 });
    const url = new URL(last()[0]);
    expect(url.pathname).toMatch(/\/batches$/);
    expect(Object.fromEntries(url.searchParams)).toEqual({
      panel: '2',
      state: 'inactive',
      cursor: '9',
      limit: '20',
    });
  });
});

describe('one batch', () => {
  it('is read by its id', async () => {
    const last = stubFetch({ id: 3, name: 'root' });
    await expect(getBatch(3)).resolves.toMatchObject({ name: 'root' });
    expect(last()[0]).toMatch(/batches\/3$/);
  });
  it('lists its users with the search and the cursor', async () => {
    const last = stubFetch({ users: [] });
    await listBatchUsers(3, { q: ' ali ', cursor: '12', limit: 50 });
    const url = new URL(last()[0]);
    expect(url.pathname).toMatch(/batches\/3\/users$/);
    expect(Object.fromEntries(url.searchParams)).toEqual({ q: 'ali', cursor: '12', limit: '50' });
  });
  it('lists its users without an empty search', async () => {
    const last = stubFetch({ users: [] });
    await listBatchUsers(3, { q: '  ' });
    expect(last()[0]).toMatch(/batches\/3\/users$/);
  });
  it('is renamed with a PATCH of the name', async () => {
    const last = stubFetch({ id: 3, name: 'resellers' });
    await renameBatch(3, 'resellers');
    const [url, init] = last();
    expect(url).toMatch(/batches\/3$/);
    expect(init.method).toBe('PATCH');
    expect(JSON.parse(init.body as string)).toEqual({ name: 'resellers' });
  });
  it('is deleted with its name typed', async () => {
    const last = stubFetch(undefined, 204);
    await deleteBatch(3, 'root');
    const [url, init] = last();
    expect(url).toMatch(/batches\/3$/);
    expect(init.method).toBe('DELETE');
    expect(JSON.parse(init.body as string)).toEqual({ confirm: 'root' });
  });
  it('is made active again with a POST', async () => {
    const last = stubFetch({ id: 3, state: 'active' });
    await reactivateBatch(3);
    const [url, init] = last();
    expect(url).toMatch(/batches\/3\/reactivate$/);
    expect(init.method).toBe('POST');
  });
  it("reads a user's new link on request", async () => {
    const last = stubFetch({ url: 'https://new.example.com/sub/abc' });
    await expect(showBatchLink(3, 1542)).resolves.toBe('https://new.example.com/sub/abc');
    const [url, init] = last();
    expect(url).toMatch(/batches\/3\/users\/1542\/link$/);
    expect(init.method).toBe('POST');
  });
});
