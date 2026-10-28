import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  deleteCertificate,
  getCertificates,
  saveCertificate,
  scanCertificates,
} from './certificates';

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

describe('stored certificates', () => {
  it('are listed with their facts and rules', async () => {
    const body = { certificates: [{ id: 1, name: 'old', kind: 'file', rules: [] }] };
    const last = stubFetch(body);
    await expect(getCertificates()).resolves.toEqual(body);
    expect(last()[0]).toMatch(/api\/v1\/certificates$/);
  });

  it('add a file pair with POST', async () => {
    const last = stubFetch({ id: 4 });
    const input = { name: 'le', certFile: '/etc/a.pem', keyFile: '/etc/a.key' };
    await saveCertificate(input);
    const [url, init] = last();
    expect(url).toMatch(/api\/v1\/certificates$/);
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body as string)).toEqual(input);
  });

  it('change a pasted one with PATCH by its id', async () => {
    const last = stubFetch({ id: 4 });
    await saveCertificate({ name: 'le', certPem: 'CERT', keyPem: '' }, 4);
    const [url, init] = last();
    expect(url).toMatch(/certificates\/4$/);
    expect(init.method).toBe('PATCH');
  });

  it('are deleted by their id', async () => {
    const last = stubFetch(undefined, 204);
    await deleteCertificate(4);
    const [url, init] = last();
    expect(url).toMatch(/certificates\/4$/);
    expect(init.method).toBe('DELETE');
  });
});

describe('the certificate scan', () => {
  it('posts no path and reads what it found and skipped', async () => {
    const body = { found: [], skipped: [{ place: '/root/cert', reason: 'not_found' }] };
    const last = stubFetch(body);
    await expect(scanCertificates()).resolves.toEqual(body);
    const [url, init] = last();
    expect(url).toMatch(/certificates\/scan$/);
    expect(init.method).toBe('POST');
    expect(init.body ?? undefined).toBeUndefined();
  });
});
