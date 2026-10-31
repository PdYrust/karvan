import { afterEach, describe, expect, it, vi } from 'vitest';
import { firstCheck } from './first-check';

afterEach(() => vi.unstubAllGlobals());

function answer(body: unknown, status = 200) {
  vi.stubGlobal(
    'fetch',
    vi.fn(() => Promise.resolve(new Response(JSON.stringify(body), { status }))),
  );
}

describe('the first session check', () => {
  it('goes in on a live session, as the installed app opens', async () => {
    answer({ authenticated: true, csrfToken: 't' });
    await expect(firstCheck()).resolves.toEqual({ signedIn: true, error: null });
  });

  it('shows the form, with nothing to say, once the session has ended', async () => {
    answer({ authenticated: false, csrfToken: 't' });
    await expect(firstCheck()).resolves.toEqual({ signedIn: false, error: null });
  });

  it('shows the form with the reason when the server cannot answer', async () => {
    answer({ error: { code: 'internal' } }, 500);
    await expect(firstCheck()).resolves.toEqual({ signedIn: false, error: 'server' });
  });

  it('says offline when the request never reaches the server', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))),
    );
    await expect(firstCheck()).resolves.toEqual({ signedIn: false, error: 'offline' });
  });
});
