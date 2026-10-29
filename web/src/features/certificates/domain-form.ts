import type { AcmeMethod } from '../../lib/api/acme';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import type { CertFacts, CertWarning } from '../../lib/api/rules';

/**
 * The ways a domain gets its certificate (D61): one on this server, a new one
 * from Let's Encrypt by either check, uploaded or pasted text, typed file paths,
 * or none.
 */
export type Way = 'found' | 'acme-http' | 'acme-dns' | 'paste' | 'file' | 'none';

/**
 * The steps, one question each: the name, with or without a certificate,
 * which certificate, then what that way needs.
 */
export type Step = 'domain' | 'https' | 'source' | 'details';
export type Https = 'with' | 'without';
export type Source = 'acme-http' | 'acme-dns' | 'own';

/** A certificate picked from those on this server or stored in Karvan. */
export type FoundPick =
  { kind: 'stored'; certificateId: number } | { kind: 'file'; certFile: string; keyFile: string };

/** The Add domain and Change certificate form as typed. */
export interface DomainFormValues {
  name: string;
  https: Https;
  way: Way;
  pick: FoundPick | null;
  certPem: string;
  keyPem: string;
  certFile: string;
  keyFile: string;
  agreeTos: boolean;
}

export function newDomainForm(name = ''): DomainFormValues {
  return {
    name,
    https: 'with',
    way: 'acme-http',
    pick: null,
    certPem: '',
    keyPem: '',
    certFile: '',
    keyFile: '',
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

/** The steps a form walks: Change certificate starts at `https`; no certificate skips `source`. */
export function stepsFor(isNew: boolean, https: Https): Step[] {
  const steps: Step[] = isNew ? ['domain', 'https'] : ['https'];
  return https === 'without' ? [...steps, 'details'] : [...steps, 'source', 'details'];
}

/** Continue on the first step needs a name. */
export function stepAfter(name: string): Step {
  return domainName(name) ? 'https' : 'domain';
}

export interface ServerCert {
  key: string;
  pick: FoundPick;
  /** The stored certificate's name, or the found file. */
  title: string;
  facts: CertFacts;
  /** Whether it covers the name; one that does not is shown but cannot be chosen. */
  covers: boolean;
}

/** 0 when a name the certificate carries is the domain itself, 1 for a wildcard, 2 for neither. */
function matchRank(facts: CertFacts, name: string): number {
  if (facts.domains.some((n) => domainName(n) === name)) return 0;
  return facts.domains.some((n) => covers(n, name)) ? 1 : 2;
}

/**
 * The certificates on this server no domain uses yet (on the change page the
 * domain's own is left out too): stored ones and found files Karvan does not
 * keep yet. Exact names first, then wildcards, then those that do not cover it.
 */
export function serverCerts(
  rawName: string,
  stored: readonly Certificate[],
  scan: ScanResult | undefined,
): ServerCert[] {
  const name = domainName(rawName);
  if (!name) return [];
  const kept = stored
    .filter((c) => c.domains.length === 0)
    .map((c) => ({
      key: `stored:${String(c.id)}`,
      pick: { kind: 'stored' as const, certificateId: c.id },
      title: c.name,
      facts: c.facts,
    }));
  const isStored = (certFile: string, keyFile: string) =>
    stored.some((c) => c.certFile === certFile && c.keyFile === keyFile);
  const found = (scan?.found ?? [])
    .filter((f) => !isStored(f.certFile, f.keyFile))
    .map((f) => ({
      key: `file:${f.certFile}`,
      pick: { kind: 'file' as const, certFile: f.certFile, keyFile: f.keyFile },
      title: f.certFile,
      facts: f.facts,
    }));
  const all = [...kept, ...found].map((c) => ({ c, rank: matchRank(c.facts, name) }));
  return all
    .sort((a, b) => a.rank - b.rank)
    .map(({ c, rank }): ServerCert => ({ ...c, covers: rank < 2 }));
}

const cannotServe: ReadonlySet<CertWarning> = new Set([
  'unreadable',
  'no_key',
  'key_mismatch',
  'expired',
  'not_yet_valid',
]);

const endOf = (f: CertFacts) => (f.notAfter ? new Date(f.notAfter).getTime() : 0);

/** Whether a certificate can serve now: readable, with its key, and in its dates. */
export function canServe(facts: CertFacts, now: Date): boolean {
  if (facts.warnings.some((w) => cannotServe.has(w))) return false;
  return endOf(facts) > now.getTime();
}

/** The one to choose: it covers the name, can serve and ends last; on a tie the earlier. */
export function bestCert(list: readonly ServerCert[], now: Date): ServerCert | undefined {
  let best: ServerCert | undefined;
  for (const c of list) {
    if (!c.covers || !canServe(c.facts, now)) continue;
    if (!best || endOf(c.facts) > endOf(best.facts)) best = c;
  }
  return best;
}

/** Let's Encrypt checks a wildcard only by a TXT record (D66). */
export function acmeMethodsFor(name: string): AcmeMethod[] {
  return domainName(name).startsWith('*.') ? ['dns-01'] : ['http-01', 'dns-01'];
}

export const isWildcard = (name: string) => !acmeMethodsFor(name).includes('http-01');

/** The source marked Recommended: a certificate this server has, else Let's Encrypt. */
export function recommendedSource(name: string, best: ServerCert | undefined): Source {
  if (best) return 'own';
  return isWildcard(name) ? 'acme-dns' : 'acme-http';
}

/**
 * The way a picked source opens with: the best certificate on this server for
 * a certificate the operator has, else upload or paste.
 */
export function wayFor(
  source: Source,
  best: ServerCert | undefined,
): Pick<DomainFormValues, 'way' | 'pick'> {
  if (source !== 'own') return { way: source, pick: null };
  return best ? { way: 'found', pick: best.pick } : { way: 'paste', pick: null };
}

/** What the chosen way still needs before Add domain, if anything. */
export type Missing = 'pick' | 'pem' | 'files' | 'agree' | 'wildcard';

const filled = (s: string) => s.trim() !== '';

export function missingOf(v: DomainFormValues, needsAgreement: boolean): Missing | undefined {
  switch (v.way) {
    case 'found':
      return v.pick ? undefined : 'pick';
    case 'paste':
      return filled(v.certPem) && filled(v.keyPem) ? undefined : 'pem';
    case 'file':
      return filled(v.certFile) && filled(v.keyFile) ? undefined : 'files';
    case 'acme-http':
    case 'acme-dns':
      if (v.way === 'acme-http' && isWildcard(v.name)) return 'wildcard';
      return needsAgreement && !v.agreeTos ? 'agree' : undefined;
    case 'none':
      return undefined;
  }
}

/** What the form sends, or undefined while the chosen way is not complete. */
export function choiceOf(v: DomainFormValues, needsAgreement: boolean): CertChoice | undefined {
  if (missingOf(v, needsAgreement)) return undefined;
  switch (v.way) {
    case 'found':
      return v.pick ?? undefined;
    case 'paste':
      return { kind: 'pasted', certPem: v.certPem, keyPem: v.keyPem };
    case 'none':
      return { kind: 'none', plainOk: true };
    case 'file':
      return { kind: 'file', certFile: v.certFile.trim(), keyFile: v.keyFile.trim() };
    case 'acme-http':
    case 'acme-dns': {
      const method: AcmeMethod = v.way === 'acme-http' ? 'http-01' : 'dns-01';
      return needsAgreement ? { kind: 'acme', method, agreeTos: true } : { kind: 'acme', method };
    }
  }
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
