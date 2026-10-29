import { describe, expect, it } from 'vitest';
import type { Domain } from '../../lib/api/domains';
import type { ScanResult } from '../../lib/api/certificates';
import {
  daysLeft,
  domainLines,
  certStateOf,
  domainStateOf,
  domainWarningTone,
  notScanned,
  renewalOf,
  renewMethod,
  sortedWarnings,
} from './cert-view';

describe('domain warnings', () => {
  it('put what stops serving first, then what needs a look', () => {
    expect(sortedWarnings(['expires_soon', 'no_certificate', 'expired', 'cert_conflict'])).toEqual([
      'expired',
      'no_certificate',
      'cert_conflict',
      'expires_soon',
    ]);
  });

  it('read a missing or broken certificate as danger and the rest as warnings', () => {
    expect(domainWarningTone('expired')).toBe('danger');
    expect(domainWarningTone('unreadable')).toBe('danger');
    expect(domainWarningTone('no_certificate')).toBe('warning');
    expect(domainWarningTone('cert_conflict')).toBe('warning');
    expect(domainWarningTone('domain_mismatch')).toBe('warning');
    expect(domainWarningTone('expires_soon')).toBe('warning');
  });
});

describe('days left on a certificate', () => {
  const now = new Date('2026-10-28T12:00:00Z');

  it('counts a part of a day as a whole one', () => {
    expect(daysLeft('2026-11-04T10:00:32Z', now)).toBe(7);
    expect(daysLeft('2026-10-28T13:00:00Z', now)).toBe(1);
  });

  it('is zero once it has ended, and missing without an end', () => {
    expect(daysLeft('2026-10-01T00:00:00Z', now)).toBe(0);
    expect(daysLeft(undefined, now)).toBeUndefined();
  });
});

describe("a certificate's state", () => {
  it('is Active with no warning, and while it only ends soon', () => {
    expect(certStateOf([]).state).toBe('active');
    expect(certStateOf(['expires_soon']).state).toBe('active');
  });
  it('needs a look when it misses a domain, and does not serve on a danger', () => {
    expect(certStateOf(['domain_mismatch'])).toEqual({ state: 'attention', tone: 'warning' });
    expect(certStateOf(['domain_mismatch', 'no_key']).state).toBe('broken');
  });
});

describe('the places not scanned', () => {
  const result: ScanResult = {
    found: [],
    skipped: [
      { place: '/var/lib/marzban/certs', reason: 'not_found' },
      { place: '/', reason: 'not_allowed' },
      { place: '/var/lib/marzban/certs', reason: 'unreadable' },
    ],
  };
  it('list each place once, whatever the reason', () => {
    expect(notScanned(result)).toEqual(['/var/lib/marzban/certs', '/']);
  });
});

const domain = (patch: Partial<Domain>): Domain => ({
  id: 9,
  name: 'sub.example.com',
  certificate: null,
  certificates: [],
  pins: [],
  acme: null,
  rules: [],
  warnings: [],
  ...patch,
});
const now = new Date('2026-10-28T12:00:00Z');

describe('the lines of a domain', () => {
  it('turn a conflict into one line per pinned port', () => {
    const d = domain({
      warnings: ['cert_conflict'],
      pins: [
        { port: 8443, certificate: { id: 2, name: 'old-cert' } },
        { port: 2096, certificate: { id: 3, name: 'cdn' } },
      ],
    });
    expect(domainLines(d, now)).toEqual([
      { key: 'pin:8443', word: 'pin', tone: 'warning', params: { port: 8443, cert: 'old-cert' } },
      { key: 'pin:2096', word: 'pin', tone: 'warning', params: { port: 2096, cert: 'cdn' } },
    ]);
  });
  it('count the days left on a certificate that ends soon', () => {
    const d = domain({
      warnings: ['expires_soon'],
      certificate: {
        id: 1,
        name: 'le',
        origin: '',
        facts: { domains: [], issuer: '', notAfter: '2026-11-04T10:00:00Z', warnings: [] },
      },
    });
    expect(domainLines(d, now)).toEqual([
      { key: 'expires_soon', word: 'expires_soon', tone: 'warning', params: { count: 7 } },
    ]);
  });
  it('carry the code of a failed renewal, worst first', () => {
    const d = domain({
      warnings: ['renewal_failed', 'expired'],
      acme: {
        method: 'http-01',
        nextRenewal: null,
        tries: 2,
        lastError: { code: 'acme_port_busy', detail: '' },
        order: null,
      },
    });
    expect(domainLines(d, now).map((l) => [l.word, l.tone, l.params])).toEqual([
      ['expired', 'danger', {}],
      ['renewal_failed', 'warning', { code: 'acme_port_busy' }],
    ]);
  });
});

