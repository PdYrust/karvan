import type { StatusTone } from '../../components/feedback/StatusBadge';
import type { AcmeMethod, AcmeState } from '../../lib/api/acme';
import type { RuleRef, ScanResult } from '../../lib/api/certificates';
import type { Domain, DomainWarning } from '../../lib/api/domains';

/** What stops a handshake first, then what needs a look (D50). */
const order: readonly DomainWarning[] = [
  'unreadable',
  'no_key',
  'key_mismatch',
  'expired',
  'not_yet_valid',
  'no_certificate',
  'certificate_not_adopted',
  'renewal_failed',
  'cert_conflict',
  'domain_mismatch',
  'expires_soon',
  'renewal_waiting_dns',
];

const danger = new Set<DomainWarning>([
  'unreadable',
  'no_key',
  'key_mismatch',
  'expired',
  'not_yet_valid',
]);

export function sortedWarnings<W extends DomainWarning>(list: readonly W[]): W[] {
  return [...list].sort((a, b) => order.indexOf(a) - order.indexOf(b));
}

/** A certificate that cannot serve is a danger; a missing or partial one, or one ending soon, a warning. */
export function domainWarningTone(w: DomainWarning): StatusTone {
  return danger.has(w) ? 'danger' : 'warning';
}

const day = 24 * 60 * 60 * 1000;

/** Whole days until a certificate ends, a started day counted; 0 once ended. */
export function daysLeft(notAfter: string | undefined, now: Date): number | undefined {
  if (!notAfter) return undefined;
  return Math.max(0, Math.ceil((new Date(notAfter).getTime() - now.getTime()) / day));
}

/** The domains of one rule that the certificate it uses does not cover. */
export function uncoveredDomains(
  certificateId: number,
  rule: Pick<RuleRef, 'kind' | 'id'>,
  domains: readonly Domain[],
): string[] {
  return domains
    .filter((d) => d.rules.some((r) => r.kind === rule.kind && r.id === rule.id))
    .filter((d) => !d.certificates.some((c) => c.id === certificateId))
    .map((d) => d.name);
}

/** Every place the scan did not read, once, whatever the reason (D51, D59). */
export function notScanned(r: ScanResult): string[] {
  return [...new Set(r.skipped.map((s) => s.place))];
}

/** One status line of a domain: a warning's words, or a pinned port's. */
export interface DomainLine {
  key: string;
  word: DomainWarning | 'pin';
  tone: StatusTone;
  params: Record<string, string | number>;
}

/** A domain's lines, worst first: each pin by port, the days left, a failed renewal's code. */
export function domainLines(d: Domain, now: Date): DomainLine[] {
  const lines: DomainLine[] = [];
  for (const w of sortedWarnings(d.warnings)) {
    if (w === 'cert_conflict' && d.pins.length > 0) {
      for (const p of d.pins) {
        lines.push({
          key: `pin:${String(p.port)}`,
          word: 'pin',
          tone: 'warning',
          params: { port: p.port, cert: p.certificate.name },
        });
      }
      continue;
    }
    const params: DomainLine['params'] = {};
    const left = daysLeft(d.certificate?.facts.notAfter, now);
    if (w === 'expires_soon' && left !== undefined) params.count = left;
    if (w === 'renewal_failed' && d.acme?.lastError) params.code = d.acme.lastError.code;
    lines.push({ key: w, word: w, tone: domainWarningTone(w), params });
  }
  return lines;
}

export type RunningState = Exclude<AcmeState, 'done' | 'failed' | 'cancelled'>;
const runningStates: readonly AcmeState[] = ['starting', 'waiting_dns', 'validating', 'issuing'];
const isRunningState = (s: AcmeState | undefined): s is RunningState =>
  s !== undefined && runningStates.includes(s);

/** A Let's Encrypt domain's running order, else its next renewal. */
export function renewalOf(
  d: Domain,
): { kind: 'order'; state: RunningState } | { kind: 'renews'; at: Date } | undefined {
  const a = d.acme;
  if (!a) return undefined;
  const state = a.order?.state;
  if (isRunningState(state)) return { kind: 'order', state };
  return a.nextRenewal ? { kind: 'renews', at: new Date(a.nextRenewal) } : undefined;
}

/** The way Renew now orders again: a Let's Encrypt certificate's own, while no order runs. */
export function renewMethod(d: Domain): AcmeMethod | undefined {
  const a = d.acme;
  if (d.certificate?.origin !== 'acme' || !a?.method || isRunningState(a.order?.state))
    return undefined;
  return a.method;
}

export type DomainState = 'active' | 'plain' | 'attention' | 'broken';

/** A domain's lines without the end date's warning, which sits next to the date itself. */
export function listLines(d: Domain, now: Date): DomainLine[] {
  return domainLines(d, now).filter((l) => l.word !== 'expires_soon');
}

/**
 * One word for the head of a domain's row: its worst line decides, none is Active.
 * A certificate that ends soon still serves, so it stays Active.
 */
export function domainStateOf(d: Domain, now: Date): { state: DomainState; tone: StatusTone } {
  const lines = listLines(d, now);
  if (lines.some((l) => l.tone === 'danger')) return { state: 'broken', tone: 'danger' };
  if (lines.length === 0) return { state: 'active', tone: 'success' };
  if (lines.every((l) => l.word === 'no_certificate')) return { state: 'plain', tone: 'neutral' };
  return { state: 'attention', tone: 'warning' };
}
