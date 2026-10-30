import { afterEach, describe, expect, it, vi } from 'vitest';
import { addDomain, deleteDomain, getDomains, setDomainCertificate } from './domains';

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

describe('domains', () => {
  it('are listed with their certificates, rules and warnings', async () => {
    const d = { id: 1, name: 'old.example.com', certificate: null, certificates: [], pins: [] };
    const body = { domains: [{ ...d, acme: null, rules: [], warnings: [] }] };
    const last = stubFetch(body);
    await expect(getDomains()).resolves.toEqual(body);
    expect(last()[0]).toMatch(/api\/v1\/domains$/);
  });

  it('read every null list as empty', async () => {
    stubFetch({ domains: null });
    await expect(getDomains()).resolves.toEqual({ domains: [] });
    const facts = { domains: null, issuer: '', warnings: null };
    const d = { id: 1, certificate: { id: 2, facts }, certificates: null, pins: null };
    stubFetch({ domains: [{ ...d, rules: null, warnings: null }] });
    await expect(getDomains()).resolves.toEqual({
      domains: [
        {
          id: 1,
          certificate: { id: 2, facts: { domains: [], issuer: '', warnings: [] } },
          certificates: [],
          pins: [],
          rules: [],
          warnings: [],
        },
      ],
    });
  });

  it('are added by name, trimmed, with their certificate', async () => {
    const last = stubFetch({ id: 2, name: 'sub.example.com' });
    await addDomain(' sub.example.com ', { kind: 'stored', certificateId: 4 });
    const [url, init] = last();
    expect(url).toMatch(/api\/v1\/domains$/);
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body as string)).toEqual({
      name: 'sub.example.com',
      certificate: { kind: 'stored', certificateId: 4 },
    });
  });

  it('change their certificate by id', async () => {
    const last = stubFetch({ id: 2, name: 'sub.example.com' });
    await setDomainCertificate(2, { kind: 'none', plainOk: true });
    const [url, init] = last();
    expect(url).toMatch(/domains\/2\/certificate$/);
    expect(init.method).toBe('PUT');
    expect(JSON.parse(init.body as string)).toEqual({
      certificate: { kind: 'none', plainOk: true },
    });
  });

  it('are deleted by their id', async () => {
    const last = stubFetch(undefined, 204);
    await deleteDomain(2);
    const [url, init] = last();
    expect(url).toMatch(/domains\/2$/);
    expect(init.method).toBe('DELETE');
  });
});
