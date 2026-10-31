import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import type { CertFacts, CertWarning } from '../../lib/api/rules';
import {
  acmeMethodsFor,
  bestCert,
  choiceOf,
  covers,
  domainName,
  isPortError,
  missingOf,
  newDomainForm,
  recommendedSource,
  refusedDomainField,
  serverCerts,
  stepAfter,
  stepsFor,
  wayFor,
  type DomainFormValues,
  type ServerCert,
} from './domain-form';

const facts = (
  domains: string[],
  notAfter = '2027-01-01T00:00:00Z',
  warnings: CertWarning[] = [],
): CertFacts => ({ domains, issuer: 'R11', notAfter, warnings });
const form = (patch: Partial<DomainFormValues>): DomainFormValues => ({
  ...newDomainForm(),
  name: 'sub.example.com',
  ...patch,
});
const now = new Date('2026-10-29T00:00:00Z');

describe('a domain name', () => {
  it('is read as the server stores it: trimmed, lower case, no trailing dot', () => {
    expect(domainName('  Sub.Example.COM. ')).toBe('sub.example.com');
  });
});

describe('a certificate name covering a domain', () => {
  it('matches the same name in any case', () => {
    expect(covers('Sub.Example.com', 'sub.example.com')).toBe(true);
  });
  it('matches one label under a wildcard, never two or the bare name', () => {
    expect(covers('*.example.com', 'sub.example.com')).toBe(true);
    expect(covers('*.example.com', 'a.sub.example.com')).toBe(false);
    expect(covers('*.example.com', 'example.com')).toBe(false);
  });
  it('lets a wildcard domain take only the same wildcard', () => {
    expect(covers('*.example.com', '*.example.com')).toBe(true);
    expect(covers('sub.example.com', '*.example.com')).toBe(false);
  });
});

describe('the steps', () => {
  it('ask for the name, HTTPS, the certificate, then its details', () => {
    expect(stepsFor(true, 'with')).toEqual(['domain', 'https', 'source', 'details']);
  });
  it('skip the certificate without one', () => {
    expect(stepsFor(true, 'without')).toEqual(['domain', 'https', 'details']);
  });
  it('start at HTTPS on the change page', () => {
    expect(stepsFor(false, 'with')).toEqual(['https', 'source', 'details']);
  });
  it('go on from the name only once one is typed', () => {
    expect(stepAfter('')).toBe('domain');
    expect(stepAfter('  . ')).toBe('domain');
    expect(stepAfter('sub.example.com')).toBe('https');
  });
});

const cert = (id: number, name: string, f: CertFacts, used = false): Certificate => ({
  id,
  name,
  kind: 'pasted',
  origin: '',
  hasKey: true,
  facts: f,
  domains: used ? [{ id: 9, name: 'other.test', warnings: [] }] : [],
  rules: [],
  found: false,
  copies: [],
});

describe('the certificates on this server', () => {
  const stored = [
    {
      ...cert(4, 'le-main', facts(['*.example.com'])),
      kind: 'file' as const,
      certFile: '/etc/letsencrypt/live/a/fullchain.pem',
      keyFile: '/etc/letsencrypt/live/a/privkey.pem',
    },
    cert(5, 'other', facts(['other.test'])),
    cert(6, 'taken', facts(['sub.example.com']), true),
  ];
  const scan: ScanResult = {
    found: [
      {
        place: '/etc/letsencrypt/live',
        certFile: '/etc/letsencrypt/live/a/fullchain.pem',
        keyFile: '/etc/letsencrypt/live/a/privkey.pem',
        facts: facts(['*.example.com']),
        added: true,
      },
      {
        place: '/root/cert',
        certFile: '/root/cert/sub.pem',
        keyFile: '/root/cert/sub.key',
        facts: facts(['sub.example.com']),
        added: false,
      },
    ],
    skipped: [],
  };
  it('list exact names, then wildcards, then those that do not cover it', () => {
    const list = serverCerts('Sub.example.com', stored, scan);
    expect(list.map((c) => [c.title, c.covers])).toEqual([
      ['/root/cert/sub.pem', true],
      ['le-main', true],
      ['other', false],
    ]);
  });
  it('leave out those another domain uses', () => {
    expect(serverCerts('sub.example.com', stored, scan).some((c) => c.title === 'taken')).toBe(
      false,
    );
  });
  it('show a found file that is already stored only once, as the stored one', () => {
    const list = serverCerts('sub.example.com', stored, scan);
    expect(list.some((c) => c.title === '/etc/letsencrypt/live/a/fullchain.pem')).toBe(false);
  });
  it('list the stored ones while the scan has not answered', () => {
    expect(serverCerts('sub.example.com', stored, undefined)).toHaveLength(2);
  });
  it('are none before the name is typed', () => {
    expect(serverCerts('  ', stored, scan)).toEqual([]);
  });
});

