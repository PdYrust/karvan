import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import { batchKeys, listBatches, listBatchUsers, type Batch } from '../../lib/api/batches';

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

/** One page of batches; while searching, one deeper page is read instead. */
export function useBatchList(
  state: Batch['state'] | undefined,
  panel: number | undefined,
  search: boolean,
  enabled: boolean,
) {
  const paging = useCursors(`${state ?? ''}|${String(panel ?? 0)}`);
  const f = {
    state,
    panel,
    cursor: search ? undefined : paging.cursor,
    limit: search ? searchDepth : batchesPerPage,
  };
  const query = useQuery({
    queryKey: batchKeys.list(f),
    queryFn: () => listBatches(f),
    placeholderData: keepPreviousData,
    enabled,
  });
  const next = query.data?.next;
  return {
    query,
    filtered: Boolean(state) || Boolean(panel),
    offset: search ? 0 : paging.page * batchesPerPage,
    hasNext: Boolean(next),
    next: () => paging.next(next),
    previous: paging.previous,
  };
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

/** The state filter lives in the address (`?state=inactive`), so Back and a shared link keep it. */
export function useBatchStateFilter() {
  const [params, setParams] = useSearchParams();
  const raw = params.get('state');
  const state: BatchStateFilter = raw === 'active' || raw === 'inactive' ? raw : 'all';
  const setState = (s: BatchStateFilter) =>
    setParams(s === 'all' ? {} : { state: s }, { replace: true });
  return [state, setState] as const;
}
