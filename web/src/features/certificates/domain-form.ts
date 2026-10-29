import type { AcmeMethod } from '../../lib/api/acme';
import { ApiError } from '../../lib/api/client';
import type { Certificate, ScanResult } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import type { CertFacts, CertWarning } from '../../lib/api/rules';

/**
 * The ways a domain gets its certificate (D61). Let's Encrypt is two ways, one
 * per check, since the Other ways menu offers them apart.
 */
export type Way = 'found' | 'acme-http' | 'acme-dns' | 'upload' | 'paste' | 'file' | 'none';

/** Add domain asks for the name first, then for its certificate. */
export type Step = 'domain' | 'certificate';

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
  agreeTos: boolean;
}

export function newDomainForm(name = ''): DomainFormValues {
  return {
    name,
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

export interface FoundChoice {
  key: string;
  pick: FoundPick;
  /** The stored certificate's name, or the found file. */
  title: string;
  facts: CertFacts;
}

/** 0 when a name the certificate carries is the domain itself, 1 when only a wildcard covers it. */
function matchRank(facts: CertFacts, name: string): number | undefined {
  const d = domainName(name);
  if (facts.domains.some((n) => domainName(n) === d)) return 0;
  return facts.domains.some((n) => covers(n, d)) ? 1 : undefined;
}

/**
 * The certificates that cover the name: exact names before wildcards, and in each
 * stored ones before found files Karvan does not keep yet. `except` leaves out
 * the domain's own certificate on the change page.
 */
export function foundChoices(
  rawName: string,
  stored: readonly Certificate[],
  scan: ScanResult | undefined,
  except?: number,
): FoundChoice[] {
  const name = domainName(rawName);
  if (!name) return [];
  const kept = stored
    .filter((c) => c.id !== except)
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
  const all: FoundChoice[] = [...kept, ...found];
  const ranked: { c: FoundChoice; rank: number }[] = [];
  for (const c of all) {
    const rank = matchRank(c.facts, name);
    if (rank !== undefined) ranked.push({ c, rank });
  }
  return ranked.sort((a, b) => a.rank - b.rank).map((x) => x.c);
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

/** The one to choose: it can serve and ends last; on a tie the earlier in the list. */
export function bestChoice(choices: readonly FoundChoice[], now: Date): FoundChoice | undefined {
  let best: FoundChoice | undefined;
  for (const c of choices) {
    if (canServe(c.facts, now) && (!best || endOf(c.facts) > endOf(best.facts))) best = c;
  }
  return best;
}

/** Let's Encrypt checks a wildcard only by a TXT record (D66). */
export function acmeMethodsFor(name: string): AcmeMethod[] {
  return domainName(name).startsWith('*.') ? ['dns-01'] : ['http-01', 'dns-01'];
}

const isWildcard = (name: string) => !acmeMethodsFor(name).includes('http-01');

/** The answer chosen when the certificate step opens: the best found one, else Let's Encrypt. */
export function recommended(
  name: string,
  choices: readonly FoundChoice[],
  now: Date,
): Pick<DomainFormValues, 'way' | 'pick'> {
  const best = bestChoice(choices, now);
  if (best) return { way: 'found', pick: best.pick };
  return { way: isWildcard(name) ? 'acme-dns' : 'acme-http', pick: null };
}

/** The Other ways menu: every way but the chosen one, No certificate last. */
export function otherWays(current: Way, name: string, found: boolean): Way[] {
  const all: Way[] = ['found', 'acme-http', 'acme-dns', 'upload', 'paste', 'file', 'none'];
  return all.filter(
    (w) => w !== current && (w !== 'found' || found) && (w !== 'acme-http' || !isWildcard(name)),
  );
}

/** What the chosen way still needs before Add domain, if anything. */
export type Missing = 'pick' | 'pem' | 'files' | 'agree' | 'wildcard';

const filled = (s: string) => s.trim() !== '';

export function missingOf(v: DomainFormValues, needsAgreement: boolean): Missing | undefined {
  switch (v.way) {
    case 'found':
      return v.pick ? undefined : 'pick';
    case 'upload':
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
    case 'upload':
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

/** Continue opens the certificate step only once a name is typed. */
export function stepAfter(name: string): Step {
  return domainName(name) ? 'certificate' : 'domain';
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
