import { Button } from '@cloudflare/kumo/components/button';
import { FileTextIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { DataTable, type Column } from '../../../components/data/DataTable';
import { EmptyState } from '../../../components/feedback/EmptyState';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { StatusLabel } from '../../../components/feedback/StatusBadge';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import { accountKeys, getAudit, type AuditEvent, type AuditFilter } from '../../../lib/api/account';
import { useFormat } from '../../../lib/preferences/region';
import { AuditFilters } from './AuditFilters';
import { auditMessages } from './messages';
import { actionMessages, isAuditAction, isAuditReason, reasonMessages } from './words';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

const none: AuditFilter = { action: '', outcome: '' };

/** The action in words, and for a failure why it failed. */
function What({ e }: { e: AuditEvent }) {
  const ta = useT(actionMessages);
  const tr = useT(reasonMessages);
  const reason = e.detail.reason;
  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-medium text-kumo-strong">
        {isAuditAction(e.action) ? ta(e.action) : <Machine>{e.action}</Machine>}
      </span>
      {e.outcome === 'failure' && isAuditReason(reason) ? (
        <span className="text-sm text-kumo-subtle">{tr(reason)}</span>
      ) : null}
    </span>
  );
}

function Outcome({ e }: { e: AuditEvent }) {
  const t = useT(auditMessages);
  return e.outcome === 'success' ? (
    <StatusLabel tone="success">{t('success')}</StatusLabel>
  ) : (
    <StatusLabel tone="danger">{t('failure')}</StatusLabel>
  );
}

function When({ at }: { at: string }) {
  const fmt = useFormat();
  const d = new Date(at);
  return (
    <time dateTime={d.toISOString()} className="whitespace-nowrap tabular">
      {fmt.dateTime(d)}
    </time>
  );
}

/** Who did what and when, newest first, with filters and "Load more". */
export function AuditTab() {
  const t = useT(auditMessages);
  const [filter, setFilter] = useState<AuditFilter>(none);
  const log = useInfiniteQuery({
    queryKey: [...accountKeys.audit, filter],
    queryFn: ({ pageParam }) => getAudit(filter, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => last.next ?? undefined,
  });
  const filtered = filter.action !== '' || filter.outcome !== '';
  const events = log.data?.pages.flatMap((p) => p.events) ?? [];
  const columns: Column<AuditEvent>[] = [
    { key: 'when', header: t('when'), cell: (e) => <When at={e.at} /> },
    { key: 'what', header: t('what'), cell: (e) => <What e={e} /> },
    { key: 'outcome', header: t('outcome'), cell: (e) => <Outcome e={e} /> },
    { key: 'who', header: t('who'), cell: (e) => <Machine>{e.actor}</Machine> },
    { key: 'ip', header: t('ip'), cell: (e) => <Machine>{e.ip}</Machine> },
  ];

  const load = useFirstLoad(log.data);
  function body() {
    if (log.isPending || load.holding) {
      return (
        <div role="status" aria-label={t('caption')}>
          <Bone className="h-64 w-full" />
        </div>
      );
    }
    if (log.isError) {
      return (
        <ErrorState
          message={t('loadFailed')}
          error={log.error}
          onRetry={() => void log.refetch()}
        />
      );
    }
    if (events.length === 0) {
      return filtered ? (
        <EmptyState
          icon={<MagnifyingGlassIcon size={32} />}
          title={t('noMatchTitle')}
          action={<Button onClick={() => setFilter(none)}>{t('clearFilters')}</Button>}
        >
          {t('noMatchBody')}
        </EmptyState>
      ) : (
        <EmptyState icon={<FileTextIcon size={32} />} title={t('emptyTitle')}>
          {t('emptyBody')}
        </EmptyState>
      );
    }
    return (
      <>
        <DataTable
          caption={t('caption')}
          rows={events}
          rowKey={(e) => e.id}
          columns={columns}
          card={(e) => (
            <div className="flex flex-col gap-2 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline">
              <div className="flex items-start justify-between gap-3">
                <What e={e} />
                <Outcome e={e} />
              </div>
              <span className="text-sm text-kumo-subtle">
                <When at={e.at} />
              </span>
              <span className="flex flex-wrap gap-x-4 gap-y-1 text-sm">
                <Machine>{e.actor}</Machine>
                <Machine>{e.ip}</Machine>
              </span>
            </div>
          )}
        />
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-kumo-subtle" aria-live="polite">
            {t('shown', { count: events.length })}
          </p>
          {log.hasNextPage ? (
            <Button
              className="pointer-coarse:h-9"
              loading={log.isFetchingNextPage}
              onClick={() => void log.fetchNextPage()}
            >
              {t('loadMore')}
            </Button>
          ) : null}
        </div>
      </>
    );
  }

  return (
    <section aria-labelledby="audit-title" className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 id="audit-title" className="text-heading font-semibold text-kumo-strong">
          {t('title')}
        </h2>
        <p className="max-w-prose text-sm text-kumo-subtle">{t('description')}</p>
      </div>
      <AuditFilters filter={filter} onChange={setFilter} />
      <Reveal on={load.reveal}>{body()}</Reveal>
    </section>
  );
}
