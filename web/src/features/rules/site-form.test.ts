import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { LinkSite } from '../../lib/api/rules';
import {
  canCheckCert,
  certInputOf,
  inputOf,
  isPlainHttp,
  newValues,
  parseDomains,
  refusedField,
  valuesOf,
} from './site-form';

const saved: LinkSite = {
  id: 3,
  name: 'old-443',
  panelId: 2,
  sourceKind: 'pasarguard',
  sourceUrl: 'https://old.example.com:8443',
  oldPath: 'sub',
  domains: ['a.example.com', 'b.example.com'],
  port: 443,
  certPem: 'CERT',
  hasKey: true,
  lazy: true,
  plainHttpOk: false,
  https: true,
};

describe('the site form', () => {
  it('starts a new site from a migration with HTTPS on 443 and lazy checks on', () => {
    const v = newValues({ panelId: 2, sourceKind: 'pasarguard', sourceUrl: 'https://old' }, 1);
    expect(v).toMatchObject({ panelId: 2, sourceKind: 'pasarguard', sourceUrl: 'https://old' });
    expect(v).toMatchObject({ oldPath: 'sub', port: '443', certMode: 'files', lazy: true });
    expect(newValues({}, 1).panelId).toBe(1);
  });

  it('opens a saved site with its key field empty', () => {
    const v = valuesOf(saved);
    expect(v).toMatchObject({ certMode: 'paste', certPem: 'CERT', keyPem: '' });
    expect(v.domains).toBe('a.example.com\nb.example.com');
    expect(valuesOf({ ...saved, certPem: undefined, domains: null }).certMode).toBe('none');
  });

  it('sends only the chosen way of getting a certificate', () => {
    const v = { ...valuesOf(saved), certFile: '/c.pem', keyFile: '/k.pem' };
    expect(inputOf(v)).toMatchObject({ certPem: 'CERT' });
    expect(inputOf(v)).not.toHaveProperty('keyPem');
    expect(inputOf(v)).not.toHaveProperty('certFile');
    expect(inputOf({ ...v, keyPem: 'KEY' })).toMatchObject({ certPem: 'CERT', keyPem: 'KEY' });
    const files = inputOf({ ...v, certMode: 'files' });
    expect(files).toMatchObject({ certFile: '/c.pem', keyFile: '/k.pem' });
    expect(files).not.toHaveProperty('certPem');
    expect(inputOf({ ...v, certMode: 'none' })).not.toHaveProperty('certPem');
  });

  it('reads the port as a number and the domains as a list', () => {
    const input = inputOf({ ...valuesOf(saved), port: '8443', domains: ' a.x, b.x\n\nc.x ' });
    expect(input.port).toBe(8443);
    expect(input.domains).toEqual(['a.x', 'b.x', 'c.x']);
    expect(parseDomains('')).toEqual([]);
  });

  it('accepts a plain http old panel only for an http address', () => {
    expect(isPlainHttp(' HTTP://old')).toBe(true);
    expect(isPlainHttp('https://old')).toBe(false);
    const v = { ...valuesOf(saved), plainHttpOk: true };
    expect(inputOf(v).plainHttpOk).toBe(false);
    expect(inputOf({ ...v, sourceUrl: 'http://old' }).plainHttpOk).toBe(true);
  });

  it('checks a certificate with the saved key of a site being changed', () => {
    const v = valuesOf(saved);
    expect(certInputOf(v, 3)).toEqual({
      id: 3,
      certPem: 'CERT',
      domains: ['a.example.com', 'b.example.com'],
    });
    expect(certInputOf(v)).not.toHaveProperty('id');
    expect(canCheckCert(v)).toBe(true);
    expect(canCheckCert({ ...v, certMode: 'files', certFile: '/c' })).toBe(false);
    expect(canCheckCert({ ...v, certMode: 'none' })).toBe(false);
  });

  it('finds the field a refused save names', () => {
    expect(refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'keyPem' }))).toBe(
      'keyPem',
    );
    expect(
      refusedField(new ApiError(400, 'site_invalid', undefined, { field: 'other' })),
    ).toBeUndefined();
    expect(refusedField(new ApiError(409, 'name_taken'))).toBeUndefined();
  });
});
