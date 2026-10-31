import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import {
  batchKeys,
  getBatchGroup,
  listBatches,
  listBatchGroups,
  listBatchUsers,
  type Batch,
} from '../../lib/api/batches';

const batchesPerPage = 20;
export const usersPerPage = 50;
/** A search looks through this many of the latest batches that match the filters. */
export const searchDepth = 100;

/**
 * Cursor paging: the cursors of the pages already seen are kept to step
 * back, and a new key (filters or search) starts again on the first page.
 */
function useCursors(key: string) {
  const [paging, setPaging] = useState<{ key: string; cursors: (string | undefined)[] }>({
    key,
    cursors: [undefined],
  });
  // The key changed since the last page: back to the first one, during render (no effect).
  const cursors = paging.key === key ? paging.cursors : [undefined];
  if (paging.key !== key) setPaging({ key, cursors });
  return {
    cursor: cursors.at(-1),
    page: cursors.length - 1,
    next: (c: string | undefined) => {
      if (c) setPaging({ key, cursors: [...cursors, c] });
    },
    previous: () => {
      if (cursors.length > 1) setPaging({ key, cursors: cursors.slice(0, -1) });
    },
  };
}

/** One page of one old panel's batches; while searching, one deeper page is read instead. */
export function useBatchList(state: Batch['state'] | undefined, group: number, search: boolean) {
  const paging = useCursors(`${state ?? ''}|${String(group)}`);
  const f = {
    state,
    group,
    cursor: search ? undefined : paging.cursor,
    limit: search ? searchDepth : batchesPerPage,
  };
  const query = useQuery({
    queryKey: batchKeys.list(f),
    queryFn: () => listBatches(f),
    placeholderData: keepPreviousData,
  });
  const next = query.data?.next;
  return {
    query,
    filtered: Boolean(state),
    offset: search ? 0 : paging.page * batchesPerPage,
    hasNext: Boolean(next),
    next: () => paging.next(next),
    previous: paging.previous,
  };
}

const groupsPerPage = 10;

/** One page of old panels with their newest runs; while searching, one deeper page instead. */
export function useGroupList(
  state: Batch['state'] | undefined,
  panel: number | undefined,
  search: boolean,
  enabled: boolean,
) {
  const paging = useCursors(`${state ?? ''}|${String(panel ?? 0)}`);
  const f = {
    state,
    panel,
    runs: groupRuns,
    cursor: search ? undefined : paging.cursor,
    limit: search ? searchDepth : groupsPerPage,
  };
  const query = useQuery({
    queryKey: batchKeys.groups(f),
    queryFn: () => listBatchGroups(f),
    placeholderData: keepPreviousData,
    enabled,
  });
  const next = query.data?.next;
  return {
    query,
    filtered: Boolean(state) || Boolean(panel),
    offset: search ? 0 : paging.page * groupsPerPage,
    hasNext: Boolean(next),
    next: () => paging.next(next),
    previous: paging.previous,
  };
}

/** How many of an old panel's newest runs the list shows under it. */
export const groupRuns = 5;

/** One old panel, by any of its batches. */
export function useGroup(id: number) {
  return useQuery({ queryKey: batchKeys.group(id), queryFn: () => getBatchGroup(id) });
}

/** One page of a batch's users; the server searches. */
export function useBatchUsers(id: number, q: string) {
  const term = q.trim();
  const paging = useCursors(term);
  const query = useQuery({
    queryKey: batchKeys.users(id, { q: term, cursor: paging.cursor }),
    queryFn: () => listBatchUsers(id, { q: term, cursor: paging.cursor, limit: usersPerPage }),
    placeholderData: keepPreviousData,
  });
  const next = query.data?.next;
  return {
    query,
    offset: paging.page * usersPerPage,
    hasNext: Boolean(next),
    next: () => paging.next(next),
    previous: paging.previous,
  };
}

const states = ['all', 'active', 'inactive'] as const;
export type BatchStateFilter = (typeof states)[number];
export const batchStateFilters: readonly BatchStateFilter[] = states;

/**
 * The state filter and the old panel shown alone live in the address
 * (`?state=inactive&group=12`), so Back and a shared link keep them.
 */
export function useBatchStateFilter() {
  const [params, setParams] = useSearchParams();
  const raw = params.get('state');
  const state: BatchStateFilter = raw === 'active' || raw === 'inactive' ? raw : 'all';
  const g = Number(params.get('group'));
  const group = Number.isInteger(g) && g > 0 ? g : 0;
  const setState = (s: BatchStateFilter) =>
    setParams(
      {
        ...(s === 'all' ? {} : { state: s }),
        ...(group ? { group: String(group) } : {}),
      },
      { replace: true },
    );
  return [state, setState, group] as const;
}
