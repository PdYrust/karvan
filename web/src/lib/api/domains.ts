import type { AcmeError, AcmeMethod, AcmeOrder } from './acme';
import type { RuleRef } from './certificates';
import { api } from './client';
import { factsOf, list } from './lists';
import type { CertFacts } from './rules';

/** What is wrong with a domain: its own codes, then those of its certificate (D50). */
export type DomainWarning =
  | 'no_certificate'
  | 'cert_conflict'
  | 'certificate_not_adopted'
  | 'renewal_failed'
  | 'renewal_waiting_dns'
  | 'unreadable'
  | 'no_key'
  | 'key_mismatch'
  | 'domain_mismatch'
  | 'expired'
  | 'not_yet_valid'
  | 'expires_soon';

/** A domain rules can answer: a host, one leading `*.` wildcard or an IP (D48). */
export interface Domain {
  id: number;
  name: string;
  /** Its certificate (D61); null is plain HTTP. */
  certificate: { id: number; name: string; origin: '' | 'acme'; facts: CertFacts } | null;
  /** The stored certificates that cover it. */
  certificates: { id: number; name: string }[];
  /** Ports that keep serving another certificate for it since the move (D62). */
  pins: { port: number; certificate: { id: number; name: string } }[];
  /** Its Let's Encrypt state (D69, D70); null unless its certificate or an order is from Let's Encrypt. */
  acme: {
    method: AcmeMethod | '';
    nextRenewal: string | null;
    tries: number;
    lastError: AcmeError | null;
    order: AcmeOrder | null;
  } | null;
  rules: RuleRef[];
  warnings: DomainWarning[];
}

/** How a domain gets its certificate (D61). Keys go in and never come back. */
export type CertChoice =
  | { kind: 'stored'; certificateId: number }
  | { kind: 'file'; certFile: string; keyFile: string; name?: string }
  | { kind: 'pasted'; certPem: string; keyPem: string; name?: string }
  | { kind: 'none'; plainOk: true }
  | { kind: 'acme'; method: AcmeMethod; agreeTos?: boolean };

export const domainKeys = {
  all: ['domains'] as const,
};

/** A domain with its lists, whatever the server wrote for an empty one. */
function domainOf(d: Domain): Domain {
  return {
    ...d,
    certificate: d.certificate && { ...d.certificate, facts: factsOf(d.certificate.facts) },
    certificates: list(d.certificates),
    pins: list(d.pins),
    rules: list(d.rules),
    warnings: list(d.warnings),
  };
}

export async function getDomains(): Promise<{ domains: Domain[] }> {
  const r = await api<{ domains: Domain[] | null }>('GET', 'domains');
  return { domains: list(r.domains).map(domainOf) };
}

/** 400 domain_invalid or certificate_invalid names the field, 409 name_taken for a name already there. */
export async function addDomain(name: string, certificate: CertChoice): Promise<Domain> {
  return domainOf(await api<Domain>('POST', 'domains', { name: name.trim(), certificate }));
}

/**
 * Changes a domain's certificate; 409 domain_in_use for none while an HTTPS rule holds it (D65).
 * The acme way starts an order (202) and keeps the current certificate until it is done.
 */
export async function setDomainCertificate(id: number, certificate: CertChoice): Promise<Domain> {
  return domainOf(await api<Domain>('PUT', `domains/${String(id)}/certificate`, { certificate }));
}

/** 409 domain_in_use while a rule holds it. */
export function deleteDomain(id: number): Promise<void> {
  return api('DELETE', `domains/${String(id)}`);
}
