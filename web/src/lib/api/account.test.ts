import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  beginTotp,
  disableTotp,
  endOtherSessions,
  endSession,
  getAudit,
  newAddressURL,
  randomPort,
  renewLogin,
  startAddress,
} from './account';
import { api } from './client';

afterEach(() => vi.unstubAllGlobals());

function stub(body: unknown, status = 200) {
  const f = vi.fn(() =>
    Promise.resolve(
      new Response(body === null ? null : JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    ),
  );
  vi.stubGlobal('fetch', f);
  return () => f.mock.calls.at(-1) as unknown as [string, RequestInit];
}

describe('two-step sign-in', () => {
  it('starts setup and reads the secret', async () => {
    const last = stub({ secret: 'JBSWY3DPEHPK3PXP', uri: 'otpauth://totp/Karvan:op' });
    await expect(beginTotp()).resolves.toEqual({
      secret: 'JBSWY3DPEHPK3PXP',
      uri: 'otpauth://totp/Karvan:op',
    });
    expect(last()[0]).toMatch(/account\/totp$/);
    expect(last()[1].method).toBe('POST');
  });

  it('turns off with the password and a code', async () => {
    const last = stub(null, 204);
    await disableTotp({ password: 'pw', code: '123456' });
    expect(last()[1].method).toBe('DELETE');
    expect(JSON.parse(last()[1].body as string)).toEqual({ password: 'pw', code: '123456' });
  });
});

describe('sessions', () => {
  it('signs out the others and reads how many ended', async () => {
    const last = stub({ ended: 3 });
    await expect(endOtherSessions()).resolves.toBe(3);
    expect(last()[0]).toMatch(/account\/sessions$/);
    expect(last()[1].method).toBe('DELETE');
  });

  it('signs out one by its id', async () => {
    const last = stub(null, 204);
    await endSession('Ab-_09xyzAb-_09x');
    expect(last()[0]).toMatch(/account\/sessions\/Ab-_09xyzAb-_09x$/);
  });
});

describe('new login', () => {
  it('returns the new values and keeps the new CSRF token for later requests', async () => {
    stub({ username: 'k7x2', password: 'secret-1', csrfToken: 'fresh-token' });
    await expect(renewLogin({ password: 'old', code: '' })).resolves.toEqual({
      username: 'k7x2',
      password: 'secret-1',
    });
    const last = stub(null, 204);
    await api('DELETE', 'account/sessions/x');
    expect(new Headers(last()[1].headers).get('X-CSRF-Token')).toBe('fresh-token');
  });
});

describe('address', () => {
  it('asks for a random port with the password', async () => {
    const last = stub({ port: 31000, path: 'p4th', expiresAt: '2026-10-25T10:10:00Z' });
    await startAddress({ newPath: false, port: randomPort }, { password: 'pw', code: '' });
    expect(JSON.parse(last()[1].body as string)).toEqual({
      password: 'pw',
      code: '',
      newPath: false,
      port: -1,
    });
  });

  it('builds the new address on this host and scheme', () => {
    const to = { port: 31000, path: 'n3wP4th', expiresAt: '' };
    expect(newAddressURL('http://203.0.113.5:24364/oldPath/', to, 'settings/access')).toBe(
      'http://203.0.113.5:31000/n3wP4th/settings/access',
    );
    expect(newAddressURL('https://[2001:db8::1]:8443/old/settings?x=1#y', to)).toBe(
      'https://[2001:db8::1]:31000/n3wP4th/',
    );
  });

  it('leaves the port out where it is the scheme default', () => {
    expect(
      newAddressURL('https://panel.example.com:8443/a/', { port: 443, path: 'b', expiresAt: '' }),
    ).toBe('https://panel.example.com/b/');
  });
});

describe('audit log', () => {
  it('pages by id with the filters', async () => {
    const last = stub({ events: [], next: null });
    await getAudit({ action: 'login', outcome: 'failure' }, 120, 50);
    const url = new URL(last()[0]);
    expect(url.pathname).toMatch(/audit$/);
    expect(Object.fromEntries(url.searchParams)).toEqual({
      limit: '50',
      before: '120',
      action: 'login',
      outcome: 'failure',
    });
  });

  it('leaves out the filters it does not have', async () => {
    const last = stub({ events: [], next: null });
    await getAudit({ action: '', outcome: '' });
    expect([...new URL(last()[0]).searchParams.keys()]).toEqual(['limit']);
  });
});
