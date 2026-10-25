import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { useDeferredValue, useId, useMemo, useState } from 'react';
import { Pager } from '../../components/data/Pager';
import { SearchField } from '../../components/data/SearchField';
import { useT } from '../../i18n/locale';
import type { JobState } from '../../lib/api/migrations';
import type { ReportUser } from '../../lib/api/report';
import { useFormat } from '../../lib/preferences/region';
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

function Row({ user, state }: { user: ReportUser; state: JobState }) {
  const t = useT(reportsMessages);
  const changes = user.applied?.length ?? 0;
  const fixes = user.fixed?.length ?? 0;
  return (
    <li className="flex min-h-12 flex-col gap-1.5 border-t border-kumo-hairline px-3 py-2.5 first:border-0 sm:flex-row sm:items-center sm:gap-4">
      <span className="min-w-0 sm:w-2/5">
        <UserName user={user} />
      </span>
      <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-kumo-subtle">
        <span className="text-kumo-default">{t(userOutcome(user, state))}</span>
        {changes > 0 ? <span>{t('changesCount', { count: changes })}</span> : null}
        {fixes > 0 ? <span>{t('fixesCount', { count: fixes })}</span> : null}
      </span>
      {user.moved && state !== 'rolled_back' && (user.changesConfig || user.changesLink) ? (
        <span className="flex flex-wrap gap-1.5 sm:ms-auto">
          {user.changesConfig ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
          {user.changesLink ? <Badge variant="outline">{t('linkFlag')}</Badge> : null}
        </span>
      ) : null}
    </li>
  );
}

/** Every user of the report: filter buttons with counts, a search, 50 a page. */
export function ReportUsers({ users, state }: { users: readonly ReportUser[]; state: JobState }) {
  const t = useT(reportsMessages);
  const fmt = useFormat();
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
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div role="group" aria-label={t('filtersLabel')} className="flex flex-wrap gap-2">
          {userFilters.map((f) => (
            <Button
              key={f}
              aria-pressed={filter === f}
              variant={filter === f ? 'secondary' : 'ghost'}
              className="pointer-coarse:h-11"
              onClick={() => {
                setFilter(f);
                setOffset(0);
              }}
            >
              {t(`f_${f}`)}
              <span className="tabular opacity-80">{fmt.number(counts[f])}</span>
            </Button>
          ))}
        </div>
        <SearchField
          label={t('usersSearch')}
          value={term}
          onChange={(v) => {
            setTerm(v);
            setOffset(0);
          }}
          className="lg:max-w-xs"
        />
      </div>
      {page.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('noUsers')}</p>
      ) : (
        <ul
          aria-label={t('usersTable')}
          className="flex flex-col rounded-lg bg-kumo-base ring ring-kumo-hairline"
        >
          {page.map((u) => (
            <Row key={u.key} user={u} state={state} />
          ))}
        </ul>
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
