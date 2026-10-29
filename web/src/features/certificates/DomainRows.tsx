import { Link } from 'react-router';
import type { Column } from '../../components/data/DataTable';
import { CardFacts, RowCard } from '../../components/data/RowCard';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { useFormat } from '../../lib/preferences/region';
import { daysLeft, domainStateOf, listLines, renewalOf } from './cert-view';
import { DomainMenu } from './DomainMenu';
import { certificatesMessages } from './messages';
import { linkClass, Mono, RuleName, useAcmeErrorWords } from './parts';
import { StateMenu } from './StateMenu';

const pageOf = (d: Domain) => `/domains/${String(d.id)}/certificate`;

/** The domain's state, its problems behind it. */
function DomainState({ domain, now, badge }: { domain: Domain; now: Date; badge?: boolean }) {
  const t = useT(certificatesMessages);
  const acmeWords = useAcmeErrorWords();
  const { state, tone } = domainStateOf(domain, now);
  const word = t(`state_${state}`);
  const error = domain.acme?.lastError;
  const problems = listLines(domain, now).map((l) => ({
    key: l.key,
    tone: l.tone,
    text: t(`dwarn_${l.word}`, { ...l.params, reason: error ? acmeWords(error) : '' }),
  }));
  return (
    <StateMenu
      word={word}
      tone={tone}
      problems={problems}
      label={t('showProblems', { name: domain.name, state: word })}
      to={pageOf(domain)}
      badge={badge}
    />
  );
}

/** The certificate and one quiet line: an order's step, the next renewal, or the end date. */
function CertCell({ domain, now }: { domain: Domain; now: Date }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const c = domain.certificate;
  if (!c) return <span className="text-kumo-subtle">{t('plainHttp')}</span>;
  const r = renewalOf(domain);
  const soon = domain.warnings.includes('expires_soon')
    ? daysLeft(c.facts.notAfter, now)
    : undefined;
  let line = '';
  if (r?.kind === 'order') line = t(`order_${r.state}`);
  else if (r && soon === undefined) line = t('renewsOn', { date: fmt.date(r.at) });
  else if (c.facts.notAfter) line = t('endsOn', { date: fmt.date(new Date(c.facts.notAfter)) });
  return (
    <span className="flex min-w-0 flex-col items-start gap-0.5">
      <bdi className="max-w-full truncate">{c.name}</bdi>
      <span className="flex flex-wrap items-center gap-x-3 text-sm">
        {line ? <span className="text-kumo-subtle">{line}</span> : null}
        {soon === undefined ? null : (
          <StatusLabel tone="warning">{t('expiresIn', { count: soon })}</StatusLabel>
        )}
      </span>
    </span>
  );
}

/** Names in a column, or a quiet "None". */
export function Names({ links }: { links: readonly { key: string; node: React.ReactNode }[] }) {
  const t = useT(certificatesMessages);
  if (links.length === 0) return <span className="text-kumo-subtle">{t('none')}</span>;
  return (
    <span className="flex min-w-0 flex-col items-start gap-0.5">
      {links.map((l) => (
        <span key={l.key} className="max-w-full truncate">
          {l.node}
        </span>
      ))}
    </span>
  );
}

const rules = (d: Domain) =>
  d.rules.map((r) => ({ key: `${r.kind}:${String(r.id)}`, node: <RuleName rule={r} /> }));

/** The domain's name as the row's link, to its certificate page. */
function DomainLink({ domain }: { domain: Domain }) {
  return (
    <Link to={pageOf(domain)} className={linkClass}>
      <Mono text={domain.name} />
    </Link>
  );
}

export function useDomainColumns(now: Date, onDelete: (d: Domain) => void): Column<Domain>[] {
  const t = useT(certificatesMessages);
  return [
    {
      key: 'domain',
      header: t('colDomain'),
      cell: (d) => <DomainLink domain={d} />,
      className: 'max-w-64',
    },
    { key: 'state', header: t('colState'), cell: (d) => <DomainState domain={d} now={now} /> },
    {
      key: 'cert',
      header: t('colCert'),
      cell: (d) => <CertCell domain={d} now={now} />,
      className: 'max-w-64',
    },
    {
      key: 'rules',
      header: t('colRules'),
      cell: (d) => <Names links={rules(d)} />,
      className: 'max-w-56',
    },
    {
      key: 'menu',
      header: t('colMenu'),
      hideHeader: true,
      cell: (d) => <DomainMenu domain={d} onDelete={() => onDelete(d)} />,
      className: 'w-12 text-end',
    },
  ];
}

/** On a phone a domain is a card: the name, its state and menu, then the certificate and rules. */
export function DomainRowCard({
  domain,
  now,
  onDelete,
}: {
  domain: Domain;
  now: Date;
  onDelete: () => void;
}) {
  const t = useT(certificatesMessages);
  return (
    <RowCard
      lead={<DomainLink domain={domain} />}
      badge={<DomainState domain={domain} now={now} badge />}
      menu={<DomainMenu domain={domain} onDelete={onDelete} />}
    >
      <CardFacts
        className="grid-cols-2"
        facts={[
          { label: t('colCert'), value: <CertCell domain={domain} now={now} /> },
          { label: t('colRules'), value: <Names links={rules(domain)} /> },
        ]}
      />
    </RowCard>
  );
}
