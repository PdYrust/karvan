import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsLeftRightIcon, PlusIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { listView } from '../../lib/lists/list-view';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useNow } from '../overview/use-now';
import { useCurrentPanel } from '../panels/current-panel';
import { newMigrationPath } from '../wizard/steps';
import { useDeleteDraft } from './DeleteDraft';
import { stateFilters, statesOf } from './job-place';
import { JobsToolbar } from './JobsToolbar';
import { ToolbarAdd } from '../../components/data/ToolbarAdd';
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { ListBody } from './ListBody';
import { listMessages } from './list-messages';
import { MigrationsTable } from './MigrationsTable';
import { useJobList, useStateFilter } from './use-job-list';

/** Drafts, runs and finished jobs, newest first. */
export function MigrationsPage() {
  const t = useT(listMessages);
  usePageTitle(t('title'));
  const navigate = useNavigate();
  const now = useNow();
  const [search, setSearch] = useState('');
  const [state, setState] = useStateFilter(stateFilters);
  const { panel, settled } = useCurrentPanel();
  const list = useJobList(statesOf(state), panel?.id, search.trim() !== '', settled);
  const deleteDraft = useDeleteDraft();
  const { query } = list;
  const load = useFirstLoad(query.data);
  // The skeleton until the first page, then the empty card alone or the toolbar with the list.
  const view = listView(
    load.data?.jobs,
    list.filtered || list.offset > 0 || query.isPlaceholderData || search.trim() !== '',
  );
  let body;
  if (!load.data) {
    body = query.isError ? (
      <ErrorState
        message={t('loadFailed')}
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    ) : (
      <ListSkeleton label={t('title')} />
    );
  } else {
    body = (
      <Reveal on={load.reveal}>
        {view === 'list' ? (
          <JobsToolbar
            search={search}
            onSearch={setSearch}
            searchLabel={t('search')}
            states={stateFilters}
            state={state}
            onState={setState}
            refreshing={query.isFetching}
            onRefresh={() => void query.refetch()}
          >
            <ToolbarAdd label={t('newMigration')} onClick={() => void navigate(newMigrationPath)} />
          </JobsToolbar>
        ) : null}
        <ListBody
          list={list}
          page={load.data}
          search={search}
          noMatch={t('noMatch')}
          empty={
            <EmptyState
              icon={<ArrowsLeftRightIcon size={32} aria-hidden="true" />}
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
          }
        >
          {(shown) => <MigrationsTable jobs={shown} now={now} onDelete={deleteDraft.ask} />}
        </ListBody>
      </Reveal>
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex flex-col gap-4 p-4 sm:p-8">{body}</div>
      {deleteDraft.dialog}
    </div>
  );
}
