import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { AcmeError } from '../../lib/api/acme';
import type { RuleRef } from '../../lib/api/certificates';
import type { Domain } from '../../lib/api/domains';
import { acmeErrorKey, acmeErrorMessages } from '../../lib/errors/acme';
import { useFormat } from '../../lib/preferences/region';
import { domainLines } from './cert-view';
import { certFactMessages } from './fact-messages';
import { certificatesMessages } from './messages';

export const linkClass =
  'text-kumo-default underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** A rule by name: a panel rule links to its form; a redirect rule has no page of its own yet. */
export function RuleName({ rule }: { rule: RuleRef }) {
  if (rule.kind !== 'panel') return <bdi>{rule.name}</bdi>;
  return (
    <Link to={`/rules/${String(rule.id)}`} className={linkClass}>
      <bdi>{rule.name}</bdi>
    </Link>
  );
}

/** Names in a column, or a quiet "None". */
export function NameList({ names }: { names: readonly { key: string; node: ReactNode }[] }) {
  const t = useT(certificatesMessages);
  if (names.length === 0) return <span className="text-kumo-subtle">{t('none')}</span>;
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      {names.map((n) => (
        <span key={n.key} className="min-w-0 truncate">
          {n.node}
        </span>
      ))}
    </span>
  );
}

/** A Let's Encrypt error in words, a rate limit with the time to try again (D71). */
export function useAcmeErrorWords(): (e: AcmeError) => string {
  const t = useT(acmeErrorMessages);
  const fmt = useFormat();
  return (e) => {
    const time = e.retryAt ? fmt.dateTime(new Date(e.retryAt)) : '';
    return t(acmeErrorKey(e.code) ?? 'acme_failed', { time });
  };
}

/** A domain's warnings as status lines, worst first, or that none was found. */
export function DomainLinesView({ domain, now }: { domain: Domain; now: Date }) {
  const t = useT(certificatesMessages);
  const tf = useT(certFactMessages);
  const acmeWords = useAcmeErrorWords();
  const lines = domainLines(domain, now);
  if (lines.length === 0) {
    return (
      <span className="text-sm">
        <StatusLabel tone="success">{tf('certFine')}</StatusLabel>
      </span>
    );
  }
  const error = domain.acme?.lastError;
  return (
    <ul className="flex flex-col gap-1.5 text-sm">
      {lines.map((l) => (
        <li key={l.key} className="flex flex-col items-start gap-0.5">
          <StatusLabel tone={l.tone}>
            {t(`dwarn_${l.word}`, {
              ...l.params,
              reason: error ? acmeWords(error) : '',
            })}
          </StatusLabel>
          {l.word === 'renewal_waiting_dns' ? (
            <Link
              to={`/domains/${String(domain.id)}/certificate`}
              className={cn('ms-6', linkClass)}
            >
              {t('showDns')}
            </Link>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

/** Machine text such as a domain or a path, left to right in either language. */
export function Mono({ text, className }: { text: string; className?: string }) {
  return (
    <bdi dir="ltr" className={cn('font-mono text-[0.9em] break-all', className)}>
      {text}
    </bdi>
  );
}
