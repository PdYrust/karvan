import type { StatusTone } from '../../components/feedback/StatusBadge';
import type { Domain, DomainWarning } from '../../lib/api/domains';

/** A warning a rule card shows: its domain's (with the domain), or the rule's own. */
export interface WarningLine {
  domain?: Domain;
  code: DomainWarning;
}

/** Plain HTTP is in the card's summary already; a domain not filled yet is the rule's line. */
const hidden: ReadonlySet<DomainWarning> = new Set(['no_certificate', 'certificate_not_adopted']);

/** The rule's own line first, then each of its domains' warnings (D50). */
export function warningLines(
  names: readonly string[] | null,
  warnings: readonly string[],
  domains: readonly Domain[],
): WarningLine[] {
  const lines: WarningLine[] = warnings.includes('certificate_not_adopted')
    ? [{ code: 'certificate_not_adopted' }]
    : [];
  for (const name of names ?? []) {
    const d = domains.find((x) => x.name === name);
    for (const code of d?.warnings ?? []) {
      if (!hidden.has(code)) lines.push({ domain: d, code });
    }
  }
  return lines;
}

const danger: ReadonlySet<DomainWarning> = new Set([
  'unreadable',
  'no_key',
  'key_mismatch',
  'expired',
  'not_yet_valid',
  'renewal_failed',
]);

/** What stops HTTPS now is a danger; the rest is a warning. */
export function warningTone(code: DomainWarning): StatusTone {
  return danger.has(code) ? 'danger' : 'warning';
}
