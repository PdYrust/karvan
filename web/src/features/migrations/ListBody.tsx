import type { ReactNode } from 'react';
import { CursorPager } from '../../components/data/CursorPager';
import { useT } from '../../i18n/locale';
import type { JobPage, JobRow } from '../../lib/api/migrations';
import { matchesJob } from './job-place';
import { listMessages } from './list-messages';
import type { JobList } from './use-job-list';

/** The rows that show, the line under them, and the empty state, once the first page is there. */
export function ListBody({
  list,
  page,
  search,
  empty,
  noMatch,
  children,
}: {
  list: JobList;
  page: JobPage;
  search: string;
  empty: ReactNode;
  noMatch: string;
  children: (shown: JobRow[]) => ReactNode;
}) {
  const t = useT(listMessages);
  const shown = page.jobs.filter((j) => matchesJob(j, search));
  return (
    <>
      {shown.length > 0 ? (
        children(shown)
      ) : list.filtered || search ? (
        <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
          {noMatch}
        </p>
      ) : (
        empty
      )}
      {search.trim() ? (
        <p className="text-sm text-kumo-subtle" aria-live="polite">
          {list.hasNext ? t('searchDepth', { count: page.jobs.length }) : null}
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
    </>
  );
}
