import { describe, expect, it } from 'vitest';
import type { Domain } from '../../lib/api/domains';
import type { Https, PendingHttps } from '../../lib/api/https';
import type { CertFacts } from '../../lib/api/rules';
import {
  certifiedDomains,
  formOf,
  httpsLink,
  httpsStart,
  sharedPortsOf,
  typedDigits,
  type HttpsForm,
} from './https-form';

const plain: Https = {
  tls: 'off',
  domainId: 0,
  shared: false,
  port: 24364,
  pending: null,
  sharedPorts: [
    { domain: 'panel.example.com', port: 443 },
    { domain: 'panel.example.com', port: 8443 },
    { domain: 'other.example.com', port: 443 },
  ],
};

function domain(id: number, name: string, extra: Partial<Domain> = {}): Domain {
  return {
    id,
    name,
    certificate: { id, name, origin: '', facts: {} as CertFacts },
    certificates: [],
    pins: [],
    acme: null,
    rules: [],
    warnings: [],
    ...extra,
  };
}

const form = (f: Partial<HttpsForm>): HttpsForm => ({ ...formOf(plain), ...f });

describe('httpsStart', () => {
  it('sends a mode change on the same port as port 0', () => {
    expect(httpsStart(form({ tls: 'self' }), plain)).toEqual({
      tls: 'self',
      domainId: 0,
      port: 0,
      shared: false,
    });
  });
  it('sends nothing when nothing changes', () => {
    expect(httpsStart(form({}), plain)).toBeNull();
    expect(httpsStart(form({ where: 'typed', typed: '24364' }), plain)).toBeNull();
  });
  it('needs a domain in domain mode', () => {
    expect(httpsStart(form({ tls: 'domain' }), plain)).toBeNull();
    expect(httpsStart(form({ tls: 'domain', domainId: 3 }), plain)?.domainId).toBe(3);
  });
  it('takes a typed port from 1 to 65535', () => {
    expect(httpsStart(form({ tls: 'self', where: 'typed', typed: '8443' }), plain)?.port).toBe(
      8443,
    );
    expect(httpsStart(form({ tls: 'self', where: 'typed', typed: '' }), plain)).toBeNull();
    expect(httpsStart(form({ tls: 'self', where: 'typed', typed: '70000' }), plain)).toBeNull();
    expect(httpsStart(form({ tls: 'self', where: 'typed', typed: '0' }), plain)).toBeNull();
  });
  it('shares a bridge port only in domain mode', () => {
    const shared = form({ tls: 'domain', domainId: 3, where: 'shared', sharedPort: 443 });
    expect(httpsStart(shared, plain)).toEqual({
      tls: 'domain',
      domainId: 3,
      port: 443,
      shared: true,
    });
    expect(httpsStart({ ...shared, sharedPort: 0 }, plain)).toBeNull();
    expect(httpsStart({ ...shared, tls: 'self' }, plain)?.shared).toBe(false);
  });
  it('leaves a shared port only for a port of its own', () => {
    const on: Https = { ...plain, tls: 'domain', domainId: 3, shared: true, port: 443 };
    expect(httpsStart(formOf(on), on)).toBeNull();
    expect(httpsStart({ ...formOf(on), where: 'keep' }, on)).toBeNull();
    expect(httpsStart({ ...formOf(on), where: 'typed', typed: '2053' }, on)).toEqual({
      tls: 'domain',
      domainId: 3,
      port: 2053,
      shared: false,
    });
  });
});

describe('domains and ports', () => {
  it('offers domains whose certificate covers their name', () => {
    const list = [
      domain(1, 'a.example.com'),
      domain(2, 'b.example.com', { certificate: null }),
      domain(3, 'c.example.com', { warnings: ['domain_mismatch'] }),
      domain(4, 'd.example.com', { warnings: ['expires_soon'] }),
    ];
    expect(certifiedDomains(list).map((d) => d.id)).toEqual([1, 4]);
  });
  it('lists the HTTPS rule ports of the chosen domain', () => {
    expect(sharedPortsOf(plain, domain(3, 'panel.example.com'))).toEqual([443, 8443]);
    expect(sharedPortsOf(plain, undefined)).toEqual([]);
  });
  it('reads Persian digits in a typed port', () => {
    expect(typedDigits('۸۴۴۳x99')).toBe('84439');
  });
});

describe('httpsLink', () => {
  const pending: PendingHttps = {
    tls: 'self',
    domainId: 0,
    shared: false,
    port: 24364,
    pathChanged: false,
    expiresAt: '2026-10-29T10:00:00Z',
    fromNew: false,
  };
  const base = 'http://203.0.113.5:24364/secret/';
  it('opens the same host over HTTPS with this session', () => {
    expect(httpsLink(pending, [], base)).toEqual({
      url: 'https://203.0.113.5:24364/secret/settings/https',
      signInFirst: false,
    });
  });
  it("uses the domain's name in domain mode, where the session does not go", () => {
    const p = { ...pending, tls: 'domain' as const, domainId: 3, port: 443, shared: true };
    expect(httpsLink(p, [domain(3, 'panel.example.com')], base)).toEqual({
      url: 'https://panel.example.com/secret/settings/https',
      signInFirst: true,
    });
  });
  it('keeps this host for a wildcard domain', () => {
    const p = { ...pending, tls: 'domain' as const, domainId: 3 };
    expect(httpsLink(p, [domain(3, '*.example.com')], base).url).toBe(
      'https://203.0.113.5:24364/secret/settings/https',
    );
  });
  it('signs in again when HTTPS turns into HTTP', () => {
    const p = { ...pending, tls: 'off' as const, port: 8080 };
    expect(httpsLink(p, [], 'https://203.0.113.5:24364/secret/')).toEqual({
      url: 'http://203.0.113.5:8080/secret/settings/https',
      signInFirst: true,
    });
  });
  it('has no link while a new hidden path waits', () => {
    expect(httpsLink({ ...pending, pathChanged: true }, [], base).url).toBeNull();
  });
});
