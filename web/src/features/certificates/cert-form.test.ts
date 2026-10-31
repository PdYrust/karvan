import { describe, expect, it } from 'vitest';
import { ApiError } from '../../lib/api/client';
import type { Certificate } from '../../lib/api/certificates';
import {
  canSaveCert,
  certFormOf,
  certificateInputOf,
  refusedCertField,
  type CertFormValues,
} from './cert-form';

const facts = { domains: [], issuer: '', warnings: [] };
const blank: CertFormValues = {
  name: '',
  certMode: 'files',
  certFile: '',
  keyFile: '',
  certPem: '',
  keyPem: '',
};

describe('the certificate form', () => {
  it('opens a stored file pair with its paths', () => {
    const c: Certificate = {
      id: 1,
      name: 'le',
      kind: 'file',
      certFile: '/etc/a.pem',
      keyFile: '/etc/a.key',
      hasKey: true,
      origin: '',
      facts,
      domains: [],
      rules: [],
      found: false,
      copies: [],
    };
    expect(certFormOf(c)).toMatchObject({
      name: 'le',
      certMode: 'files',
      certFile: '/etc/a.pem',
      keyFile: '/etc/a.key',
    });
  });

  it('opens a pasted one with its certificate and an empty key', () => {
    const c: Certificate = {
      id: 2,
      name: 'own',
      kind: 'pasted',
      certPem: 'CERT',
      hasKey: true,
      origin: '',
      facts,
      domains: [],
      rules: [],
      found: false,
      copies: [],
    };
    expect(certFormOf(c)).toMatchObject({ certMode: 'paste', certPem: 'CERT', keyPem: '' });
  });

  it('sends only the chosen way, paths trimmed', () => {
    const v = { ...blank, name: ' le ', certFile: ' /etc/a.pem ', keyFile: '/etc/a.key' };
    expect(certificateInputOf(v)).toEqual({
      name: 'le',
      certFile: '/etc/a.pem',
      keyFile: '/etc/a.key',
    });
  });

  it('sends an uploaded pair as pasted text', () => {
    const v = {
      ...blank,
      name: 'up',
      certMode: 'upload' as const,
      certPem: 'C',
      keyPem: 'K',
    };
    expect(certificateInputOf(v)).toEqual({ name: 'up', certPem: 'C', keyPem: 'K' });
  });

  it('needs a name and both parts, but a change may keep the stored key', () => {
    const paste = { ...blank, name: 'own', certMode: 'paste' as const, certPem: 'C' };
    expect(canSaveCert(paste, false)).toBe(false);
    expect(canSaveCert(paste, true)).toBe(true);
    expect(canSaveCert({ ...paste, name: ' ' }, true)).toBe(false);
    expect(canSaveCert({ ...blank, name: 'le', certFile: '/a' }, false)).toBe(false);
  });

  it('names the field the server refused', () => {
    const err = new ApiError(400, 'certificate_invalid', undefined, {
      field: 'keyFile',
      reason: 'unreadable',
    });
    expect(refusedCertField(err)).toBe('keyFile');
    expect(
      refusedCertField(new ApiError(400, 'certificate_invalid', undefined, { field: 'x' })),
    ).toBe(undefined);
    expect(refusedCertField(new ApiError(409, 'name_taken'))).toBe(undefined);
  });
});
