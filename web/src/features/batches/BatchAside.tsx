import { Link } from 'react-router';
import { AsideBlock } from '../../components/layout/AsideBlock';
import { FactList, type Fact } from '../../components/layout/FactList';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import type { Batch } from '../../lib/api/batches';
import { useFormat } from '../../lib/preferences/region';
import { batchesMessages } from './messages';

const link =
  'font-medium text-kumo-strong underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** Where the old links pointed, and how many users proved one or are closed for good. */
function OldLinksBlock({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  const fmt = useFormat();
  const { path, hosts, port } = batch.oldLinks;
  const unset = <span className="text-kumo-subtle">{t('notSet')}</span>;
  const facts: Fact[] = [
    { label: t('oldPath'), value: path ? <Machine>{`/${path}`}</Machine> : unset },
    {
      label: t('domains'),
      value: hosts.length ? (
        <span className="flex flex-col items-end gap-0.5">
          {hosts.map((h) => (
            <Machine key={h}>{h}</Machine>
          ))}
        </span>
      ) : (
        unset
      ),
    },
    { label: t('port'), value: port ? <Machine>{String(port)}</Machine> : unset },
    { label: t('verifiedCount'), value: fmt.number(batch.counts.verified) },
    { label: t('closedCount'), value: fmt.number(batch.counts.closed) },
  ];
  return (
    <AsideBlock title={t('oldLinks')}>
      <FactList facts={facts} />
    </AsideBlock>
  );
}

function RulesBlock({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  return (
    <AsideBlock title={t('rules')}>
      {batch.rules.length === 0 ? (
        <p className="text-sm text-kumo-subtle">{t('noRuleUses')}</p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {batch.rules.map((r) => (
            <li key={r.id}>
              <Link to={`/links/${String(r.id)}`} className={link}>
                <bdi>{r.name}</bdi>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AsideBlock>
  );
}

function DetailsBlock({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  const fmt = useFormat();
  const facts: Fact[] = [
    {
      label: t('migration'),
      value: (
        <Link to={`/migrations/${encodeURIComponent(batch.jobId)}`} className={link}>
          {t('openReport')}
        </Link>
      ),
    },
    { label: t('oldPanel'), value: <Machine>{batch.source.origin}</Machine> },
    {
      label: t('newPanel'),
      value: <PanelLabel kind={batch.panel.kind || undefined} name={batch.panel.name} />,
    },
    { label: t('made'), value: fmt.dateTime(new Date(batch.createdAt)) },
  ];
  if (batch.inactiveAt) {
    facts.push({ label: t('inactiveSince'), value: fmt.dateTime(new Date(batch.inactiveAt)) });
  }
  return (
    <AsideBlock title={t('facts')}>
      <FactList facts={facts} />
    </AsideBlock>
  );
}

/** The batch page's aside: its old links, the rules that use it, then the details. */
export function BatchAside({ batch }: { batch: Batch }) {
  return (
    <div className="flex flex-col">
      <OldLinksBlock batch={batch} />
      <RulesBlock batch={batch} />
      <DetailsBlock batch={batch} />
    </div>
  );
}
