import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import type { CertFacts, CertWarning } from '../../lib/api/rules';
import {
  acmeMethodsFor,
  bestChoice,
  choiceOf,
  covers,
  domainName,
  foundChoices,
  isPortError,
  missingOf,
  newDomainForm,
  otherWays,
  recommended,
  refusedDomainField,
  stepAfter,
  type DomainFormValues,
  type FoundChoice,
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

describe('the choices found on this server', () => {
  const stored: Certificate[] = [
    {
      id: 4,
      name: 'le-main',
      kind: 'file',
      origin: '',
      certFile: '/etc/letsencrypt/live/a/fullchain.pem',
      keyFile: '/etc/letsencrypt/live/a/privkey.pem',
      hasKey: true,
      facts: facts(['*.example.com']),
      domains: [],
      rules: [],
    },
    {
      id: 5,
      name: 'other',
      kind: 'pasted',
      origin: '',
      hasKey: true,
      facts: facts(['other.test']),
      domains: [],
      rules: [],
    },
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
      {
        place: '/root/cert',
        certFile: '/root/cert/x.pem',
        keyFile: '/root/cert/x.key',
        facts: facts(['x.test']),
        added: false,
      },
    ],
    skipped: [],
  };
  it('lists exact names before wildcards, and stored ones before found files', () => {
    const list = foundChoices('Sub.example.com', stored, scan);
    expect(list.map((c) => c.pick)).toEqual([
      { kind: 'file', certFile: '/root/cert/sub.pem', keyFile: '/root/cert/sub.key' },
      { kind: 'stored', certificateId: 4 },
    ]);
    const exact: Certificate = {
      id: 6,
      name: 'exact',
      kind: 'pasted',
      origin: '',
      hasKey: true,
      facts: facts(['sub.example.com']),
      domains: [],
      rules: [],
    };
    const both = foundChoices('sub.example.com', [...stored, exact], scan);
    expect(both.map((c) => c.title)).toEqual(['exact', '/root/cert/sub.pem', 'le-main']);
  });
  it("leaves out the domain's own certificate on the change page", () => {
    const list = foundChoices('sub.example.com', stored, scan, 4);
    expect(list.map((c) => c.title)).toEqual(['/root/cert/sub.pem']);
  });
  it('shows a found file that is already stored only once, as the stored one', () => {
    const list = foundChoices('sub.example.com', stored, scan);
    expect(list.filter((c) => c.title === 'le-main')).toHaveLength(1);
    expect(list.some((c) => c.title === '/etc/letsencrypt/live/a/fullchain.pem')).toBe(false);
  });
  it('lists the stored ones while the scan has not answered', () => {
    expect(foundChoices('sub.example.com', stored, undefined)).toHaveLength(1);
  });
  it('offers nothing before the name is typed', () => {
    expect(foundChoices('  ', stored, scan)).toEqual([]);
  });
});

describe("the Let's Encrypt methods", () => {
  it('offer both for a host name', () => {
    expect(acmeMethodsFor('sub.example.com')).toEqual(['http-01', 'dns-01']);
  });
  it('offer only the manual way for a wildcard', () => {
    expect(acmeMethodsFor('*.example.com')).toEqual(['dns-01']);
  });
});

const now = new Date('2026-10-29T00:00:00Z');
const choice = (key: string, f: CertFacts): FoundChoice => ({
  key,
  pick: { kind: 'stored', certificateId: Number(key) },
  title: key,
  facts: f,
});

describe('the steps of Add domain', () => {
  it('give no certificate step without a domain', () => {
    expect(stepAfter('')).toBe('domain');
    expect(stepAfter('  . ')).toBe('domain');
  });
  it('open the certificate step once a name is typed', () => {
    expect(stepAfter('sub.example.com')).toBe('certificate');
  });
});

describe('the best certificate found', () => {
  it('is the one that ends last', () => {
    const list = [
      choice('1', facts(['a.test'], '2026-12-01T00:00:00Z')),
      choice('2', facts(['a.test'], '2027-03-01T00:00:00Z')),
      choice('3', facts(['a.test'], '2027-01-01T00:00:00Z')),
    ];
    expect(bestChoice(list, now)?.key).toBe('2');
  });
  it('keeps the earlier one in the list on a tie', () => {
    const list = [choice('1', facts(['a.test'])), choice('2', facts(['*.test']))];
    expect(bestChoice(list, now)?.key).toBe('1');
  });
  it('is never one that has ended or cannot serve', () => {
    const list = [
      choice('1', facts(['a.test'], '2026-10-01T00:00:00Z')),
      choice('2', facts(['a.test'], '2028-01-01T00:00:00Z', ['expired'])),
      choice('3', facts(['a.test'], '2028-01-01T00:00:00Z', ['no_key'])),
      choice('4', facts(['a.test'], '2026-11-02T00:00:00Z', ['expires_soon'])),
    ];
    expect(bestChoice(list, now)?.key).toBe('4');
    expect(bestChoice(list.slice(0, 3), now)).toBeUndefined();
  });
});

