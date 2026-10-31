import { cn } from '@cloudflare/kumo/utils';
import { Link } from 'react-router';
import { useT } from '../../i18n/locale';
import type { AcmeError } from '../../lib/api/acme';
import type { RuleRef } from '../../lib/api/certificates';
import { rulePath } from '../../lib/api/rules';
import { acmeErrorKey, acmeErrorMessages } from '../../lib/errors/acme';
import { useFormat } from '../../lib/preferences/region';

export const linkClass =
  'text-kumo-default underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** A rule by name, as a link to its form. */
export function RuleName({ rule }: { rule: RuleRef }) {
  return (
    <Link to={rulePath(rule)} className={linkClass}>
      <bdi>{rule.name}</bdi>
    </Link>
  );
}

/** A Let's Encrypt error in words, a rate limit with the time to try again. */
export function useAcmeErrorWords(): (e: AcmeError) => string {
  const t = useT(acmeErrorMessages);
  const fmt = useFormat();
  return (e) => {
    const time = e.retryAt ? fmt.dateTime(new Date(e.retryAt)) : '';
    return t(acmeErrorKey(e.code) ?? 'acme_failed', { time });
  };
}

/** Machine text such as a domain or a path, left to right in either language. */
export function Mono({ text, className }: { text: string; className?: string }) {
  return (
    <bdi dir="ltr" className={cn('font-mono text-[0.9em] break-all', className)}>
      {text}
    </bdi>
  );
}
