import { Badge } from '@cloudflare/kumo/components/badge';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { useDeferredValue, useId, useMemo, useState } from 'react';
import { DataTable, type Column } from '../../components/data/DataTable';
import { Pager } from '../../components/data/Pager';
import { RowCard } from '../../components/data/RowCard';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { ListToolbar } from '../../components/data/ListToolbar';
import { dataMessages } from '../../components/data/messages';
import { useT } from '../../i18n/locale';
import type { JobState } from '../../lib/api/migrations';
import type { ReportUser } from '../../lib/api/report';
import { reportsMessages } from './messages';
import {
  filterCounts,
  userFilters,
  userMatches,
  userOutcome,
  type UserFilter,
} from './report-view';

const perPage = 50;

/** The old name, and the new one after an arrow when the run renamed the user. */
export function UserName({ user }: { user: ReportUser }) {
  const t = useT(reportsMessages);
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5">
      <bdi dir="ltr" className="truncate font-medium text-kumo-strong">
        {user.sourceUsername}
      </bdi>
      {user.username !== user.sourceUsername ? (
        <>
          <ArrowRightIcon
            size={12}
            aria-label={t('f_renamed')}
            className="text-kumo-subtle rtl:-scale-x-100"
          />
          <bdi dir="ltr" className="truncate text-kumo-default">
            {user.username}
          </bdi>
        </>
      ) : null}
    </span>
  );
}

/** The changes and fixes Karvan made to a user, and whether its config or link changes. */
function Changes({ user, state }: { user: ReportUser; state: JobState }) {
  const t = useT(reportsMessages);
  const changes = user.applied?.length ?? 0;
  const fixes = user.fixed?.length ?? 0;
  const flags = user.moved && state !== 'rolled_back';
  if (!changes && !fixes && !(flags && (user.changesConfig || user.changesLink))) {
    return <span className="text-kumo-subtle">{t('noChanges')}</span>;
  }
  return (
    <span className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-sm text-kumo-subtle">
      {changes > 0 ? <span>{t('changesCount', { count: changes })}</span> : null}
      {fixes > 0 ? <span>{t('fixesCount', { count: fixes })}</span> : null}
      {flags && user.changesConfig ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
      {flags && user.changesLink ? <Badge variant="outline">{t('linkFlag')}</Badge> : null}
    </span>
  );
}

const outcomeTones = {
  outcomeMoved: 'success',
  outcomeLeft: 'neutral',
  outcomeUndone: 'neutral',
} as const;

function Outcome({ user, state }: { user: ReportUser; state: JobState }) {
  const t = useT(reportsMessages);
  const key = userOutcome(user, state);
  return <StatusLabel tone={outcomeTones[key]}>{t(key)}</StatusLabel>;
}

function UsersTable({ users, state }: { users: readonly ReportUser[]; state: JobState }) {
  const t = useT(reportsMessages);
  const columns: Column<ReportUser>[] = [
    { key: 'user', header: t('colUser'), cell: (u) => <UserName user={u} />, className: 'w-2/5' },
    { key: 'outcome', header: t('colOutcome'), cell: (u) => <Outcome user={u} state={state} /> },
    { key: 'changes', header: t('colChanges'), cell: (u) => <Changes user={u} state={state} /> },
  ];
  return (
    <DataTable
      caption={t('usersTable')}
      rows={users}
      rowKey={(u) => u.key}
      columns={columns}
      card={(u) => (
        <RowCard lead={<UserName user={u} />} badge={<Outcome user={u} state={state} />}>
          <Changes user={u} state={state} />
        </RowCard>
      )}
    />
  );
}

/** Every user of the report: the list toolbar with a search and a filter with counts, 50 a page. */
export function ReportUsers({ users, state }: { users: readonly ReportUser[]; state: JobState }) {
  const t = useT(reportsMessages);
  const td = useT(dataMessages);
  const titleId = useId();
  const [filter, setFilter] = useState<UserFilter>('all');
  const [term, setTerm] = useState('');
  const [offset, setOffset] = useState(0);
  const search = useDeferredValue(term);
  const counts = useMemo(() => filterCounts(users), [users]);
  const shown = useMemo(
    () => users.filter((u) => userMatches(u, filter, search)),
    [users, filter, search],
  );
  const page = shown.slice(offset, offset + perPage);
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('users')}
      </h2>
      <ListToolbar
        search={{
          label: t('usersSearch'),
          value: term,
          onChange: (v) => {
            setTerm(v);
            setOffset(0);
          },
        }}
        choices={[
          {
            label: td('filter'),
            value: filter,
            all: 'all',
            options: userFilters.map((f) => {
              const text = f === 'all' ? td('showAll') : t(`f_${f}`);
              return { value: f, label: text, text, count: counts[f] };
            }),
            onChange: (f) => {
              setFilter(f as UserFilter);
              setOffset(0);
            },
          },
        ]}
      />
      {page.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('noUsers')}</p>
      ) : (
        <UsersTable users={page} state={state} />
      )}
      <Pager
        offset={offset}
        shown={page.length}
        total={shown.length}
        perPage={perPage}
        onOffset={setOffset}
      />
    </section>
  );
}
