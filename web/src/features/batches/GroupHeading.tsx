import { Button } from '@cloudflare/kumo/components/button';
import { LinkIcon } from '@phosphor-icons/react';
import { Link, useNavigate } from 'react-router';
import { useT } from '../../i18n/locale';
import { groupPath, type Batch, type BatchGroup } from '../../lib/api/batches';
import { newBatchRulePath } from '../../lib/api/rules';
import { useFormat } from '../../lib/preferences/region';
import { BatchesTable } from './BatchesTable';
import { BatchMove } from './BatchCells';
import { batchesMessages } from './messages';

const link =
  'font-medium text-kumo-strong underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** The rules that serve an old panel's users, or a way to set one up. */
function GroupRules({ group }: { group: BatchGroup }) {
  const t = useT(batchesMessages);
  const navigate = useNavigate();
  if (group.rules.length === 0) {
    return (
      <Button
        icon={<LinkIcon size={16} />}
        className="pointer-coarse:h-9"
        onClick={() => void navigate(newBatchRulePath(group.latestId))}
      >
        {t('setUp')}
      </Button>
    );
  }
  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-sm">
      <span className="text-kumo-subtle">{t('servedBy')}</span>
      <span className="flex min-w-0 flex-wrap gap-x-3 gap-y-1">
        {group.rules.map((r) => (
          <Link
            key={r.id}
            to={`/rules/${String(r.id)}`}
            className={`${link} inline-flex items-center pointer-coarse:min-h-9`}
          >
            <bdi>{r.name}</bdi>
          </Link>
        ))}
      </span>
    </div>
  );
}

/**
 * An old panel's heading: the move, its address, the new panel and its totals, then the rules
 * that serve every run of it.
 */
export function GroupHeading({ group, now }: { group: BatchGroup; now: Date }) {
  const t = useT(batchesMessages);
  const fmt = useFormat();
  const last = new Date(group.lastAt);
  return (
    <div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
      <div className="flex min-w-0 items-start gap-3">
        <span className="shrink-0 pt-0.5">
          <BatchMove batch={group} size={20} />
        </span>
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="min-w-0 font-semibold break-all text-kumo-strong">
            <bdi dir="ltr">{group.source.origin}</bdi>
          </h2>
          <p className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
            <span className="min-w-0">
              {t('toPanel', { name: '' })}
              <bdi>{group.panel.name}</bdi>
            </span>
            <span className="tabular">{t('runCount', { count: group.counts.runs })}</span>
            <span className="tabular">{t('users', { count: group.counts.users })}</span>
            <time dateTime={last.toISOString()} title={fmt.dateTime(last)}>
              {t('latestRun', { time: fmt.relative(last, now) })}
            </time>
          </p>
        </div>
      </div>
      <GroupRules group={group} />
    </div>
  );
}

/** An old panel on the Batches list: its heading, its newest runs and a way to see them all. */
export function GroupSection({
  group,
  state,
  now,
}: {
  group: BatchGroup;
  state?: Batch['state'];
  now: Date;
}) {
  const t = useT(batchesMessages);
  const { runs, active, inactive } = group.counts;
  const total = state === 'active' ? active : state === 'inactive' ? inactive : runs;
  return (
    <section aria-label={group.source.origin} className="flex flex-col gap-3">
      <GroupHeading group={group} now={now} />
      <BatchesTable batches={group.runs} now={now} inGroup />
      {total > group.runs.length ? (
        <Link
          to={groupPath(group.id, state)}
          className={`${link} inline-flex items-center self-start text-sm pointer-coarse:min-h-9`}
        >
          {t('showAll', { count: total })}
        </Link>
      ) : null}
    </section>
  );
}
