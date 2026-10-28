import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../i18n/core';
import { ApiError } from '../api/client';
import { acmeCodes, acmeErrorKey, acmeErrorMessages, acmeErrorText } from './acme';
import { batchCodes } from './batches';
import { knownCodes } from './errors';
import { subscriptionCodes, subscriptionErrorKey } from './subscriptions';

describe("Let's Encrypt words", () => {
  it('give each code its own sentence in both languages, once', () => {
    for (const code of acmeCodes) {
      expect(knownCodes, code).not.toContain(code);
      expect(batchCodes as readonly string[], code).not.toContain(code);
      expect(subscriptionCodes as readonly string[], code).not.toContain(code);
      expect(acmeErrorMessages.en[code], code).toBeTruthy();
      expect(acmeErrorMessages.fa[code], code).toBeTruthy();
    }
    expect([...acmeCodes].sort()).toEqual(
      [
        'acme_account',
        'acme_busy',
        'acme_caa',
        'acme_challenge_failed',
        'acme_dns_not_found',
        'acme_expired',
        'acme_failed',
        'acme_invalid',
        'acme_port_busy',
        'acme_port_denied',
        'acme_port_tls',
        'acme_rate_limited',
        'acme_rejected',
        'acme_state',
        'acme_unreachable',
      ].sort(),
    );
  });

  it('name why a request was refused, by reason and then by field', () => {
    expect(acmeErrorKey('acme_invalid', { field: 'method', reason: 'wildcard_needs_dns' })).toBe(
      'acme_invalid_wildcard_needs_dns',
    );
    expect(acmeErrorKey('acme_invalid', { field: 'name', reason: 'ip' })).toBe('acme_invalid_ip');
    expect(acmeErrorKey('acme_invalid', { field: 'agreeTos' })).toBe('acme_invalid_agreeTos');
    expect(acmeErrorKey('acme_invalid', { field: 'email' })).toBe('acme_invalid_email');
    expect(acmeErrorKey('acme_invalid', { field: 'method' })).toBe('acme_invalid');
    expect(acmeErrorKey('acme_invalid', { reason: 'constructor' })).toBe('acme_invalid');
    expect(acmeErrorKey('acme_invalid_ip')).toBeUndefined();
    expect(acmeErrorKey('site_overlap')).toBeUndefined();
    expect(
      subscriptionErrorKey('redirect_invalid', { field: 'fromPath', reason: 'reserved' }),
    ).toBe('redirect_invalid_reserved');
  });

  it('fall back to the other sentences for other codes', () => {
    const t = createTranslate(acmeErrorMessages.en, 'en');
    const other = () => 'other';
    expect(acmeErrorText(t, other, new ApiError(409, 'acme_port_busy'))).toBe(
      'Port 80 is used by another program. Use the manual way (DNS).',
    );
    expect(acmeErrorText(t, other, new ApiError(409, 'site_overlap'))).toBe('other');
  });
});
