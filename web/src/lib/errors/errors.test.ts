import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../i18n/core';
import { ApiError, offline } from '../api/client';
import { errorKey, errorMessages, errorOf, errorText, knownCodes } from './errors';

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
