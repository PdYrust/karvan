import type { RuleRef } from './certificates';
import { api } from './client';
import type { CertFacts } from './rules';

/** What is wrong with a domain: its own codes, then those of its certificate (D50). */
export type DomainWarning =
  | 'no_certificate'
  | 'cert_conflict'
  | 'certificate_not_adopted'
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
  rules: RuleRef[];
  warnings: DomainWarning[];
}

/** How a domain gets its certificate (D61). Keys go in and never come back. */
export type CertChoice =
  | { kind: 'stored'; certificateId: number }
  | { kind: 'file'; certFile: string; keyFile: string; name?: string }
  | { kind: 'pasted'; certPem: string; keyPem: string; name?: string }
  | { kind: 'none'; plainOk: true };

export const domainKeys = {
  all: ['domains'] as const,
};

export function getDomains(): Promise<{ domains: Domain[] }> {
  return api('GET', 'domains');
}

/** 400 domain_invalid or certificate_invalid names the field, 409 name_taken for a name already there. */
export function addDomain(name: string, certificate: CertChoice): Promise<Domain> {
  return api('POST', 'domains', { name: name.trim(), certificate });
}

/** Changes a domain's certificate; 409 domain_in_use for none while an HTTPS rule holds it (D65). */
export function setDomainCertificate(id: number, certificate: CertChoice): Promise<Domain> {
  return api('PUT', `domains/${String(id)}/certificate`, { certificate });
}

/** 409 domain_in_use while a rule holds it. */
export function deleteDomain(id: number): Promise<void> {
  return api('DELETE', `domains/${String(id)}`);
}
