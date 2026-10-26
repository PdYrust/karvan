import type { ReactNode } from 'react';
import { CursorPager } from '../../components/data/CursorPager';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { useT } from '../../i18n/locale';
import type { JobRow } from '../../lib/api/migrations';
import { matchesJob } from './job-place';
import { listMessages } from './list-messages';
import type { JobList } from './use-job-list';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';

/** The toolbar and a few rows, the shape the list takes. */
export function ListSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-4">
      <Bone className="h-8 w-full" />
      <Bone className="h-56 w-full rounded-lg" />
    </div>
  );
}

/** The rows that show, the line under them, and the empty or failed states. */
export function ListBody({
  list,
  search,
  empty,
  failed,
  noMatch,
  children,
}: {
  list: JobList;
  search: string;
  empty: ReactNode;
  failed: string;
  noMatch: string;
  children: (shown: JobRow[]) => ReactNode;
}) {
  const t = useT(listMessages);
  const { query } = list;
  const load = useFirstLoad(query.data);
  if (!load.data) {
    return query.isError ? (
      <ErrorState message={failed} error={query.error} onRetry={() => void query.refetch()} />
    ) : (
      <Bone className="h-56 w-full rounded-lg" />
    );
  }
  const shown = load.data.jobs.filter((j) => matchesJob(j, search));
  return (
    <Reveal on={load.reveal}>
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
          {list.hasNext ? t('searchDepth', { count: load.data.jobs.length }) : null}
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
