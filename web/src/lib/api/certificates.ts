import { api } from './client';
import { factsOf, list } from './lists';
import type { CertFacts } from './rules';

/** A panel rule or a redirect rule, as the certificate pages name it. */
export interface RuleRef {
  kind: 'panel' | 'redirect';
  id: number;
  name: string;
}

/** A stored certificate. Its private key never leaves the server. */
export interface Certificate {
  id: number;
  name: string;
  /** Files on this server, or a pasted certificate Karvan keeps. */
  kind: 'file' | 'pasted';
  /** `acme` for one Karvan got from Let's Encrypt and renews itself. */
  origin: '' | 'acme';
  certFile?: string;
  keyFile?: string;
  /** Only for a pasted one. */
  certPem?: string;
  hasKey: boolean;
  facts: CertFacts;
  /** The domains whose certificate it is, each with its domain_mismatch. */
  domains: { id: number; name: string; warnings: string[] }[];
  /** The rules that use it, each with its domain_mismatch. */
  rules: (RuleRef & { warnings: string[] })[];
}

export const certificateKeys = {
  all: ['certificates'] as const,
};

function certificateOf(c: Certificate): Certificate {
  return {
    ...c,
    facts: factsOf(c.facts),
    domains: list(c.domains).map((d) => ({ ...d, warnings: list(d.warnings) })),
    rules: list(c.rules).map((r) => ({ ...r, warnings: list(r.warnings) })),
  };
}

export async function getCertificates(): Promise<{ certificates: Certificate[] }> {
  const r = await api<{ certificates: Certificate[] | null }>('GET', 'certificates');
  return { certificates: list(r.certificates).map(certificateOf) };
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
export async function saveCertificate(input: CertificateInput, id?: number): Promise<Certificate> {
  const c = await (id
    ? api<Certificate>('PATCH', `certificates/${String(id)}`, input)
    : api<Certificate>('POST', 'certificates', input));
  return certificateOf(c);
}

/** 409 certificate_in_use while a domain, a pin or a rule uses it. */
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

/** Looks in the server's known places, read only; 409 busy while a scan runs. */
export async function scanCertificates(): Promise<ScanResult> {
  const r = await api<ScanResult>('POST', 'certificates/scan');
  return {
    found: list(r.found).map((f) => ({ ...f, facts: factsOf(f.facts) })),
    skipped: list(r.skipped),
  };
}