const row = (key: string, f: CertFacts, ok = true): ServerCert => ({
  key,
  pick: { kind: 'stored', certificateId: Number(key) },
  title: key,
  facts: f,
  covers: ok,
});

describe('the best certificate on this server', () => {
  it('is the one that ends last', () => {
    const list = [
      row('1', facts(['a.test'], '2026-12-01T00:00:00Z')),
      row('2', facts(['a.test'], '2027-03-01T00:00:00Z')),
      row('3', facts(['a.test'], '2027-01-01T00:00:00Z')),
    ];
    expect(bestCert(list, now)?.key).toBe('2');
  });
  it('keeps the earlier one on a tie', () => {
    expect(bestCert([row('1', facts(['a.test'])), row('2', facts(['*.test']))], now)?.key).toBe(
      '1',
    );
  });
  it('is never one that has ended, cannot serve or does not cover the name', () => {
    const list = [
      row('1', facts(['a.test'], '2026-10-01T00:00:00Z')),
      row('2', facts(['a.test'], '2028-01-01T00:00:00Z', ['expired'])),
      row('3', facts(['a.test'], '2028-01-01T00:00:00Z', ['no_key'])),
      row('4', facts(['b.test'], '2029-01-01T00:00:00Z'), false),
      row('5', facts(['a.test'], '2026-11-02T00:00:00Z', ['expires_soon'])),
    ];
    expect(bestCert(list, now)?.key).toBe('5');
    expect(bestCert(list.slice(0, 4), now)).toBeUndefined();
  });
});

describe('the recommended certificate', () => {
  const best = row('2', facts(['a.test']));
  it('is the one this server has', () => {
    expect(recommendedSource('a.test', best)).toBe('own');
    expect(wayFor('own', best)).toEqual({ way: 'found', pick: best.pick });
  });
  it("is Let's Encrypt automatic when this server has none", () => {
    expect(recommendedSource('a.test', undefined)).toBe('acme-http');
  });
  it("is Let's Encrypt by DNS for a wildcard", () => {
    expect(recommendedSource('*.example.com', undefined)).toBe('acme-dns');
  });
  it('opens upload or paste for a certificate the operator has when the server has none', () => {
    expect(wayFor('own', undefined)).toEqual({ way: 'paste', pick: null });
    expect(wayFor('acme-dns', best)).toEqual({ way: 'acme-dns', pick: null });
  });
});

describe("the Let's Encrypt methods", () => {
  it('offer both for a host name', () => {
    expect(acmeMethodsFor('sub.example.com')).toEqual(['http-01', 'dns-01']);
  });
  it('offer only DNS for a wildcard', () => {
    expect(acmeMethodsFor('*.example.com')).toEqual(['dns-01']);
  });
});

