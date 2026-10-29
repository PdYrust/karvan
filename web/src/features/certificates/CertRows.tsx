import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { Link, useNavigate } from 'react-router';
import type { Column } from '../../components/data/DataTable';
import { CardFacts, RowCard } from '../../components/data/RowCard';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { DotsThreeIcon, PencilSimpleIcon, TrashIcon } from '../../components/icons/regular';
import { useT } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import { useFormat } from '../../lib/preferences/region';
import { certProblems, certStateOf, daysLeft, domainWarningTone } from './cert-view';
import { Names } from './DomainRows';
import { certFactMessages } from './fact-messages';
import { certificatesMessages } from './messages';
import { linkClass, Mono, RuleName } from './parts';
import { StateMenu } from './StateMenu';

/** Nothing uses it, so it can go; Karvan refuses the delete otherwise. */
const isUnused = (c: Certificate) => c.domains.length === 0 && c.rules.length === 0;
/** How it is kept: files, pasted, or from Let's Encrypt, which Karvan renews itself. */
export const kindOf = (c: Certificate) =>
  c.origin === 'acme' ? 'kind_acme' : (`kind_${c.kind}` as const);
/** Where a certificate is fixed: its edit page, or its first domain's for a Let's Encrypt one. */
const pageOf = (c: Certificate) =>
  c.origin !== 'acme'
    ? `/domains/certificates/${String(c.id)}`
    : c.domains[0]
      ? `/domains/${String(c.domains[0].id)}/certificate`
      : undefined;

function CertMenu({ cert, onDelete }: { cert: Certificate; onDelete: () => void }) {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  const label = t('moreFor', { name: cert.name });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} />}
            className="shrink-0 pointer-coarse:size-9"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-44">
        {cert.origin === 'acme' ? null : (
          <DropdownMenu.Item
            className="gap-2"
            onClick={() => void navigate(`/domains/certificates/${String(cert.id)}`)}
          >
            <PencilSimpleIcon size={16} />
            {t('edit')}
          </DropdownMenu.Item>
        )}
        <DropdownMenu.Item
          className="gap-2"
          variant="danger"
          disabled={!isUnused(cert)}
          onClick={onDelete}
        >
          <TrashIcon size={16} />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

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

/** The domains (each a link to its certificate page) and the rules that use it. */
const users = (c: Certificate) => [
  ...c.domains.map((d) => ({
    key: `d:${String(d.id)}`,
    node: (
      <Link to={`/domains/${String(d.id)}/certificate`} className={linkClass}>
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
      key: 'menu',
      header: t('colMenu'),
      hideHeader: true,
      cell: (c) => <CertMenu cert={c} onDelete={() => onDelete(c)} />,
      className: 'w-12 text-end',
    },
  ];
}

/** On a phone a certificate is a card: the name, its state and menu, then its facts. */
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
  return (
    <RowCard
      lead={<bdi className="truncate font-medium text-kumo-strong">{cert.name}</bdi>}
      badge={<CertState cert={cert} badge />}
      menu={<CertMenu cert={cert} onDelete={onDelete} />}
    >
      <CardFacts
        className="grid-cols-2"
        facts={[
          { label: t('kindLabel'), value: t(kindOf(cert)) },
          { label: t('colEnds'), value: <Ends cert={cert} now={now} /> },
          { label: t('colCovers'), value: covers(cert), wide: true },
          { label: t('usedBy'), value: <Names links={users(cert)} />, wide: true },
        ]}
      />
    </RowCard>
  );
}
