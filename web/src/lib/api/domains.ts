import type { RuleRef } from './certificates';
import { api } from './client';

/** What is wrong with a domain: its own codes, then those of its rules' certificates (D50). */
export type DomainWarning =
  | 'no_certificate'
  | 'cert_conflict'
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
  /** The stored certificates that cover it. */
  certificates: { id: number; name: string }[];
  rules: RuleRef[];
  warnings: DomainWarning[];
}

export const domainKeys = {
  all: ['domains'] as const,
};

export function getDomains(): Promise<{ domains: Domain[] }> {
  return api('GET', 'domains');
}

/** 400 domain_invalid for a name that is not a domain, 409 name_taken for one already there. */
export function addDomain(name: string): Promise<Domain> {
  return api('POST', 'domains', { name: name.trim() });
}

/** 409 domain_in_use while a rule holds it. */
export function deleteDomain(id: number): Promise<void> {
  return api('DELETE', `domains/${String(id)}`);
}
