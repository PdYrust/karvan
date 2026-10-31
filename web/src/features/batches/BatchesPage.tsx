import { Button } from '@cloudflare/kumo/components/button';
import { PlusIcon, StackIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { CursorPager } from '../../components/data/CursorPager';
import { ListToolbar, type ToolbarChoice } from '../../components/data/ListToolbar';
import { ToolbarRefresh } from '../../components/data/ToolbarRefresh';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { usePanelChoice } from '../migrations/JobsToolbar';
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { useNow } from '../overview/use-now';
import { useCurrentPanel } from '../panels/current-panel';
import { newMigrationPath } from '../wizard/steps';
import { BatchesTable } from './BatchesTable';
import { matchesBatch } from './batch-view';
import { batchesMessages } from './messages';
import {
  batchStateFilters,
  useBatchList,
  useBatchStateFilter,
  type BatchStateFilter,
} from './use-batches';

function Empty() {
  const t = useT(batchesMessages);
  const navigate = useNavigate();
  return (
    <EmptyState
      icon={<StackIcon size={32} aria-hidden="true" />}
      title={t('emptyTitle')}
      action={
        <Button
          icon={<PlusIcon size={16} />}
          className="pointer-coarse:h-9"
          onClick={() => void navigate(newMigrationPath)}
        >
          {t('newMigration')}
        </Button>
      }
    >
      {t('emptyBody')}
    </EmptyState>
  );
}

/** Every batch, newest first, with the panel and state filters. */
export function BatchesPage() {
  const t = useT(batchesMessages);
  usePageTitle(t('title'));
  const now = useNow();
  const [search, setSearch] = useState('');
  const [state, setState] = useBatchStateFilter();
  const { panel, settled } = useCurrentPanel();
  const searching = search.trim() !== '';
  const list = useBatchList(state === 'all' ? undefined : state, panel?.id, searching, settled);
  const { query } = list;
  const load = useFirstLoad(query.data);
  const panelChoice = usePanelChoice();
  // No batch at all: only the empty card shows, without the toolbar.
  const bare =
    query.data?.batches.length === 0 &&
    !query.isPlaceholderData &&
    !list.filtered &&
    list.offset === 0 &&
    !searching;
  const choices: ToolbarChoice[] = [
    {
      label: t('stateLabel'),
      value: state,
      all: 'all',
      options: batchStateFilters.map((s) => {
        const text = t(`filter_${s}`);
        return { value: s, label: text, text };
      }),
      onChange: (s) => setState(s as BatchStateFilter),
    },
    panelChoice,
  ];
  let body;
  if (!load.data) {
    body = query.isError ? (
      <ErrorState
        message={t('loadFailed')}
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    ) : (
      <Bone className="h-56 w-full rounded-lg" />
    );
  } else {
    const shown = load.data.batches.filter((b) => matchesBatch(b, search));
    body = (
      <Reveal on={load.reveal}>
        {shown.length > 0 ? (
          <BatchesTable batches={shown} now={now} />
        ) : list.filtered || searching ? (
          <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
            {t('noMatch')}
          </p>
        ) : (
          <Empty />
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
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex flex-col gap-4 p-4 sm:p-8">
        {!settled ? (
          <ListSkeleton label={t('title')} />
        ) : (
          <>
            {bare ? null : (
              <ListToolbar
                search={{ label: t('search'), value: search, onChange: setSearch }}
                choices={choices}
                refresh={
                  <ToolbarRefresh
                    label={t('refresh')}
                    loading={query.isFetching}
                    onClick={() => void query.refetch()}
                  />
                }
              />
            )}
            {body}
          </>
        )}
      </div>
    </div>
  );
}
