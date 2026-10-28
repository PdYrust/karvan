import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import {
  acmeMethodsFor,
  canSubmit,
  choiceOf,
  covers,
  domainName,
  foundChoices,
  isPortError,
  newDomainForm,
  refusedDomainField,
  type DomainFormValues,
} from './domain-form';

const facts = (domains: string[]) => ({ domains, issuer: 'R11', warnings: [] });
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
  it('lists the stored ones that cover the name first, then the found ones', () => {
    const list = foundChoices('Sub.example.com', stored, scan);
    expect(list.map((c) => c.pick)).toEqual([
      { kind: 'stored', certificateId: 4 },
      { kind: 'file', certFile: '/root/cert/sub.pem', keyFile: '/root/cert/sub.key' },
    ]);
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
  it('asks for the agreement only before the first order', () => {
    const v = form({ way: 'acme', method: 'dns-01' });
    expect(choiceOf(v, true)).toBeUndefined();
    expect(choiceOf({ ...v, agreeTos: true }, true)).toEqual({
      kind: 'acme',
      method: 'dns-01',
      agreeTos: true,
    });
    expect(choiceOf(v, false)).toEqual({ kind: 'acme', method: 'dns-01' });
  });
  it('never sends the automatic way for a wildcard', () => {
    expect(choiceOf(form({ name: '*.example.com', way: 'acme' }), false)).toBeUndefined();
  });
});

describe('submitting', () => {
  it('needs a name for a new domain and a complete way', () => {
    expect(canSubmit(form({ way: 'none' }), true, false)).toBe(true);
    expect(canSubmit(form({ way: 'none', name: ' ' }), true, false)).toBe(false);
    expect(canSubmit(form({ way: 'paste' }), true, false)).toBe(false);
  });
  it('needs no name when the domain is there already', () => {
    expect(canSubmit(form({ way: 'none', name: '' }), false, false)).toBe(true);
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
