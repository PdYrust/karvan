import { CardFacts, RowCard } from '../../components/data/RowCard';
import { DataTable, type Column } from '../../components/data/DataTable';
import { useT } from '../../i18n/locale';
import type { Batch } from '../../lib/api/batches';
import { listMessages } from '../migrations/list-messages';
import {
  BatchAdmins,
  BatchCardName,
  BatchDestination,
  BatchMade,
  BatchMove,
  BatchName,
  BatchRules,
  BatchState,
  BatchStateBadge,
  BatchUserCount,
} from './BatchCells';
import { batchLook } from './batch-view';
import { batchesMessages } from './messages';

/** On a phone a batch is a card: the move and its state, the name, the admins and rules, then the users and when. */
function BatchCard({ batch, now }: { batch: Batch; now: Date }) {
  const t = useT(batchesMessages);
  const reason = batchLook(batch).reason;
  return (
    <RowCard
      lead={<BatchMove batch={batch} size={20} />}
      badge={<BatchStateBadge batch={batch} />}
      footer={
        <>
          <span>
            {t('users', { count: batch.counts.users })}
            <span aria-hidden="true" className="px-1.5">
              ·
            </span>
            {t('verified', { count: batch.counts.verified })}
          </span>
          <BatchMade batch={batch} now={now} />
        </>
      }
    >
      <BatchCardName batch={batch} />
      {reason ? <p className="text-sm text-kumo-subtle">{t(reason)}</p> : null}
      <CardFacts
        className="grid-cols-2"
        facts={[
          { label: t('colAdmins'), value: <BatchAdmins batch={batch} /> },
          { label: t('colRules'), value: <BatchRules batch={batch} /> },
        ]}
      />
    </RowCard>
  );
}

/** Batches newest first: the move and name, the new panel, admins, users, state, rules and when. */
export function BatchesTable({ batches, now }: { batches: readonly Batch[]; now: Date }) {
  const t = useT(batchesMessages);
  const tl = useT(listMessages);
  const columns: Column<Batch>[] = [
    {
      key: 'batch',
      header: t('colBatch'),
      cell: (b) => <BatchName batch={b} />,
      className: 'max-w-72',
    },
    {
      key: 'panel',
      header: tl('colDestination'),
      cell: (b) => <BatchDestination batch={b} />,
      className: 'max-w-56',
    },
    {
      key: 'admins',
      header: t('colAdmins'),
      cell: (b) => <BatchAdmins batch={b} />,
      className: 'max-w-48',
    },
    { key: 'users', header: t('colUsers'), cell: (b) => <BatchUserCount batch={b} /> },
    { key: 'state', header: t('colState'), cell: (b) => <BatchState batch={b} /> },
    {
      key: 'rules',
      header: t('colRules'),
      cell: (b) => <BatchRules batch={b} />,
      className: 'max-w-48',
    },
    { key: 'made', header: t('colMade'), cell: (b) => <BatchMade batch={b} now={now} /> },
  ];
  return (
    <DataTable
      caption={t('table')}
      rows={batches}
      rowKey={(b) => b.id}
      columns={columns}
      card={(b) => <BatchCard batch={b} now={now} />}
    />
  );
}
