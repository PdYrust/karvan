import { Button } from '@cloudflare/kumo/components/button';
import { PlusIcon, StackIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { CursorPager } from '../../components/data/CursorPager';
import { ListToolbar } from '../../components/data/ListToolbar';
import { ToolbarRefresh } from '../../components/data/ToolbarRefresh';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { listView } from '../../lib/lists/list-view';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { usePanelChoice } from '../migrations/JobsToolbar';
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { useNow } from '../overview/use-now';
import { useCurrentPanel } from '../panels/current-panel';
import { newMigrationPath } from '../wizard/steps';
import { GroupView } from './GroupView';
import { matchesGroup, newestBatches } from './batch-view';
import { BatchesTable } from './BatchesTable';
import { useOldPanelChoice } from './old-panel-choice';
import { batchesMessages } from './messages';
import { useStateChoice } from './state-choice';
import { useBatchStateFilter, useGroupList, type BatchStateFilter } from './use-batches';

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

/** Batches newest first in one table; one old panel alone with `?group=`. */
export function BatchesPage() {
  const t = useT(batchesMessages);
  usePageTitle(t('title'));
  const [state, setState, group] = useBatchStateFilter();
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex flex-col gap-4 p-4 sm:p-8">
        {group ? (
          <GroupView id={group} state={state} setState={setState} />
        ) : (
          <GroupList state={state} setState={setState} />
        )}
      </div>
    </div>
  );
}

function GroupList({
  state,
  setState,
}: {
  state: BatchStateFilter;
  setState: (s: BatchStateFilter) => void;
}) {
  const t = useT(batchesMessages);
  const now = useNow();
  const [search, setSearch] = useState('');
  const [oldPanel, setOldPanel] = useState('all');
  const { panel, settled } = useCurrentPanel();
  // An old-panel filter reads the searched page (the latest 100 groups), not
  // just the first page of 10, so an address is not missed for paging alone.
  const searching = search.trim() !== '' || oldPanel !== 'all';
  const filter = state === 'all' ? undefined : state;
  const list = useGroupList(filter, panel?.id, searching, settled);
  const { query } = list;
  const load = useFirstLoad(query.data);
  const stateChoice = useStateChoice(state, setState);
  const panelChoice = usePanelChoice();
  const oldPanelChoice = useOldPanelChoice(query.data?.groups ?? [], oldPanel, setOldPanel);
  // The skeleton until the first page, then the empty card alone or the toolbar with the list.
  const view = listView(
    load.data?.groups,
    list.filtered || list.offset > 0 || query.isPlaceholderData || searching,
  );
  if (!load.data) {
    return query.isError ? (
      <ErrorState
        message={t('loadFailed')}
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    ) : (
      <ListSkeleton label={t('title')} />
    );
  }
  const shownGroups = load.data.groups.filter(
    (g) => (oldPanel === 'all' || g.source.origin === oldPanel) && matchesGroup(g, search),
  );
  // One table, not one per old panel: every shown group's newest runs, newest
  // first by the date they were made, in the table's usual column order.
  // No footer links under it: the full runs of an old panel stay reachable
  // from any batch's page, through its old-panel link.
  const shown = newestBatches(shownGroups.flatMap((g) => g.runs));
  return (
    <Reveal on={load.reveal}>
      {view === 'list' ? (
        <ListToolbar
          search={{ label: t('search'), value: search, onChange: setSearch }}
          choices={[stateChoice, panelChoice, oldPanelChoice]}
          refresh={
            <ToolbarRefresh
              label={t('refresh')}
              loading={query.isFetching}
              onClick={() => void query.refetch()}
            />
          }
        />
      ) : null}
      {shown.length > 0 ? (
        <BatchesTable batches={shown} now={now} />
      ) : list.filtered || searching ? (
        <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
          {t('noGroupMatch')}
        </p>
      ) : (
        <Empty />
      )}
      {searching ? (
        <p className="text-sm text-kumo-subtle" aria-live="polite">
          {list.hasNext ? t('searchDepthGroups', { count: load.data.groups.length }) : null}
        </p>
      ) : list.offset > 0 || list.hasNext ? (
        // The pager only when there is another page to go to: on a single
        // page its "showing all" text says nothing and counts old panels,
        // not the table's rows.
        <CursorPager
          offset={list.offset}
          shown={shownGroups.length}
          hasNext={list.hasNext}
          onPrevious={list.previous}
          onNext={list.next}
        />
      ) : null}
    </Reveal>
  );
}
