import type { AcmeMethod } from '../../lib/api/acme';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import type { CertFacts } from '../../lib/api/rules';

/** The ways a domain gets its certificate (D61); `file` sits under More, for experts. */
export type Way = 'found' | 'upload' | 'paste' | 'none' | 'acme' | 'file';
export const mainWays: readonly Way[] = ['found', 'upload', 'paste', 'none', 'acme'];

/** A certificate picked from those found on this server or stored in Karvan. */
export type FoundPick =
  { kind: 'stored'; certificateId: number } | { kind: 'file'; certFile: string; keyFile: string };

/** The Add domain and Change certificate form as typed. */
export interface DomainFormValues {
  name: string;
  way: Way;
  pick: FoundPick | null;
  certPem: string;
  keyPem: string;
  certFile: string;
  keyFile: string;
  method: AcmeMethod;
  agreeTos: boolean;
}

export function newDomainForm(name = ''): DomainFormValues {
  return {
    name,
    way: 'found',
    pick: null,
    certPem: '',
    keyPem: '',
    certFile: '',
    keyFile: '',
    method: name.startsWith('*.') ? 'dns-01' : 'http-01',
    agreeTos: false,
  };
}

/** The name as the server stores it: trimmed, lower case, no trailing dot. */
export function domainName(raw: string): string {
  return raw.trim().toLowerCase().replace(/\.$/, '');
}

/** Whether a name a certificate carries covers the domain: the same name, or one label under a wildcard. */
export function covers(certName: string, domain: string): boolean {
  const c = domainName(certName);
  const d = domainName(domain);
  if (c === d) return true;
  if (!c.startsWith('*.') || d.startsWith('*.')) return false;
  const dot = d.indexOf('.');
  return dot > 0 && d.slice(dot + 1) === c.slice(2);
}

export interface FoundChoice {
  key: string;
  pick: FoundPick;
  /** The stored certificate's name, or the found file. */
  title: string;
  facts: CertFacts;
}

const coversName = (facts: CertFacts, name: string) => facts.domains.some((n) => covers(n, name));

/** The certificates that cover the name: stored ones first, then found files Karvan does not keep yet. */
export function foundChoices(
  rawName: string,
  stored: readonly Certificate[],
  scan: ScanResult | undefined,
): FoundChoice[] {
  const name = domainName(rawName);
  if (!name) return [];
  const kept = stored
    .filter((c) => coversName(c.facts, name))
    .map((c) => ({
      key: `stored:${String(c.id)}`,
      pick: { kind: 'stored' as const, certificateId: c.id },
      title: c.name,
      facts: c.facts,
    }));
  const isStored = (certFile: string, keyFile: string) =>
    stored.some((c) => c.certFile === certFile && c.keyFile === keyFile);
  const found = (scan?.found ?? [])
    .filter((f) => coversName(f.facts, name) && !isStored(f.certFile, f.keyFile))
    .map((f) => ({
      key: `file:${f.certFile}`,
      pick: { kind: 'file' as const, certFile: f.certFile, keyFile: f.keyFile },
      title: f.certFile,
      facts: f.facts,
    }));
  return [...kept, ...found];
}

/** Let's Encrypt checks a wildcard only by a TXT record (D66). */
export function acmeMethodsFor(name: string): AcmeMethod[] {
  return domainName(name).startsWith('*.') ? ['dns-01'] : ['http-01', 'dns-01'];
}

const both = (a: string, b: string) => a.trim() !== '' && b.trim() !== '';

/** What the form sends, or undefined while the chosen way is not complete. */
export function choiceOf(v: DomainFormValues, needsAgreement: boolean): CertChoice | undefined {
  switch (v.way) {
    case 'found':
      return v.pick ?? undefined;
    case 'upload':
    case 'paste':
      return both(v.certPem, v.keyPem)
        ? { kind: 'pasted', certPem: v.certPem, keyPem: v.keyPem }
        : undefined;
    case 'none':
      return { kind: 'none', plainOk: true };
    case 'file':
      return both(v.certFile, v.keyFile)
        ? { kind: 'file', certFile: v.certFile.trim(), keyFile: v.keyFile.trim() }
        : undefined;
    case 'acme':
      if (!acmeMethodsFor(v.name).includes(v.method)) return undefined;
      if (!needsAgreement) return { kind: 'acme', method: v.method };
      return v.agreeTos ? { kind: 'acme', method: v.method, agreeTos: true } : undefined;
  }
}

/** A new domain needs its name; both need a complete way. */
export function canSubmit(v: DomainFormValues, isNew: boolean, needsAgreement: boolean): boolean {
  if (isNew && domainName(v.name) === '') return false;
  return choiceOf(v, needsAgreement) !== undefined;
}

export type DomainField = 'name' | 'certPem' | 'keyPem' | 'certFile' | 'keyFile';
const certFields: readonly DomainField[] = ['certPem', 'keyPem', 'certFile', 'keyFile'];

/** The field a refused answer names, when the form shows it. */
export function refusedDomainField(err: unknown): DomainField | undefined {
  if (!(err instanceof ApiError)) return undefined;
  const f = err.detail.field;
  if (err.code === 'domain_invalid' && f === 'name') return 'name';
  if (err.code !== 'certificate_invalid') return undefined;
  return certFields.find((x) => x === f);
}

/** The challenge port could not be used; the manual way (DNS) needs no port. */
export function isPortError(err: unknown): boolean {
  return err instanceof ApiError && err.code.startsWith('acme_port_');
}