describe('the recommended answer', () => {
  it('picks the best found certificate', () => {
    const list = [
      choice('1', facts(['a.test'], '2026-12-01T00:00:00Z')),
      choice('2', facts(['a.test'], '2027-03-01T00:00:00Z')),
    ];
    expect(recommended('a.test', list, now)).toEqual({
      way: 'found',
      pick: { kind: 'stored', certificateId: 2 },
    });
  });
  it("picks Let's Encrypt automatic when none is found", () => {
    expect(recommended('a.test', [], now)).toEqual({ way: 'acme-http', pick: null });
  });
  it("picks Let's Encrypt automatic when every found one has ended", () => {
    const list = [choice('1', facts(['a.test'], '2026-01-01T00:00:00Z', ['expired']))];
    expect(recommended('a.test', list, now).way).toBe('acme-http');
  });
  it("picks Let's Encrypt by DNS for a wildcard", () => {
    expect(recommended('*.example.com', [], now)).toEqual({ way: 'acme-dns', pick: null });
  });
});

describe('the Other ways menu', () => {
  it('offers every way but the chosen one, No certificate last', () => {
    expect(otherWays('found', 'a.test', true)).toEqual([
      'acme-http',
      'acme-dns',
      'upload',
      'paste',
      'file',
      'none',
    ]);
    expect(otherWays('paste', 'a.test', true)).toEqual([
      'found',
      'acme-http',
      'acme-dns',
      'upload',
      'file',
      'none',
    ]);
  });
  it('offers found certificates only when there are some', () => {
    expect(otherWays('acme-http', 'a.test', false)).not.toContain('found');
  });
  it('never offers the automatic way for a wildcard', () => {
    expect(otherWays('acme-dns', '*.example.com', false)).toEqual([
      'upload',
      'paste',
      'file',
      'none',
    ]);
  });
});

describe('what a way still needs', () => {
  it('needs a pick among the found ones', () => {
    expect(missingOf(form({ way: 'found', pick: null }), false)).toBe('pick');
    const pick = { kind: 'stored' as const, certificateId: 4 };
    expect(missingOf(form({ way: 'found', pick }), false)).toBeUndefined();
  });
  it('needs both the certificate and the key for upload and paste', () => {
    expect(missingOf(form({ way: 'paste', certPem: 'CERT' }), false)).toBe('pem');
    expect(missingOf(form({ way: 'upload', keyPem: 'KEY' }), false)).toBe('pem');
    expect(missingOf(form({ way: 'paste', certPem: 'C', keyPem: 'K' }), false)).toBeUndefined();
  });
  it('needs both paths for a file path', () => {
    expect(missingOf(form({ way: 'file', certFile: '/a.pem', keyFile: ' ' }), false)).toBe('files');
    const v = form({ way: 'file', certFile: '/a.pem', keyFile: '/a.key' });
    expect(missingOf(v, false)).toBeUndefined();
  });
  it("needs the agreement before the first Let's Encrypt order", () => {
    expect(missingOf(form({ way: 'acme-http' }), true)).toBe('agree');
    expect(missingOf(form({ way: 'acme-dns', agreeTos: true }), true)).toBeUndefined();
    expect(missingOf(form({ way: 'acme-dns' }), false)).toBeUndefined();
  });
  it('needs the manual way for a wildcard', () => {
    expect(missingOf(form({ name: '*.example.com', way: 'acme-http' }), false)).toBe('wildcard');
  });
  it('needs nothing for no certificate', () => {
    expect(missingOf(form({ way: 'none' }), false)).toBeUndefined();
  });
});

describe('the certificate choice the form sends', () => {
  it('sends a stored or found pick as it is', () => {
    const pick = { kind: 'stored' as const, certificateId: 4 };
    expect(choiceOf(form({ way: 'found', pick }), false)).toEqual(pick);
    expect(choiceOf(form({ way: 'found', pick: null }), false)).toBeUndefined();
  });
  it('sends an upload as the pasted text it read', () => {
    const v = form({ way: 'upload', certPem: 'CERT', keyPem: 'KEY' });
    expect(choiceOf(v, false)).toEqual({ kind: 'pasted', certPem: 'CERT', keyPem: 'KEY' });
    expect(choiceOf({ ...v, keyPem: ' ' }, false)).toBeUndefined();
  });
  it('confirms plain HTTP for no certificate', () => {
    expect(choiceOf(form({ way: 'none' }), false)).toEqual({ kind: 'none', plainOk: true });
  });
  it('sends both paths for a file path, trimmed', () => {
    const v = form({ way: 'file', certFile: ' /etc/a.pem ', keyFile: '/etc/a.key' });
    expect(choiceOf(v, false)).toEqual({
      kind: 'file',
      certFile: '/etc/a.pem',
      keyFile: '/etc/a.key',
    });
    expect(choiceOf({ ...v, keyFile: '' }, false)).toBeUndefined();
  });
  it("sends the Let's Encrypt check of the chosen way, with the agreement only before the first order", () => {
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
    expect(choiceOf(v, false)).toEqual({ kind: 'acme', method: 'dns-01' });
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
  it('tells a port problem, which the manual way avoids', () => {
    expect(isPortError(new ApiError(409, 'acme_port_busy', undefined, {}))).toBe(true);
    expect(isPortError(new ApiError(409, 'acme_port_tls', undefined, {}))).toBe(true);
    expect(isPortError(new ApiError(409, 'acme_busy', undefined, {}))).toBe(false);
  });
});
