import { describe, expect, it } from 'vitest';
import { createTranslate } from '../../i18n/core';
import { ApiError } from '../api/client';
import { knownCodes } from './errors';
import { httpsCodes, httpsErrorKey, httpsErrorMessages, httpsErrorText } from './https';

describe('HTTPS words', () => {
  it('give each code a sentence in both languages', () => {
    for (const code of httpsCodes) {
      expect(knownCodes, code).not.toContain(code);
      const key = httpsErrorKey(code) ?? 'https_invalid';
      expect(httpsErrorKey(code), code).toBe(key);
      expect(httpsErrorMessages.en[key], code).toBeTruthy();
      expect(httpsErrorMessages.fa[key], code).toBeTruthy();
    }
  });

  it('name why a domain was refused, and nothing else', () => {
    const domain = (reason: string) =>
      httpsErrorKey('https_invalid', { field: 'domainId', reason });
    expect(domain('unknown')).toBe('https_invalid_unknown');
    expect(domain('no_certificate')).toBe('https_invalid_no_certificate');
    expect(domain('constructor')).toBe('https_invalid');
    expect(httpsErrorKey('https_invalid', { field: 'tls', reason: 'unknown' })).toBe(
      'https_invalid',
    );
    expect(httpsErrorKey('port_mixed')).toBe('https_port_mixed');
    expect(httpsErrorKey('https_port_mixed')).toBeUndefined();
    expect(httpsErrorKey('site_overlap')).toBeUndefined();
  });

  it('leave other codes to the shared sentences', () => {
    const t = createTranslate(httpsErrorMessages.en, 'en');
    const shared = () => 'shared';
    expect(httpsErrorText(t, shared, new ApiError(409, 'port_busy'))).toBe(
      httpsErrorMessages.en.port_busy,
    );
    expect(httpsErrorText(t, shared, new ApiError(500, 'internal'))).toBe('shared');
  });
});
