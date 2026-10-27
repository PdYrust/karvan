import { FileTextIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { EmptyState } from '../../components/feedback/EmptyState';
import { useT } from '../../i18n/locale';
import { reportStates, statesOf, type StateFilter } from '../migrations/job-place';
import { JobsToolbar } from '../migrations/JobsToolbar';
import { ListBody, ListSkeleton } from '../migrations/ListBody';
import { useJobList, useStateFilter } from '../migrations/use-job-list';
import { useNow } from '../overview/use-now';
import { useCurrentPanel } from '../panels/current-panel';
import { reportsMessages } from '../migrations/report/messages';
import { ReportsTable } from './ReportsTable';

const filters: readonly StateFilter[] = ['all', 'done', 'failed', 'rolled_back'];

/** Finished jobs with their result, counts and files (4.7). */
export function ReportsPage() {
  const t = useT(reportsMessages);
  usePageTitle(t('title'));
  const now = useNow();
  const [search, setSearch] = useState('');
  const [state, setState] = useStateFilter(filters);
  const { panel, settled } = useCurrentPanel();
  const states = state === 'all' ? reportStates : statesOf(state);
  const list = useJobList(states, panel?.id, search.trim() !== '', settled);
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
              states={filters}
              state={state}
              onState={setState}
              refreshing={list.query.isFetching}
              onRefresh={() => void list.query.refetch()}
            />
            <ListBody
              list={{ ...list, filtered: state !== 'all' || Boolean(panel) }}
              search={search}
              failed={t('loadFailed')}
              noMatch={t('noMatch')}
              empty={
                <EmptyState
                  icon={<FileTextIcon size={32} aria-hidden="true" />}
                  title={t('emptyTitle')}
                >
                  {t('emptyBody')}
                </EmptyState>
              }
            >
              {(shown) => <ReportsTable jobs={shown} now={now} />}
            </ListBody>
          </>
        )}
      </div>
    </div>
  );
}
