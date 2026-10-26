import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsLeftRightIcon, PlusIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { useNow } from '../overview/use-now';
import { useCurrentPanel } from '../panels/current-panel';
import { newMigrationPath } from '../wizard/steps';
import { stateFilters, statesOf } from './job-place';
import { JobsToolbar } from './JobsToolbar';
import { ListBody, ListSkeleton } from './ListBody';
import { listMessages } from './list-messages';
import { MigrationsTable } from './MigrationsTable';
import { useJobList, useStateFilter } from './use-job-list';

/** Drafts, runs and finished jobs, newest first (4.4). */
export function MigrationsPage() {
  const t = useT(listMessages);
  usePageTitle(t('title'));
  const navigate = useNavigate();
  const now = useNow();
  const [search, setSearch] = useState('');
  const [state, setState] = useStateFilter(stateFilters);
  const { panel, settled } = useCurrentPanel();
  const list = useJobList(statesOf(state), panel?.id, search.trim() !== '', settled);
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex max-w-[1400px] flex-col gap-4 p-4 sm:p-8">
        {!settled ? (
          <ListSkeleton label={t('title')} />
        ) : (
          <>
            <JobsToolbar
              search={search}
              onSearch={setSearch}
              searchLabel={t('search')}
              states={stateFilters}
              state={state}
              onState={setState}
              refreshing={list.query.isFetching}
              onRefresh={() => void list.query.refetch()}
            >
              <Button
                variant="primary"
                shape="square"
                aria-label={t('newMigration')}
                title={t('newMigration')}
                icon={<PlusIcon size={16} />}
                className="pointer-coarse:size-11"
                onClick={() => void navigate(newMigrationPath)}
              />
            </JobsToolbar>
            <ListBody
              list={list}
              search={search}
              failed={t('loadFailed')}
              noMatch={t('noMatch')}
              empty={
                <EmptyState
                  icon={<ArrowsLeftRightIcon size={32} aria-hidden="true" />}
                  title={t('emptyTitle')}
                  action={
                    <Button
                      icon={<PlusIcon size={16} />}
                      className="pointer-coarse:h-11"
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
              {(shown) => <MigrationsTable jobs={shown} now={now} />}
            </ListBody>
          </>
        )}
      </div>
    </div>
  );
}
