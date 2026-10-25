import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import { listJobs, type JobState } from '../../lib/api/migrations';
import type { StateFilter } from './job-place';

export const jobsPerPage = 20;
/** A search looks through this many of the latest jobs that match the filters. */
export const searchDepth = 100;

/**
 * One page of the jobs list. The server pages by cursor, so the cursors of
 * the pages already seen are kept to step back; new filters start again on
 * the first page. While searching, one deeper page is read instead.
 */
export function useJobList(
  states: readonly JobState[],
  panel: number | undefined,
  search: boolean,
  /** False while the current panel is not known yet, so the list does not first read every panel's jobs. */
  enabled = true,
) {
  const filterKey = `${states.join(',')}|${String(panel ?? 0)}`;
  const [paging, setPaging] = useState<{ key: string; cursors: (string | undefined)[] }>({
    key: filterKey,
    cursors: [undefined],
  });
  // Filters changed since the last page: back to the first one, during render (no effect).
  const cursors = paging.key === filterKey ? paging.cursors : [undefined];
  if (paging.key !== filterKey) setPaging({ key: filterKey, cursors });
  const cursor = search ? undefined : cursors.at(-1);
  const limit = search ? searchDepth : jobsPerPage;
  const query = useQuery({
    queryKey: ['migrations', 'list', { states, panel, cursor, limit }],
    queryFn: () => listJobs({ states, panel, cursor, limit }),
    placeholderData: keepPreviousData,
    enabled,
  });
  const next = query.data?.next;
  return {
    query,
    /** A filter is on, so an empty page does not mean there are no jobs. */
    filtered: states.length > 0 || Boolean(panel),
    offset: search ? 0 : (cursors.length - 1) * jobsPerPage,
    hasNext: Boolean(next),
    next: () => {
      if (next) setPaging({ key: filterKey, cursors: [...cursors, next] });
    },
    previous: () => {
      if (cursors.length > 1) setPaging({ key: filterKey, cursors: cursors.slice(0, -1) });
    },
  };
}

export type JobList = ReturnType<typeof useJobList>;

/** The state filter lives in the address (`?state=failed`), so Back and a shared link keep it. */
export function useStateFilter(allowed: readonly StateFilter[]) {
  const [params, setParams] = useSearchParams();
  const raw = params.get('state') as StateFilter | null;
  const state: StateFilter = raw && allowed.includes(raw) ? raw : 'all';
  const setState = (s: StateFilter) =>
    setParams(s === 'all' ? {} : { state: s }, { replace: true });
  return [state, setState] as const;
}
