import { ArrowRightIcon, StackIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { Section } from '../../components/layout/Section';
import { usePageTitle } from '../../components/layout/use-page-title';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { batchKeys, getBatch, type Batch } from '../../lib/api/batches';
import { errorOf } from '../../lib/errors/errors';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useBatchActions } from './BatchActions';
import { BatchAside } from './BatchAside';
import { AdminName, BatchMove, BatchStateBadge } from './BatchCells';
import { BatchUserList } from './BatchUserList';
import { batchesMessages } from './messages';

/** The move, the old panel's address and the new panel. */
function HeaderLine({ batch }: { batch: Batch }) {
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
      <BatchMove batch={batch} size={20} />
      <Machine>{batch.source.origin}</Machine>
      <ArrowRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
      <bdi>{batch.panel.name}</bdi>
    </span>
  );
}

/** One row per admin: the old name, the new one, and its users. */
function Admins({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  return (
    <Section title={t('admins')}>
      <ul className="flex flex-col px-4 pb-2">
        {batch.admins.map((a) => (
          <li
            key={a.source}
            className="flex items-center justify-between gap-4 border-b border-kumo-hairline py-3 last:border-0"
          >
            <span className="min-w-0 font-medium text-kumo-strong">
              <AdminName admin={a} />
            </span>
            <span className="shrink-0 text-kumo-subtle">{t('users', { count: a.users })}</span>
          </li>
        ))}
      </ul>
    </Section>
  );
}

/** The main column and the aside, split by a hairline as on the report page. */
function Body({ batch, failed }: { batch: Batch; failed?: string }) {
  const t = useT(batchesMessages);
  const reason = batch.state === 'inactive' && batch.inactiveReason ? batch.inactiveReason : null;
  return (
    <div className="grid lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8">
        {failed ? <InlineAlert>{failed}</InlineAlert> : null}
        {reason ? (
          <InlineAlert tone="warning">{`${t('inactiveNote')} ${t(`why_${reason}`)}`}</InlineAlert>
        ) : null}
        <Admins batch={batch} />
        <BatchUserList batchId={batch.id} />
      </div>
      <aside
        aria-label={t('facts')}
        className="border-t border-kumo-hairline p-4 sm:p-8 lg:col-start-2 lg:row-start-1 lg:border-s lg:border-t-0"
      >
        <BatchAside batch={batch} />
      </aside>
    </div>
  );
}

function Loaded({ batch, reveal }: { batch: Batch; reveal: boolean }) {
  const { actions, dialogs, failed } = useBatchActions(batch);
  return (
    <>
      <PageHeader
        eyebrow={<Eyebrow />}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <bdi className="min-w-0 break-all">{batch.name}</bdi>
            <BatchStateBadge batch={batch} />
          </span>
        }
        description={<HeaderLine batch={batch} />}
        actions={actions}
      />
      <Reveal on={reveal}>
        <Body batch={batch} failed={failed} />
      </Reveal>
      {dialogs}
    </>
  );
}

function Eyebrow() {
  const t = useT(batchesMessages);
  return (
    <Link
      to="/batches"
      className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
    >
      {t('title')}
    </Link>
  );
}

/** One batch: its admins, users and old links, with rename, delete and make active again. */
export function BatchPage() {
  const { id = '' } = useParams();
  const t = useT(batchesMessages);
  const num = Number(id);
  const valid = Number.isInteger(num) && num > 0;
  const q = useQuery({
    queryKey: batchKeys.batch(num),
    queryFn: () => getBatch(num),
    enabled: valid,
  });
  const load = useFirstLoad(q.data);
  usePageTitle(load.data?.name ?? t('title'));
  if (load.data)
    return (
      <div className="flex flex-col">
        <Loaded batch={load.data} reveal={load.reveal} />
      </div>
    );
  const missing = !valid || (q.isError && errorOf(q.error).code === 'not_found');
  return (
    <div className="flex flex-col">
      <PageHeader eyebrow={<Eyebrow />} title={t('title')} />
      <div className="flex flex-col gap-6 p-4 sm:p-8">
        {missing ? (
          <EmptyState icon={<StackIcon size={32} aria-hidden="true" />} title={t('notFound')} />
        ) : q.isError ? (
          <ErrorState message={t('batchFailed')} error={q.error} onRetry={() => void q.refetch()} />
        ) : (
          <div role="status" aria-label={t('title')} className="flex flex-col gap-4">
            <Bone className="h-24 w-full" />
            <Bone className="h-64 w-full" />
          </div>
        )}
      </div>
    </div>
  );
}
