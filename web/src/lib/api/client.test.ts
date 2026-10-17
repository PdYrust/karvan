import { afterEach, describe, expect, it, vi } from 'vitest';
import { api, ApiError, offline, setCsrfToken } from './client';
import { getOverview } from './overview';
import { getSession, signIn, signOut } from './session';

function reply(status: number, body?: unknown, raw?: string) {
  const fetchMock = vi.fn(() =>
    Promise.resolve(
      new Response(raw ?? (body === undefined ? null : JSON.stringify(body)), { status }),
    ),
  );
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function lastCall(fetchMock: ReturnType<typeof reply>) {
  const call = fetchMock.mock.calls.at(-1) as unknown as [string, RequestInit];
  return { url: call[0], init: call[1], headers: call[1].headers as Record<string, string> };
}

afterEach(() => vi.unstubAllGlobals());

describe('api', () => {
  it('resolves paths under the base URL', async () => {
    const f = reply(200, { ok: true });
    await api('GET', 'overview');
    expect(lastCall(f).url).toBe(new URL('api/v1/overview', document.baseURI).toString());
  });

  it('sends JSON and the CSRF token on writes, not on reads', async () => {
    const f = reply(200, {});
    setCsrfToken('tok');
    await api('POST', 'session', { a: 1 });
    expect(lastCall(f).headers['X-CSRF-Token']).toBe('tok');
    expect(lastCall(f).headers['Content-Type']).toBe('application/json');
    expect(lastCall(f).init.body).toBe('{"a":1}');
    await api('GET', 'session');
    expect(lastCall(f).headers['X-CSRF-Token']).toBeUndefined();
  });

  it('turns error bodies into ApiError', async () => {
    reply(429, { error: { code: 'locked', retryAfter: 90 } });
    const err = await api('POST', 'session', {}).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 429, code: 'locked', retryAfter: 90 });
  });

  it('falls back to internal for non-JSON errors', async () => {
    reply(502, undefined, '<html>bad gateway</html>');
    await expect(api('GET', 'overview')).rejects.toMatchObject({ status: 502, code: 'internal' });
  });

  it('reports network failures as offline', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.reject(new TypeError('failed'))),
    );
    await expect(api('GET', 'overview')).rejects.toMatchObject({ status: 0, code: offline });
  });

  it('returns undefined for 204', async () => {
    reply(204);
    await expect(api('DELETE', 'session')).resolves.toBeUndefined();
  });

  it('keeps the extra fields of an error', async () => {
    reply(400, { error: { code: 'confirm_required', confirm: 'turn-off-safe-mode' } });
    const err = (await api('PUT', 'settings/safe-mode', {}).catch((e: unknown) => e)) as ApiError;
    expect(err.detail).toEqual({ confirm: 'turn-off-safe-mode' });
  });

  it('sends If-Match, the background marker and raw bodies', async () => {
    const f = reply(200, {});
    await api('PUT', 'migrations/a/plan', { x: 1 }, { ifMatch: 7 });
    expect(lastCall(f).headers['If-Match']).toBe('"7"');
    await api('GET', 'analytics/server', undefined, { background: true });
    expect(lastCall(f).headers['X-Karvan-Background']).toBe('1');
    expect(lastCall(f).headers['If-Match']).toBeUndefined();
    await api('POST', 'legacy-sites/1/tokens', undefined, { raw: 'a\nb' });
    expect(lastCall(f).headers['Content-Type']).toBe('text/plain; charset=utf-8');
    expect(lastCall(f).init.body).toBe('a\nb');
  });

  it('lets an aborted request stay an abort', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.reject(new DOMException('stop', 'AbortError'))),
    );
    await expect(api('GET', 'overview')).rejects.toMatchObject({ name: 'AbortError' });
  });
});

describe('session calls', () => {
  it('keeps the CSRF token from session responses', async () => {
    reply(200, { authenticated: false, csrfToken: 'pre' });
    await getSession();
    let f = reply(200, { authenticated: true, csrfToken: 'after', operator: { username: 'o' } });
    await signIn({ username: 'o', password: 'p' });
    expect(lastCall(f).headers['X-CSRF-Token']).toBe('pre');
    expect(lastCall(f).init.body).toBe('{"username":"o","password":"p","totpCode":""}');
    f = reply(204);
    await signOut();
    expect(lastCall(f).headers['X-CSRF-Token']).toBe('after');
  });

  it('fetches the overview', async () => {
    reply(200, { version: 'dev', panels: [] });
    await expect(getOverview()).resolves.toMatchObject({ version: 'dev' });
  });
});
