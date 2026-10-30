import { afterEach, describe, expect, it, vi } from 'vitest';
import { getHttps } from './https';

afterEach(() => vi.unstubAllGlobals());

describe('the HTTPS settings', () => {
  it('read a null list of shared ports as empty', async () => {
    const body = { tls: 'off', domainId: 0, shared: false, port: 8443, pending: null };
    const f = vi.fn(() =>
      Promise.resolve(new Response(JSON.stringify({ ...body, sharedPorts: null }))),
    );
    vi.stubGlobal('fetch', f);
    await expect(getHttps()).resolves.toEqual({ ...body, sharedPorts: [] });
    expect((f.mock.calls.at(-1) as unknown as [string])[0]).toMatch(/api\/v1\/settings\/https$/);
  });
});
