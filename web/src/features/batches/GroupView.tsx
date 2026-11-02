import { Button } from '@cloudflare/kumo/components/button';
import { ArrowLeftIcon, StackIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { CursorPager } from '../../components/data/CursorPager';
import { ListToolbar } from '../../components/data/ListToolbar';
import { ToolbarRefresh } from '../../components/data/ToolbarRefresh';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { useT } from '../../i18n/locale';
import { errorOf } from '../../lib/errors/errors';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useNow } from '../overview/use-now';
import { BatchesTable } from './BatchesTable';
import { matchesBatch } from './batch-view';
import { batchesMessages } from './messages';
import { useStateChoice } from './state-choice';
import { useBatchList, useGroup, type BatchStateFilter } from './use-batches';

function Back() {
  const t = useT(batchesMessages);
  return (
    <Link
      to="/batches"
      className="inline-flex items-center gap-1.5 self-start text-sm text-kumo-subtle hover:text-kumo-default pointer-coarse:min-h-9"
    >
      <ArrowLeftIcon size={14} aria-hidden="true" className="rtl:-scale-x-100" />
      {t('allOldPanels')}
    </Link>
  );
}

/** One old panel alone: every run of it, a page at a time. */
export function GroupView({
  id,
  state,
  setState,
}: {
  id: number;
  state: BatchStateFilter;
  setState: (s: BatchStateFilter) => void;
}) {
  const t = useT(batchesMessages);
  const now = useNow();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const searching = search.trim() !== '';
  const group = useGroup(id);
  const list = useBatchList(state === 'all' ? undefined : state, id, searching);
  const { query } = list;
  const load = useFirstLoad(group.data && query.data);
  const stateChoice = useStateChoice(state, setState);
  const failed = group.isError ? group : query.isError ? query : undefined;
  if (failed) {
    return errorOf(failed.error).code === 'not_found' ? (
      <EmptyState
        icon={<StackIcon size={32} aria-hidden="true" />}
        title={t('groupMissing')}
        action={<Button onClick={() => void navigate('/batches')}>{t('allOldPanels')}</Button>}
      />
    ) : (
      <ErrorState
        message={t('groupFailed')}
        error={failed.error}
        onRetry={() => void failed.refetch()}
      />
    );
  }
  if (!load.data || !group.data) return <ListSkeleton label={t('title')} />;
  const shown = load.data.batches.filter((b) => matchesBatch(b, search));
  return (
    <Reveal on={load.reveal}>
      <Back />
      <ListToolbar
        search={{ label: t('search'), value: search, onChange: setSearch }}
        choices={[stateChoice]}
        refresh={
          <ToolbarRefresh
            label={t('refresh')}
            loading={query.isFetching || group.isFetching}
            onClick={() => {
              void query.refetch();
              void group.refetch();
            }}
          />
        }
      />
      {shown.length > 0 ? (
        <BatchesTable batches={shown} now={now} />
      ) : (
        <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
          {t('noMatch')}
        </p>
      )}
      {searching ? (
        <p className="text-sm text-kumo-subtle" aria-live="polite">
          {list.hasNext ? t('searchDepth', { count: load.data.batches.length }) : null}
        </p>
      ) : (
        <CursorPager
          offset={list.offset}
          shown={shown.length}
          hasNext={list.hasNext}
          onPrevious={list.previous}
          onNext={list.next}
        />
      )}
    </Reveal>
  );
}
