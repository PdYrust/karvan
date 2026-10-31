import { ArrowRightIcon } from '@phosphor-icons/react';
import { useId, useState } from 'react';
import { CursorPager } from '../../components/data/CursorPager';
import { DataTable, type Column } from '../../components/data/DataTable';
import { SearchBar } from '../../components/data/SearchBar';
import { CardFacts, RowCard } from '../../components/data/RowCard';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import type { BatchUser } from '../../lib/api/batches';
import { useSettled } from '../../lib/lists/use-settled';
import { useFormat } from '../../lib/preferences/region';
import { useNow } from '../overview/use-now';
import { batchesMessages } from './messages';
import { ShowLinkButton, useShowLink } from './ShowLink';
import { useBatchUsers } from './use-batches';

/** The old username, and the new one after an arrow when the run renamed the user. */
function UserName({ user }: { user: BatchUser }) {
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5">
      <bdi dir="ltr" className="truncate font-medium text-kumo-strong">
        {user.sourceUsername}
      </bdi>
      {user.username !== user.sourceUsername ? (
        <>
          <ArrowRightIcon
            size={12}
            aria-hidden="true"
            className="text-kumo-subtle rtl:-scale-x-100"
          />
          <bdi dir="ltr" className="truncate">
            {user.username}
          </bdi>
        </>
      ) : null}
    </span>
  );
}

/** How many old links were verified, and when the last check was. */
function OldLinks({ user, now }: { user: BatchUser; now: Date }) {
  const t = useT(batchesMessages);
  const fmt = useFormat();
  const at = user.oldLinks.lastChecked ? new Date(user.oldLinks.lastChecked) : null;
  return (
    <span className="flex flex-col gap-0.5">
      <span>{t('verified', { count: user.oldLinks.verified })}</span>
      <span className="text-sm text-kumo-subtle">
        {at ? (
          <time dateTime={at.toISOString()} title={fmt.dateTime(at)}>
            {t('lastChecked', { time: fmt.relative(at, now) })}
          </time>
        ) : (
          t('neverChecked')
        )}
      </span>
    </span>
  );
}

function LinkState({ user }: { user: BatchUser }) {
  const t = useT(batchesMessages);
  return user.closed ? (
    <StatusLabel tone="neutral">{t('linkClosed')}</StatusLabel>
  ) : (
    <StatusLabel tone="success">{t('linkOpen')}</StatusLabel>
  );
}

function UsersTable({
  users,
  onShow,
}: {
  users: readonly BatchUser[];
  onShow: (u: BatchUser) => void;
}) {
  const t = useT(batchesMessages);
  const now = useNow();
  const joined = (u: BatchUser) => t(`joined_${u.joined}`);
  const columns: Column<BatchUser>[] = [
    {
      key: 'user',
      header: t('colUser'),
      cell: (u) => <UserName user={u} />,
      className: 'max-w-64',
    },
    {
      key: 'admin',
      header: t('colAdmin'),
      cell: (u) => <bdi dir="ltr">{u.admin}</bdi>,
      className: 'max-w-40',
    },
    { key: 'joined', header: t('colJoined'), cell: joined },
    { key: 'old', header: t('colOldLinks'), cell: (u) => <OldLinks user={u} now={now} /> },
    { key: 'state', header: t('colLinkState'), cell: (u) => <LinkState user={u} /> },
    {
      key: 'link',
      header: t('colNewLink'),
      hideHeader: true,
      cell: (u) => <ShowLinkButton onShow={() => onShow(u)} />,
      className: 'py-1.5 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('usersTable')}
      rows={users}
      rowKey={(u) => u.id}
      columns={columns}
      card={(u) => (
        <RowCard lead={<UserName user={u} />} badge={<LinkState user={u} />}>
          <CardFacts
            facts={[
              { label: t('colAdmin'), value: <bdi dir="ltr">{u.admin}</bdi> },
              { label: t('colJoined'), value: joined(u) },
              { label: t('colOldLinks'), value: <OldLinks user={u} now={now} /> },
            ]}
          />
          <span className="flex justify-end">
            <ShowLinkButton onShow={() => onShow(u)} />
          </span>
        </RowCard>
      )}
    />
  );
}

/** A batch's users: the server's search, 50 a page, and Show new link on each. */
export function BatchUserList({ batchId }: { batchId: number }) {
  const t = useT(batchesMessages);
  const titleId = useId();
  const [term, setTerm] = useState('');
  const search = useSettled(term);
  const list = useBatchUsers(batchId, search);
  const link = useShowLink(batchId);
  const { query } = list;
  let body;
  if (!query.data) {
    body = query.isError ? (
      <ErrorState
        message={t('usersFailed')}
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    ) : (
      <Bone className="h-56 w-full rounded-lg" />
    );
  } else if (query.data.users.length === 0) {
    body = <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('noUsers')}</p>;
  } else {
    body = <UsersTable users={query.data.users} onShow={link.show} />;
  }
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('usersTitle')}
      </h2>
      <SearchBar search={{ label: t('usersSearch'), value: term, onChange: setTerm }} />
      {body}
      <CursorPager
        offset={list.offset}
        shown={query.data?.users.length ?? 0}
        hasNext={list.hasNext}
        onPrevious={list.previous}
        onNext={list.next}
      />
      {link.dialog}
    </section>
  );
}
