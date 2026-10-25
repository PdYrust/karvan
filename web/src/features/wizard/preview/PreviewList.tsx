import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { SearchField } from '../../../components/data/SearchField';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { useT } from '../../../i18n/locale';
import { usersPerPage, type Impact, type UserRow } from '../../../lib/api/migrations';
import {
  getPreview,
  preflightKeys,
  previewFilters,
  type PreviewFilter,
} from '../../../lib/api/preflight';
import { useFormat } from '../../../lib/preferences/region';
import { wizardMessages } from '../messages';
import { expiryOf } from './expiry';
import { previewMessages } from './messages';
import { Pager } from '../../../components/data/Pager';
import { useSettled } from '../UserList';
/** Final status, data limit and expiry, as the run writes them. */
function Facts({ row }: { row: UserRow }) {
  const t = useT(previewMessages);
  const fmt = useFormat();
  const expiry = expiryOf(row);
  return (
    <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-kumo-subtle">
      <span className="text-kumo-default">{t(`status_${row.status}`)}</span>
      <span className="tabular">{row.dataLimit > 0 ? fmt.bytes(row.dataLimit) : t('noLimit')}</span>
      <span className="tabular">
        {expiry.kind === 'date'
          ? t('expires', { date: fmt.date(expiry.date) })
          : t(expiry.kind === 'onHold' ? 'onHoldExpiry' : 'noExpiry')}
      </span>
    </span>
  );
}
function Row({ row }: { row: UserRow }) {
  const t = useT(previewMessages);
  const tw = useT(wizardMessages);
  return (
    <li className="flex min-h-12 flex-col gap-1.5 border-t border-kumo-hairline px-3 py-2.5 first:border-0 sm:flex-row sm:items-center sm:gap-4">
      <span className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5 sm:w-2/5">
        <bdi dir="ltr" className="truncate font-medium text-kumo-strong">
          {row.sourceUsername}
        </bdi>
        {row.username !== row.sourceUsername ? (
          <>
            <ArrowRightIcon
              size={12}
              aria-label={t('filter_renamed')}
              className="text-kumo-subtle rtl:-scale-x-100"
            />
            <bdi dir="ltr" className="truncate text-kumo-default">
              {row.username}
            </bdi>
          </>
        ) : null}
      </span>
      {row.include ? <Facts row={row} /> : <Badge variant="outline">{tw('leftOut')}</Badge>}
      {row.include && (row.changesConfig || row.changesLink) ? (
        <span className="flex flex-wrap gap-1.5 sm:ms-auto">
          {row.changesConfig ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
          {row.changesLink ? <Badge variant="outline">{t('linkFlag')}</Badge> : null}
        </span>
      ) : null}
    </li>
  );
}
const countOf: Record<PreviewFilter, keyof Impact> = {
  '': 'users',
  renamed: 'renamed',
  configChanged: 'configChanged',
  skipped: 'skipped',
};
/** Every user as the run will write them: filters, a search on the server, 50 a page (D5). */
export function PreviewList({ jobId, impact }: { jobId: string; impact: Impact | undefined }) {
  const t = useT(previewMessages);
  const tw = useT(wizardMessages);
  const fmt = useFormat();
  const titleId = useId();
  const [filter, setFilter] = useState<PreviewFilter>('');
  const [term, setTerm] = useState('');
  const [offset, setOffset] = useState(0);
  const q = useSettled(term.trim());
  const query = { group: filter, q, offset };
  const users = useQuery({
    queryKey: preflightKeys.preview(jobId, query),
    queryFn: () => getPreview(jobId, query),
    placeholderData: keepPreviousData,
  });
  const rows = users.data?.users ?? [];
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('usersTitle')}
      </h2>
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div role="group" aria-label={t('filtersLabel')} className="flex flex-wrap gap-2">
          {previewFilters.map((f) => (
            <Button
              key={f || 'all'}
              aria-pressed={filter === f}
              variant={filter === f ? 'secondary' : 'ghost'}
              className="pointer-coarse:h-11"
              onClick={() => {
                setFilter(f);
                setOffset(0);
              }}
            >
              {f ? t(`filter_${f}`) : t('filterAll')}
              <span className="tabular opacity-80">{fmt.number(impact?.[countOf[f]] ?? 0)}</span>
            </Button>
          ))}
        </div>
        <SearchField
          label={tw('search')}
          value={term}
          onChange={(v) => {
            setTerm(v);
            setOffset(0);
          }}
          className="lg:max-w-xs"
        />
      </div>
      {users.isPending ? (
        <div role="status" aria-label={t('usersLabel')} className="flex flex-col gap-2">
          <Bone className="h-12 w-full" />
          <Bone className="h-12 w-full" />
        </div>
      ) : users.isError ? (
        <ErrorState
          message={tw('usersFailed')}
          error={users.error}
          onRetry={() => void users.refetch()}
        />
      ) : rows.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">
          {q ? tw('noMatch') : filter ? t(`empty_${filter}`) : t('emptyAll')}
        </p>
      ) : (
        <ul
          aria-label={t('usersLabel')}
          className="flex flex-col rounded-lg ring ring-kumo-hairline"
        >
          {rows.map((row) => (
            <Row key={row.key} row={row} />
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
    </section>
  );
}
