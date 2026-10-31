import { api } from './client';
import type { PanelKind } from './panels';
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
  /** Files in one of the places the scan reads. */
  found: boolean;
  /** The folders Karvan keeps it copied to. */
  copies: CertCopy[];
}

/** Why a copy was refused or failed. */
export type CopyReason =
  | 'not_found'
  | 'link'
  | 'outside'
  | 'foreign_file'
  | 'taken'
  | 'no_key'
  | 'no_name'
  | 'write_failed';

/** Where a certificate is copied: Let's Encrypt live, or a panel's certificate folder. */
export type CopyTarget = { target: 'letsencrypt' } | { target: 'panel'; panelId: number };

/** A folder Karvan keeps a certificate copied to, again after each change. */
export interface CertCopy {
  id: number;
  target: 'letsencrypt' | 'panel';
  panel: { id: number; name: string; kind: PanelKind | '' } | null;
  dir: string;
  certFile: string;
  keyFile: string;
  copiedAt: string | null;
  /** The last copy failed. */
  error: { reason: CopyReason; path: string } | null;
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
    found: c.found === true,
    copies: list(c.copies),
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

/**
 * Copies fullchain.pem and privkey.pem into a new folder named after the main domain, and again
 * after each change; 409 copy_refused names the reason and the path.
 */
export function copyCertificate(id: number, target: CopyTarget): Promise<CertCopy> {
  return api('POST', `certificates/${String(id)}/copies`, target);
}

/** Stops copying there; the files stay. */
export function stopCertificateCopy(id: number, copyId: number): Promise<void> {
  return api('DELETE', `certificates/${String(id)}/copies/${String(copyId)}`);
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
