import { Link } from 'react-router';
import type { Column } from '../../components/data/DataTable';
import { RowCard } from '../../components/data/RowCard';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import { useFormat } from '../../lib/preferences/region';
import { certProblems, certStateOf, daysLeft, domainWarningTone } from './cert-view';
import { CertMenu } from './CertMenu';
import { copyMessages } from './copy-messages';
import { CopyTargets } from './CopyTargets';
import { domainPage, Names } from './DomainRows';
import { certFactMessages } from './fact-messages';
import { certificatesMessages } from './messages';
import { linkClass, Mono, RuleName } from './parts';
import { StateMenu } from './StateMenu';

/** How it is kept: files, pasted, or from Let's Encrypt, which Karvan renews itself. */
export const kindOf = (c: Certificate) =>
  c.origin === 'acme' ? 'kind_acme' : (`kind_${c.kind}` as const);
/** Where a certificate's problems are fixed: its first domain's page, where Change certificate is. */
const pageOf = (c: Certificate) => (c.domains[0] ? domainPage(c.domains[0]) : undefined);

function CertState({ cert, badge }: { cert: Certificate; badge?: boolean }) {
  const t = useT(certificatesMessages);
  const tf = useT(certFactMessages);
  const { state, tone } = certStateOf(cert.facts.warnings);
  const word = t(`state_${state}`);
  return (
    <StateMenu
      word={word}
      tone={tone}
      problems={certProblems(cert.facts.warnings).map((w) => ({
        key: w,
        tone: domainWarningTone(w),
        text: tf(`warn_${w}`),
      }))}
      label={t('showProblems', { name: cert.name, state: word })}
      to={pageOf(cert)}
      badge={badge}
    />
  );
}

/** The end date, and "Expires in N days" beside it while it ends soon. */
function Ends({ cert, now }: { cert: Certificate; now: Date }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const end = cert.facts.notAfter;
  if (!end) return <span className="text-kumo-subtle">{t('none')}</span>;
  const soon = cert.facts.warnings.includes('expires_soon') ? daysLeft(end, now) : undefined;
  return (
    <span className="flex flex-col items-start gap-0.5">
      <span>{fmt.date(new Date(end))}</span>
      {soon === undefined ? null : (
        <span className="text-sm">
          <StatusLabel tone="warning">{t('expiresIn', { count: soon })}</StatusLabel>
        </span>
      )}
    </span>
  );
}

/** The domains (each a link to its page) and the rules that use it. */
const users = (c: Certificate) => [
  ...c.domains.map((d) => ({
    key: `d:${String(d.id)}`,
    node: (
      <Link to={domainPage(d)} className={linkClass}>
        <Mono text={d.name} />
      </Link>
    ),
  })),
  ...c.rules.map((r) => ({ key: `${r.kind}:${String(r.id)}`, node: <RuleName rule={r} /> })),
];

const covers = (c: Certificate) => <Mono text={c.facts.domains.join(', ')} className="text-sm" />;

export function useCertColumns(
  now: Date,
  onDelete: (c: Certificate) => void,
): Column<Certificate>[] {
  const t = useT(certificatesMessages);
  const tc = useT(copyMessages);
  return [
    {
      key: 'name',
      header: t('colName'),
      cell: (c) => (
        <span className="flex min-w-0 flex-col gap-0.5">
          <bdi className="max-w-full truncate text-kumo-strong">{c.name}</bdi>
          <span className="text-sm text-kumo-subtle">{t(kindOf(c))}</span>
        </span>
      ),
      className: 'max-w-56',
    },
    { key: 'state', header: t('colState'), cell: (c) => <CertState cert={c} /> },
    { key: 'covers', header: t('colCovers'), cell: covers, className: 'max-w-56' },
    { key: 'ends', header: t('colEnds'), cell: (c) => <Ends cert={c} now={now} /> },
    {
      key: 'used',
      header: t('usedBy'),
      cell: (c) => <Names links={users(c)} />,
      className: 'max-w-56',
    },
    {
      key: 'copies',
      header: tc('colCopies'),
      cell: (c) => <CopyTargets cert={c} />,
      className: 'max-w-64',
    },
    {
      key: 'menu',
      header: t('colMenu'),
      hideHeader: true,
      cell: (c) => <CertMenu cert={c} onDelete={() => onDelete(c)} />,
      className: 'w-12 text-end',
    },
  ];
}

/** On a phone a certificate is a row like a Migrations card: the name, its state and menu, then one line. */
export function CertRowCard({
  cert,
  now,
  onDelete,
}: {
  cert: Certificate;
  now: Date;
  onDelete: () => void;
}) {
  const t = useT(certificatesMessages);
  const tc = useT(copyMessages);
  const fmt = useFormat();
  const end = cert.facts.notAfter;
  const soon = end && cert.facts.warnings.includes('expires_soon') ? daysLeft(end, now) : undefined;
  return (
    <RowCard
      lead={<bdi className="truncate font-medium text-kumo-strong">{cert.name}</bdi>}
      badge={<CertState cert={cert} badge />}
      menu={<CertMenu cert={cert} onDelete={onDelete} />}
    >
      <p className="flex flex-wrap items-center gap-x-3 text-sm text-kumo-subtle">
        {cert.facts.issuer ? <bdi className="max-w-full truncate">{cert.facts.issuer}</bdi> : null}
        {end ? <span>{t('endsOn', { date: fmt.date(new Date(end)) })}</span> : null}
        {soon === undefined ? null : (
          <StatusLabel tone="warning">{t('expiresIn', { count: soon })}</StatusLabel>
        )}
      </p>
      {cert.copies.length > 0 ? (
        <div className="flex flex-col gap-1 text-sm">
          <span className="text-kumo-subtle">{tc('colCopies')}</span>
          <CopyTargets cert={cert} />
        </div>
      ) : null}
    </RowCard>
  );
}
