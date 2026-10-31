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
import { GroupSection } from './GroupHeading';
import { GroupView } from './GroupView';
import { matchesGroup } from './batch-view';
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

/** Every old panel, the latest run first, each with its newest runs; one old panel alone with `?group=`. */
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
  const { panel, settled } = useCurrentPanel();
  const searching = search.trim() !== '';
  const filter = state === 'all' ? undefined : state;
  const list = useGroupList(filter, panel?.id, searching, settled);
  const { query } = list;
  const load = useFirstLoad(query.data);
  const stateChoice = useStateChoice(state, setState);
  const panelChoice = usePanelChoice();
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
  const shown = load.data.groups.filter((g) => matchesGroup(g, search));
  return (
    <Reveal on={load.reveal}>
      {view === 'list' ? (
        <ListToolbar
          search={{ label: t('search'), value: search, onChange: setSearch }}
          choices={[stateChoice, panelChoice]}
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
        <div className="flex flex-col gap-10">
          {shown.map((g) => (
            <GroupSection key={g.id} group={g} state={filter} now={now} />
          ))}
        </div>
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
