import { Badge } from '@cloudflare/kumo/components/badge';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { useEffect, useState, type ReactNode } from 'react';
import { Pager } from '../../components/data/Pager';
import { SearchField } from '../../components/data/SearchField';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { useT } from '../../i18n/locale';
import { getUsers, migrationKeys, usersPerPage, type UserRow } from '../../lib/api/migrations';
import { wizardMessages } from './messages';

/** A search term that settles after the operator stops typing for a moment. */
export function useSettled(value: string, ms = 300): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setSettled(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return settled;
}

/** The name a user has now and, when the plan renames them, the new one. */
function UserName({ row }: { row: UserRow }) {
  const t = useT(wizardMessages);
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5">
      <bdi dir="ltr" className="truncate font-medium text-kumo-strong">
        {row.sourceUsername}
      </bdi>
      {row.username !== row.sourceUsername ? (
        <>
          <ArrowRightIcon
            size={12}
            aria-hidden="true"
            className="text-kumo-subtle rtl:-scale-x-100"
          />
          <bdi dir="ltr" className="truncate text-kumo-default">
            {row.username}
          </bdi>
        </>
      ) : null}
      {row.include ? null : <Badge variant="outline">{t('leftOut')}</Badge>}
    </span>
  );
}

/**
 * Users of one group or category, read a page at a time with a search on
 * the server (D5). `end` draws what sits at the end of each row.
 */
export function UserList({
  jobId,
  group,
  label,
  empty,
  end,
}: {
  jobId: string;
  /** As GET users takes it: "", "C02", "category:vip". */
  group: string;
  label: string;
  /** Said when the group has nobody and no search is on. */
  empty: string;
  end?: (row: UserRow) => ReactNode;
}) {
  const t = useT(wizardMessages);
  const [term, setTerm] = useState('');
  const [offset, setOffset] = useState(0);
  const q = useSettled(term.trim());
  const query = { group, q, offset };
  const users = useQuery({
    queryKey: migrationKeys.users(jobId, query),
    queryFn: () => getUsers(jobId, query),
    placeholderData: keepPreviousData,
  });
  const rows = users.data?.users ?? [];
  return (
    <div className="flex flex-col gap-3">
      <SearchField
        label={t('search')}
        value={term}
        onChange={(v) => {
          setTerm(v);
          setOffset(0);
        }}
        className="sm:max-w-sm"
      />
      {users.isPending ? (
        <div role="status" aria-label={label} className="flex flex-col gap-2">
          <Bone className="h-12 w-full" />
          <Bone className="h-12 w-full" />
        </div>
      ) : users.isError ? (
        <ErrorState
          message={t('usersFailed')}
          error={users.error}
          onRetry={() => void users.refetch()}
        />
      ) : rows.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">
          {q ? t('noMatch') : empty}
        </p>
      ) : (
        <ul aria-label={label} className="flex flex-col rounded-lg ring ring-kumo-hairline">
          {rows.map((row) => (
            <li
              key={row.key}
              className="flex min-h-12 flex-wrap items-center gap-x-4 gap-y-2 border-t border-kumo-hairline px-3 py-2 first:border-0"
            >
              <UserName row={row} />
              {end ? <div className="ms-auto flex items-center gap-2">{end(row)}</div> : null}
            </li>
          ))}
        </ul>
      )}
      {users.data ? (
        <Pager
          offset={offset}
          shown={rows.length}
          total={users.data.total}
          perPage={usersPerPage}
          onOffset={setOffset}
        />
      ) : null}
    </div>
  );
}
