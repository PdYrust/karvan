import { ApiError } from '../../lib/api/client';
import type { Certificate, CertificateInput } from '../../lib/api/certificates';
import type { CertMode, CertValues } from './CertFields';

/** A stored certificate always has one: files, or text read from files or pasted. */
export const certModes: readonly CertMode[] = ['files', 'upload', 'paste'];

export type CertFormValues = CertValues & { name: string };

const certFormFields = ['name', 'certFile', 'keyFile', 'certPem', 'keyPem'] as const;
export type CertFormField = (typeof certFormFields)[number];

/** A stored certificate in the form. The key never comes back, so its field starts empty. */
export function certFormOf(c: Certificate): CertFormValues {
  return {
    name: c.name,
    certMode: c.kind === 'file' ? 'files' : 'paste',
    certFile: c.certFile ?? '',
    keyFile: c.keyFile ?? '',
    certPem: c.certPem ?? '',
    keyPem: '',
  };
}

/** Only the chosen way's fields; upload sends the text it read, as a paste does. */
export function certificateInputOf(v: CertFormValues): CertificateInput {
  const name = v.name.trim();
  if (v.certMode === 'files') {
    return { name, certFile: v.certFile.trim(), keyFile: v.keyFile.trim() };
  }
  return { name, certPem: v.certPem, keyPem: v.keyPem };
}

/** A name and both parts; a change may leave the key empty to keep the stored one. */
export function canSaveCert(v: CertFormValues, keepsKey: boolean): boolean {
  if (!v.name.trim()) return false;
  if (v.certMode === 'files') return v.certFile.trim() !== '' && v.keyFile.trim() !== '';
  return v.certPem.trim() !== '' && (keepsKey || v.keyPem.trim() !== '');
}

/** The field a refused save names, when the form shows it. */
export function refusedCertField(err: unknown): CertFormField | undefined {
  if (!(err instanceof ApiError) || err.code !== 'certificate_invalid') return undefined;
  const f = err.detail.field;
  return certFormFields.find((x) => x === f);
}