describe('what a way still needs', () => {
  it('needs a pick on this server', () => {
    expect(missingOf(form({ way: 'found', pick: null }), false)).toBe('pick');
    const pick = { kind: 'stored' as const, certificateId: 4 };
    expect(missingOf(form({ way: 'found', pick }), false)).toBeUndefined();
  });
  it('needs both the certificate and the key for upload or paste', () => {
    expect(missingOf(form({ way: 'paste', certPem: 'CERT' }), false)).toBe('pem');
    expect(missingOf(form({ way: 'paste', keyPem: 'KEY' }), false)).toBe('pem');
    expect(missingOf(form({ way: 'paste', certPem: 'C', keyPem: 'K' }), false)).toBeUndefined();
  });
  it('needs both typed paths', () => {
    expect(missingOf(form({ way: 'file', certFile: '/a.pem', keyFile: ' ' }), false)).toBe('files');
    const v = form({ way: 'file', certFile: '/a.pem', keyFile: '/a.key' });
    expect(missingOf(v, false)).toBeUndefined();
  });
  it("needs the agreement before the first Let's Encrypt order", () => {
    expect(missingOf(form({ way: 'acme-http' }), true)).toBe('agree');
    expect(missingOf(form({ way: 'acme-dns', agreeTos: true }), true)).toBeUndefined();
    expect(missingOf(form({ way: 'acme-dns' }), false)).toBeUndefined();
  });
  it('needs DNS for a wildcard', () => {
    expect(missingOf(form({ name: '*.example.com', way: 'acme-http' }), false)).toBe('wildcard');
  });
  it('needs nothing without a certificate', () => {
    expect(missingOf(form({ way: 'none' }), false)).toBeUndefined();
  });
});

describe('the certificate choice the form sends', () => {
  it('sends a pick on this server as it is', () => {
    const pick = { kind: 'stored' as const, certificateId: 4 };
    expect(choiceOf(form({ way: 'found', pick }), false)).toEqual(pick);
    expect(choiceOf(form({ way: 'found', pick: null }), false)).toBeUndefined();
  });
  it('sends an upload or paste as its text', () => {
    const v = form({ way: 'paste', certPem: 'CERT', keyPem: 'KEY' });
    expect(choiceOf(v, false)).toEqual({ kind: 'pasted', certPem: 'CERT', keyPem: 'KEY' });
    expect(choiceOf({ ...v, keyPem: ' ' }, false)).toBeUndefined();
  });
  it('confirms plain HTTP without a certificate', () => {
    expect(choiceOf(form({ way: 'none' }), false)).toEqual({ kind: 'none', plainOk: true });
  });
  it('sends both typed paths, trimmed', () => {
    const v = form({ way: 'file', certFile: ' /etc/a.pem ', keyFile: '/etc/a.key' });
    expect(choiceOf(v, false)).toEqual({
      kind: 'file',
      certFile: '/etc/a.pem',
      keyFile: '/etc/a.key',
    });
    expect(choiceOf({ ...v, keyFile: '' }, false)).toBeUndefined();
  });
  it("sends the chosen Let's Encrypt check, with the agreement only before the first order", () => {
    expect(choiceOf(form({ way: 'acme-http' }), false)).toEqual({
      kind: 'acme',
      method: 'http-01',
    });
    const v = form({ way: 'acme-dns' });
    expect(choiceOf(v, true)).toBeUndefined();
    expect(choiceOf({ ...v, agreeTos: true }, true)).toEqual({
      kind: 'acme',
      method: 'dns-01',
      agreeTos: true,
    });
  });
  it('never sends the automatic way for a wildcard', () => {
    expect(choiceOf(form({ name: '*.example.com', way: 'acme-http' }), false)).toBeUndefined();
  });
});

describe('a refused answer', () => {
  it('names the field the form shows', () => {
    expect(
      refusedDomainField(new ApiError(400, 'domain_invalid', undefined, { field: 'name' })),
    ).toBe('name');
    expect(
      refusedDomainField(new ApiError(400, 'certificate_invalid', undefined, { field: 'keyPem' })),
    ).toBe('keyPem');
    expect(
      refusedDomainField(
        new ApiError(400, 'certificate_invalid', undefined, { field: 'certificate' }),
      ),
    ).toBeUndefined();
    expect(refusedDomainField(new Error('x'))).toBeUndefined();
  });
  it('tells a port problem, which the DNS way avoids', () => {
    expect(isPortError(new ApiError(409, 'acme_port_busy', undefined, {}))).toBe(true);
    expect(isPortError(new ApiError(409, 'acme_port_tls', undefined, {}))).toBe(true);
    expect(isPortError(new ApiError(409, 'acme_busy', undefined, {}))).toBe(false);
  });
});
