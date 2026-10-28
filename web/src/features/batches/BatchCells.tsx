import { cn } from '@cloudflare/kumo/utils';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { cardLink } from '../../components/data/RowCard';
import { StatusBadge, StatusLabel } from '../../components/feedback/StatusBadge';
import { MoveLabel, PanelLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import type { Batch } from '../../lib/api/batches';
import { originHost } from '../../lib/api/migrations';
import { useFormat } from '../../lib/preferences/region';
import { migrationsMessages } from '../migrations/messages';
import { batchLook, batchPath } from './batch-view';
import { batchesMessages } from './messages';

const link =
  'truncate font-medium text-kumo-strong underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** The old panel's logo, an arrow and the new panel's logo. */
export function BatchMove({
  batch,
  size,
}: {
  batch: Pick<Batch, 'source' | 'panel'>;
  size?: number;
}) {
  const t = useT(migrationsMessages);
  const from = batch.source.kind;
  const to = batch.panel.kind;
  if (!to) return null;
  return (
    <MoveLabel
      from={from}
      to={to}
      size={size}
      label={t('move', { from: panelNames[from], to: panelNames[to] })}
    />
  );
}

/** The move, then the name as the link and the old panel's host under it. */
export function BatchName({ batch }: { batch: Batch }) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <BatchMove batch={batch} />
      <span className="flex min-w-0 flex-col">
        <Link to={batchPath(batch.id)} className={link}>
          <bdi>{batch.name}</bdi>
        </Link>
        <bdi dir="ltr" className="truncate text-start text-sm text-kumo-subtle">
          {originHost(batch.source.origin)}
        </bdi>
      </span>
    </span>
  );
}

/** On a phone card: the name as the card's link, then the host and the new panel. */
export function BatchCardName({ batch }: { batch: Batch }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Link
        to={batchPath(batch.id)}
        className={cn('truncate font-medium text-kumo-strong', cardLink)}
      >
        <bdi>{batch.name}</bdi>
      </Link>
      <span className="flex min-w-0 items-center gap-1.5 text-sm text-kumo-subtle">
        <bdi dir="ltr" className="max-w-3/5 truncate">
          {originHost(batch.source.origin)}
        </bdi>
        <ArrowRightIcon size={12} aria-hidden="true" className="shrink-0 rtl:-scale-x-100" />
        <span className="min-w-0 truncate">{batch.panel.name}</span>
      </span>
    </div>
  );
}

export function BatchDestination({ batch }: { batch: Batch }) {
  return (
    <PanelLabel kind={batch.panel.kind || undefined} name={batch.panel.name} className="min-w-0" />
  );
}

/** An admin's old name, and its new one after an arrow when it changed. */
export function AdminName({ admin }: { admin: Batch['admins'][number] }) {
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-1.5">
      <bdi dir="ltr" className="truncate">
        {admin.source}
      </bdi>
      {admin.dest && admin.dest !== admin.source ? (
        <>
          <ArrowRightIcon
            size={12}
            aria-hidden="true"
            className="text-kumo-subtle rtl:-scale-x-100"
          />
          <bdi dir="ltr" className="truncate">
            {admin.dest}
          </bdi>
        </>
      ) : null}
    </span>
  );
}

export function BatchAdmins({ batch }: { batch: Batch }) {
  return (
    <span className="flex flex-col gap-0.5">
      {batch.admins.map((a) => (
        <AdminName key={a.source} admin={a} />
      ))}
    </span>
  );
}

/** The users, and how many have verified old links under them. */
export function BatchUserCount({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  return (
    <span className="flex flex-col gap-0.5 whitespace-nowrap">
      <span>{t('users', { count: batch.counts.users })}</span>
      <span className="text-sm text-kumo-subtle">
        {t('verified', { count: batch.counts.verified })}
      </span>
    </span>
  );
}

/** The state as an icon and a word, and why when inactive. */
export function BatchState({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  const look = batchLook(batch);
  return (
    <span className="flex flex-col gap-0.5">
      <StatusLabel tone={look.tone}>{t(look.word)}</StatusLabel>
      {look.reason ? <span className="text-sm text-kumo-subtle">{t(look.reason)}</span> : null}
    </span>
  );
}

export function BatchStateBadge({ batch }: { batch: Pick<Batch, 'state' | 'inactiveReason'> }) {
  const t = useT(batchesMessages);
  const look = batchLook(batch);
  return <StatusBadge tone={look.tone}>{t(look.word)}</StatusBadge>;
}

/** The rules that serve the batch, or a quiet "None". */
export function BatchRules({ batch }: { batch: Batch }) {
  const t = useT(batchesMessages);
  if (batch.rules.length === 0) return <span className="text-kumo-subtle">{t('noRules')}</span>;
  return (
    <span className="flex flex-col gap-0.5">
      {batch.rules.map((r) => (
        <bdi key={r.id} className="truncate">
          {r.name}
        </bdi>
      ))}
    </span>
  );
}

/** When the batch was made: relative to the page's clock, with the full time on hover. */
export function BatchMade({ batch, now }: { batch: Batch; now: Date }) {
  const fmt = useFormat();
  const at = new Date(batch.createdAt);
  return (
    <time dateTime={at.toISOString()} title={fmt.dateTime(at)} className="whitespace-nowrap">
      {fmt.relative(at, now)}
    </time>
  );
}
