import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon } from '@phosphor-icons/react';
import { Fragment, useId } from 'react';
import { useNavigate } from 'react-router';
import { StatusBadge, StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { useFormat } from '../../lib/preferences/region';
import { daysLeft, domainStateOf, listLines, renewalOf } from './cert-view';
import { DomainMenu } from './DomainMenu';
import { certificatesMessages } from './messages';
import { Mono, RuleName, useAcmeErrorWords } from './parts';

/**
 * The domain's state as a badge. With problems it opens a small menu like the
 * account menu: each problem worst first; a press on one opens
 * the domain's certificate page, where it is fixed. Without, it is only the word.
 */
function StateBadge({ domain, now }: { domain: Domain; now: Date }) {
  const t = useT(certificatesMessages);
  const acmeWords = useAcmeErrorWords();
  const navigate = useNavigate();
  const { state, tone } = domainStateOf(domain, now);
  const word = t(`state_${state}`);
  const badge = <StatusBadge tone={tone}>{word}</StatusBadge>;
  const lines = listLines(domain, now);
  if (lines.length === 0) return badge;
  const error = domain.acme?.lastError;
  const page = `/domains/${String(domain.id)}/certificate`;
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={t('showProblems', { name: domain.name, state: word })}
            className="relative inline-flex cursor-pointer items-center gap-1 rounded-md after:absolute after:-inset-x-1 after:-inset-y-3 after:content-[''] hover:opacity-80 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-kumo-brand"
          >
            {badge}
            <CaretDownIcon size={12} aria-hidden="true" className="text-kumo-subtle" />
          </button>
        }
      />
      <DropdownMenu.Content align="start" className="w-72 max-w-[calc(100vw-2rem)]">
        <DropdownMenu.Group>
          {lines.map((l) => (
            <DropdownMenu.Item key={l.key} onClick={() => void navigate(page)}>
              <span className="text-sm whitespace-normal">
                <StatusLabel tone={l.tone}>
                  {t(`dwarn_${l.word}`, { ...l.params, reason: error ? acmeWords(error) : '' })}
                </StatusLabel>
              </span>
            </DropdownMenu.Item>
          ))}
        </DropdownMenu.Group>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

/** The quiet line: the certificate, its renewal or end date, and the rules that hold it. */
function Summary({ domain, now }: { domain: Domain; now: Date }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const c = domain.certificate;
  const r = renewalOf(domain);
  const ends = c?.facts.notAfter ? new Date(c.facts.notAfter) : undefined;
  const soon = domain.warnings.includes('expires_soon')
    ? daysLeft(c?.facts.notAfter, now)
    : undefined;
  let when = '';
  if (r?.kind === 'order') when = t(`order_${r.state}`);
  else if (ends && soon !== undefined) when = t('endsSoon', { date: fmt.date(ends), count: soon });
  else if (r) when = t('renewsOn', { date: fmt.date(r.at) });
  else if (ends) when = t('endsOn', { date: fmt.date(ends) });
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
      {c ? <bdi className="text-kumo-default">{c.name}</bdi> : <span>{t('noCert')}</span>}
      {when ? <span>{when}</span> : null}
      {domain.rules.length > 0 ? (
        <span className="min-w-0">
          {t('usedBy')}{' '}
          {domain.rules.map((rule, i) => (
            <Fragment key={`${rule.kind}:${String(rule.id)}`}>
              {i > 0 ? t('listJoin') : null}
              <RuleName rule={rule} />
            </Fragment>
          ))}
        </span>
      ) : (
        <span>{t('noRules')}</span>
      )}
    </div>
  );
}

/** One domain, as short as a closed certificate card: its name, state and menu, then one line. */
export function DomainCard({
  domain,
  now,
  onDelete,
}: {
  domain: Domain;
  now: Date;
  onDelete: () => void;
}) {
  const id = useId();
  return (
    <article
      aria-labelledby={id}
      className="k-fade rounded-lg bg-kumo-base ring ring-kumo-hairline"
    >
      <div className="flex items-start gap-3 py-2 ps-4 pe-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1 py-1.5">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <h3 id={id} className="min-w-0 font-semibold text-kumo-strong">
              <Mono text={domain.name} />
            </h3>
            <StateBadge domain={domain} now={now} />
          </div>
          <Summary domain={domain} now={now} />
        </div>
        <DomainMenu domain={domain} onDelete={onDelete} />
      </div>
    </article>
  );
}
