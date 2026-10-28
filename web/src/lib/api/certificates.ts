import { api } from './client';
import type { CertFacts } from './rules';

/** A panel rule or a redirect rule, as the certificate pages name it. */
export interface RuleRef {
  kind: 'panel' | 'redirect';
  id: number;
  name: string;
}

/** A stored certificate (phase 4.1 D49). Its private key never leaves the server. */
export interface Certificate {
  id: number;
  name: string;
  /** Files on this server, or a pasted certificate Karvan keeps. */
  kind: 'file' | 'pasted';
  certFile?: string;
  keyFile?: string;
  /** Only for a pasted one. */
  certPem?: string;
  hasKey: boolean;
  facts: CertFacts;
  /** The rules that use it, each with its domain_mismatch. */
  rules: (RuleRef & { warnings: string[] })[];
}

export const certificateKeys = {
  all: ['certificates'] as const,
};

export function getCertificates(): Promise<{ certificates: Certificate[] }> {
  return api('GET', 'certificates');
}

/** A file pair, or a pasted certificate; an empty `keyPem` on a change keeps the stored key. */
export interface CertificateInput {
  name: string;
  certFile?: string;
  keyFile?: string;
  certPem?: string;
  keyPem?: string;
}

/** Adds a certificate (no id) or replaces one; 400 certificate_invalid names the field. */
export function saveCertificate(input: CertificateInput, id?: number): Promise<Certificate> {
  return id
    ? api('PATCH', `certificates/${String(id)}`, input)
    : api('POST', 'certificates', input);
}

/** 409 certificate_in_use while a rule uses it. */
export function deleteCertificate(id: number): Promise<void> {
  return api('DELETE', `certificates/${String(id)}`);
}

/** A certificate found in a known place, with the paths it was walked by. */
export interface ScanFound {
  place: string;
  certFile: string;
  keyFile: string;
  facts: CertFacts;
  /** A stored certificate has these two paths. */
  added: boolean;
}

export type SkipReason = 'not_found' | 'unreadable' | 'not_allowed' | 'too_many' | 'time';

export interface ScanResult {
  found: ScanFound[];
  skipped: { place: string; reason: SkipReason }[];
}

/** Looks in the server's known places, read only; 409 busy while a scan runs (D51). */
export function scanCertificates(): Promise<ScanResult> {
  return api('POST', 'certificates/scan');
}
