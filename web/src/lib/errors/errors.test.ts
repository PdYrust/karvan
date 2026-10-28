import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../i18n/core';
import { ApiError, offline } from '../api/client';
import { batchCodes, batchErrorMessages, batchErrorText } from './batches';
import { errorKey, errorMessages, errorOf, errorText, knownCodes } from './errors';
import {
  subscriptionCodes,
  subscriptionErrorKey,
  subscriptionErrorMessages,
  subscriptionErrorText,
} from './subscriptions';

const en = createTranslate(errorMessages.en, 'en');
const fa = createTranslate(errorMessages.fa, 'fa');

describe('errorKey', () => {
  it('gives every code the server sends a sentence of its own', () => {
    for (const code of knownCodes) {
      const key = errorKey(code);
      expect(key, code).not.toBe('unknown');
      expect(errorMessages.en[key], code).toBeTruthy();
      expect(errorMessages.fa[key], code).toBeTruthy();
    }
  });

  it('falls back to a general sentence for a code it does not know', () => {
    expect(errorKey('something_new')).toBe('unknown');
    expect(errorKey('')).toBe('unknown');
  });

  it('keeps an unknown old panel failure about the old panel', () => {
    expect(errorKey('source_brand_new')).toBe('source_unknown');
  });

  it('names the password rule that failed', () => {
    expect(errorKey('weak_password', { rule: 'password_short' })).toBe('weak_password_short');
    expect(errorKey('weak_password', { rule: 'password_has_name' })).toBe('weak_password_has_name');
    expect(errorKey('weak_password', { rule: 'later_rule' })).toBe('weak_password');
    expect(errorKey('weak_password')).toBe('weak_password');
  });

  it('does not read inherited object keys as codes', () => {
    expect(errorKey('constructor')).toBe('unknown');
    expect(errorKey('toString')).toBe('unknown');
  });
});

describe('area word lists', () => {
  it('give each Batches page code its own sentence in both languages, once', () => {
    for (const code of batchCodes) {
      expect(knownCodes, code).not.toContain(code);
      expect(batchErrorMessages.en[code], code).toBeTruthy();
      expect(batchErrorMessages.fa[code], code).toBeTruthy();
    }
    expect([...batchCodes].sort()).toEqual(
      ['batch_active', 'batch_in_use', 'batch_invalid', 'not_reactivatable'].sort(),
    );
  });

  it('give each subscriptions code its own sentence in both languages, once', () => {
    for (const code of subscriptionCodes) {
      expect(knownCodes, code).not.toContain(code);
      expect(batchCodes as readonly string[], code).not.toContain(code);
      expect(subscriptionErrorMessages.en[code], code).toBeTruthy();
      expect(subscriptionErrorMessages.fa[code], code).toBeTruthy();
    }
    expect([...subscriptionCodes].sort()).toEqual(
      [
        'certificate_in_use',
        'certificate_invalid',
        'certificate_not_adopted',
        'domain_in_use',
        'domain_invalid',
        'plain_http_off',
        'port_mixed',
        'site_invalid',
        'site_overlap',
      ].sort(),
    );
  });

  it('name why a certificate was refused', () => {
    expect(subscriptionErrorKey('certificate_invalid', { reason: 'unreadable' })).toBe(
      'certificate_invalid_unreadable',
    );
    expect(subscriptionErrorKey('certificate_invalid', { reason: 'key_mismatch' })).toBe(
      'certificate_invalid_key_mismatch',
    );
    expect(subscriptionErrorKey('certificate_invalid', { reason: 'format' })).toBe(
      'certificate_invalid',
    );
    expect(subscriptionErrorKey('certificate_invalid', { reason: 'constructor' })).toBe(
      'certificate_invalid',
    );
    expect(subscriptionErrorKey('certificate_invalid_no_cert')).toBeUndefined();
    expect(subscriptionErrorKey('site_invalid', { field: 'batchIds', reason: 'mixed' })).toBe(
      'site_invalid_mixed',
    );
    expect(subscriptionErrorKey('site_invalid', { field: 'batchIds', reason: 'inactive' })).toBe(
      'site_invalid_inactive',
    );
    expect(subscriptionErrorKey('site_invalid', { field: 'batchIds', reason: 'unknown' })).toBe(
      'site_invalid_unknown',
    );
    expect(subscriptionErrorKey('site_invalid', { field: 'port' })).toBe('site_invalid');
    expect(subscriptionErrorKey('site_invalid', { reason: 'no_cert' })).toBe('site_invalid');
    expect(subscriptionErrorKey('site_invalid_mixed')).toBeUndefined();
    for (const lang of ['en', 'fa'] as const) {
      for (const [key, text] of Object.entries(subscriptionErrorMessages[lang])) {
        expect(text, key).toBeTruthy();
      }
    }
  });

  it('use the subscriptions sentence for its codes and the shared one for the rest', () => {
    const ts = createTranslate(subscriptionErrorMessages.en, 'en');
    const shared = (err: unknown) => errorText(en, err);
    const refused = new ApiError(400, 'certificate_invalid', undefined, { reason: 'no_cert' });
    expect(subscriptionErrorText(ts, shared, refused)).toBe(ts('certificate_invalid_no_cert'));
    expect(subscriptionErrorText(ts, shared, new ApiError(409, 'site_overlap'))).toBe(
      ts('site_overlap'),
    );
    expect(subscriptionErrorText(ts, shared, new ApiError(409, 'name_taken'))).toBe(
      en('name_taken'),
    );
  });

  it('use the area sentence for its codes and the shared one for the rest', () => {
    const tb = createTranslate(batchErrorMessages.en, 'en');
    const shared = (err: unknown) => errorText(en, err);
    expect(batchErrorText(tb, shared, new ApiError(409, 'batch_active'))).toBe(tb('batch_active'));
    expect(batchErrorText(tb, shared, new ApiError(409, 'busy'))).toBe(en('busy'));
    expect(batchErrorText(tb, shared, new ApiError(409, 'constructor'))).toBe(en('unknown'));
  });
});

describe('errorOf', () => {
  it('reads the code and the extras of an API error', () => {
    const err = new ApiError(502, 'source_tls_untrusted', undefined, { sha256: 'ab' });
    expect(errorOf(err)).toEqual({ code: 'source_tls_untrusted', detail: { sha256: 'ab' } });
  });

  it('counts anything else as a server failure', () => {
    expect(errorOf(new Error('boom'))).toEqual({ code: 'internal', detail: {} });
    expect(errorOf('text')).toEqual({ code: 'internal', detail: {} });
  });
});

describe('errorText', () => {
  it('says what happened in both languages, never the raw code', () => {
    for (const t of [en, fa]) {
      const text = errorText(t, new ApiError(502, 'source_unauthorized'));
      expect(text).not.toContain('source_unauthorized');
      expect(text.length).toBeGreaterThan(10);
    }
  });

  it('tells an unreachable network apart from a server failure', () => {
    expect(errorText(en, new ApiError(0, offline))).not.toBe(
      errorText(en, new ApiError(500, 'internal')),
    );
  });

  it('reads a code from the event stream the same way', () => {
    expect(errorText(en, { code: 'cancelled', detail: {} })).toBe(en('cancelled'));
    expect(errorText(en, { code: 'no_such_code', detail: {} })).toBe(en('unknown'));
  });
});