describe('the renewal of a domain', () => {
  const acme = { method: 'http-01' as const, tries: 0, lastError: null, order: null };
  it('is the running order first', () => {
    const order = {
      id: 1,
      method: 'dns-01' as const,
      renewal: true,
      state: 'waiting_dns' as const,
      dns: null,
      error: null,
      createdAt: '',
      updatedAt: '',
      expiresAt: null,
    };
    expect(renewalOf(domain({ acme: { ...acme, nextRenewal: null, order } }))).toEqual({
      kind: 'order',
      state: 'waiting_dns',
    });
  });
  it('is the next renewal date otherwise', () => {
    const d = domain({ acme: { ...acme, nextRenewal: '2026-12-01T00:00:00Z' } });
    expect(renewalOf(d)).toEqual({ kind: 'renews', at: new Date('2026-12-01T00:00:00Z') });
  });
  it("is nothing for a domain without Let's Encrypt", () => {
    expect(renewalOf(domain({}))).toBeUndefined();
  });
});
describe('renew now', () => {
  const le = {
    id: 2,
    name: 'a',
    origin: 'acme' as const,
    facts: { domains: [], issuer: '', warnings: [] },
  };
  const acme = {
    method: 'dns-01' as const,
    nextRenewal: null,
    tries: 0,
    lastError: null,
    order: null,
  };
  it("renews a Let's Encrypt certificate by the way it was got", () => {
    expect(renewMethod(domain({ certificate: le, acme }))).toBe('dns-01');
  });
  it('waits while an order runs', () => {
    const order = {
      id: 1,
      method: 'dns-01' as const,
      renewal: true,
      state: 'validating' as const,
      dns: null,
      error: null,
      createdAt: '',
      updatedAt: '',
      expiresAt: null,
    };
    expect(renewMethod(domain({ certificate: le, acme: { ...acme, order } }))).toBeUndefined();
  });
  it('is not offered for another certificate or a domain that never had one', () => {
    expect(renewMethod(domain({ certificate: { ...le, origin: '' }, acme }))).toBeUndefined();
    expect(renewMethod(domain({ certificate: le, acme: { ...acme, method: '' } }))).toBeUndefined();
    expect(renewMethod(domain({}))).toBeUndefined();
  });
});

describe("a domain's state", () => {
  it('is Active with no warning', () => {
    expect(domainStateOf(domain({ warnings: [] }), now).state).toBe('active');
  });
  it('is plain HTTP when its only line is no certificate', () => {
    expect(domainStateOf(domain({ warnings: ['no_certificate'] }), now)).toEqual({
      state: 'plain',
      tone: 'neutral',
    });
  });
  it('stays Active while its certificate only ends soon', () => {
    expect(domainStateOf(domain({ warnings: ['expires_soon'] }), now).state).toBe('active');
  });
  it('needs a look for a warning, and does not serve for a danger', () => {
    expect(domainStateOf(domain({ warnings: ['domain_mismatch'] }), now).state).toBe('attention');
    expect(domainStateOf(domain({ warnings: ['expires_soon', 'expired'] }), now)).toEqual({
      state: 'broken',
      tone: 'danger',
    });
  });
});
