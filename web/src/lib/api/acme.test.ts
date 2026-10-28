import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  cancelAcme,
  checkAcme,
  getAcme,
  getAcmeSettings,
  isRunning,
  putAcmeSettings,
  startAcme,
  type AcmeOrder,
} from './acme';

afterEach(() => vi.unstubAllGlobals());

function stubFetch(body: unknown = {}, status = 200) {
  const f = vi.fn(() =>
    Promise.resolve(
      status === 204
        ? new Response(null, { status })
        : new Response(JSON.stringify(body), { status }),
    ),
  );
  vi.stubGlobal('fetch', f);
  return () => f.mock.calls.at(-1) as unknown as [string, RequestInit];
}

const sent = (init: RequestInit) => JSON.parse(init.body as string) as unknown;

describe("Let's Encrypt", () => {
  it('reads a domain order in the background, so polling does not extend the session', async () => {
    const last = stubFetch({ order: null, renewal: null });
    await expect(getAcme(3)).resolves.toEqual({ order: null, renewal: null });
    const [url, init] = last();
    expect(url).toMatch(/domains\/3\/acme$/);
    expect(new Headers(init.headers).get('X-Karvan-Background')).toBe('1');
  });

  it('starts, checks and cancels an order', async () => {
    let last = stubFetch({ id: 1, state: 'starting' }, 202);
    await startAcme(3, 'dns-01', true);
    expect(last()[1].method).toBe('POST');
    expect(sent(last()[1])).toEqual({ method: 'dns-01', agreeTos: true });
    await checkAcme(3);
    expect(last()[0]).toMatch(/domains\/3\/acme\/check$/);
    expect(sent(last()[1])).toEqual({ force: false });
    last = stubFetch(undefined, 204);
    await cancelAcme(3);
    expect(last()[1].method).toBe('DELETE');
  });

  it('reads and saves the settings', async () => {
    const view = {
      staging: true,
      email: '',
      keepPort80: false,
      testDirectory: false,
      hasAccount: true,
    };
    const last = stubFetch(view);
    await expect(getAcmeSettings()).resolves.toEqual(view);
    await putAcmeSettings({ staging: false, email: 'ops@example.com', keepPort80: true });
    expect(last()[0]).toMatch(/settings\/acme$/);
    expect(last()[1].method).toBe('PUT');
    expect(sent(last()[1])).toEqual({ staging: false, email: 'ops@example.com', keepPort80: true });
  });

  it('knows which orders still run', () => {
    const o = (state: AcmeOrder['state']) => ({ state }) as AcmeOrder;
    expect(
      ['starting', 'waiting_dns', 'validating', 'issuing'].map((s) =>
        isRunning(o(s as AcmeOrder['state'])),
      ),
    ).toEqual([true, true, true, true]);
    expect(isRunning(o('done'))).toBe(false);
    expect(isRunning(o('failed'))).toBe(false);
    expect(isRunning(null)).toBe(false);
  });
});
