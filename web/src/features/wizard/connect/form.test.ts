import { describe, expect, it } from 'vitest';
import type { Panel } from '../../../lib/api/panels';
import {
  certOf,
  defaultDestination,
  emptyValues,
  fingerprintPairs,
  missingField,
  sourceUrl,
  toNewJob,
  type ConnectValues,
} from './form';

const filled: ConnectValues = {
  ...emptyValues,
  panelId: 2,
  url: ' http://127.0.0.1:18000 ',
  username: ' root ',
  password: ' secret ',
};

describe('sourceUrl', () => {
  it('keeps an address that has a scheme', () => {
    expect(sourceUrl('http://10.0.0.7:8000/')).toBe('http://10.0.0.7:8000/');
    expect(sourceUrl('HTTPS://panel.example.com')).toBe('HTTPS://panel.example.com');
  });

  it('adds https to an address typed without one', () => {
    expect(sourceUrl(' panel.example.com:8443 ')).toBe('https://panel.example.com:8443');
  });

  it('leaves an empty address empty', () => {
    expect(sourceUrl('  ')).toBe('');
  });
});

describe('toNewJob', () => {
  it('sends a username and password and lets Karvan find the kind', () => {
    expect(toNewJob(filled)).toEqual({
      panelId: 2,
      sourceUrl: 'http://127.0.0.1:18000',
      username: 'root',
      password: ' secret ',
    });
  });

  it('names the kind when the operator picked one', () => {
    expect(toNewJob({ ...filled, kind: 'marzban' }).sourceKind).toBe('marzban');
  });

  it('sends an API key as a PasarGuard sign-in, without the username and password', () => {
    const job = toNewJob({ ...filled, kind: 'marzban', signIn: 'apiKey', apiKey: ' k1 ' });
    expect(job).toEqual({
      panelId: 2,
      sourceUrl: 'http://127.0.0.1:18000',
      sourceKind: 'pasarguard',
      apiKey: 'k1',
    });
  });

  it('pins the certificate the operator trusted', () => {
    expect(toNewJob(filled, 'ab').pinSha256).toBe('ab');
    expect(toNewJob(filled)).not.toHaveProperty('pinSha256');
  });
});

describe('missingField', () => {
  it('asks for each field in order', () => {
    expect(missingField(emptyValues)).toBe('panel');
    expect(missingField({ ...filled, url: ' ' })).toBe('url');
    expect(missingField({ ...filled, username: '' })).toBe('username');
    expect(missingField({ ...filled, password: '' })).toBe('password');
    expect(missingField({ ...filled, signIn: 'apiKey' })).toBe('apiKey');
  });

  it('lets a complete form go', () => {
    expect(missingField(filled)).toBeNull();
    expect(missingField({ ...filled, username: '', signIn: 'apiKey', apiKey: 'k' })).toBeNull();
  });
});

describe('defaultDestination', () => {
  const panel = (id: number, ready: boolean) => ({ id, ready }) as Panel;

  it('starts on the current panel when it is ready', () => {
    expect(defaultDestination([panel(1, true), panel(2, true)], panel(2, true))).toBe(2);
  });

  it('falls back to the first ready panel', () => {
    expect(
      defaultDestination([panel(0, true), panel(1, false), panel(3, true)], panel(1, false)),
    ).toBe(3);
    expect(defaultDestination([panel(1, false)], null)).toBe(0);
  });
});

describe('certOf', () => {
  const sha = 'AB'.repeat(32);

  it('reads the certificate of an untrusted panel', () => {
    expect(
      certOf('source_tls_untrusted', {
        sha256: sha,
        subject: 'CN=old',
        notAfter: '2027-01-01T00:00:00Z',
      }),
    ).toEqual({ sha256: sha.toLowerCase(), subject: 'CN=old', notAfter: '2027-01-01T00:00:00Z' });
  });

  it('has nothing to trust without a well formed fingerprint', () => {
    expect(certOf('source_tls_untrusted', {})).toBeNull();
    expect(certOf('source_tls_untrusted', { sha256: 'xyz' })).toBeNull();
    expect(certOf('source_unreachable', { sha256: sha })).toBeNull();
  });

  it('groups the fingerprint in pairs', () => {
    expect(fingerprintPairs('ab12cd')).toBe('AB:12:CD');
  });
});
