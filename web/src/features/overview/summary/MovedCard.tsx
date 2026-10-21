import { cn } from '@cloudflare/kumo/utils';
import { Bar } from '../../../components/charts/Bar';
import { useT } from '../../../i18n/locale';
import type { JobState } from '../../../lib/api/migrations';
import type { MigrationTotals } from '../../../lib/api/overview';
import { useFormat } from '../../../lib/preferences/region';
import { jobStates } from '../../migrations/job-state';
import { migrationsMessages } from '../../migrations/messages';
import { overviewMessages } from '../messages';
import { PartFailed } from '../PartFailed';
import { useOverview } from '../use-overview';
import { SummaryCard, SummarySkeleton } from './SummaryCard';

/** How runs end. A run cut short by a restart is rare and needs a look, so it shows only when there is one. */
function outcomesOf(totals: MigrationTotals): JobState[] {
  const ends: JobState[] = ['done', 'failed', 'rolled_back'];
  return totals.jobs.interrupted ? [...ends, 'interrupted'] : ends;
}

/** How many users moved and stayed, the share left out, and how the runs ended. */
function Moved({ totals }: { totals: MigrationTotals }) {
  const t = useT(overviewMessages);
  const tm = useT(migrationsMessages);
  const fmt = useFormat();
  const read = totals.moved + totals.skipped;
  const outcomes = outcomesOf(totals);
  return (
    <>
      <p className="text-title font-semibold text-kumo-strong tabular">
        {fmt.number(totals.moved)}
      </p>
      {read > 0 ? (
        // The filled part is the number above; the rest is the users the runs left out.
        <div className="flex flex-col gap-2">
          <Bar share={totals.moved / read} />
          {totals.skipped > 0 ? (
            <p className="text-sm text-kumo-subtle">
              {t('skippedCount', { count: totals.skipped })}
            </p>
          ) : null}
        </div>
      ) : (
        <p className="text-sm text-kumo-subtle">{t('movedNone')}</p>
      )}
      <dl
        className={cn(
          'mt-auto grid gap-3 border-t border-kumo-hairline pt-3',
          outcomes.length > 3 ? 'grid-cols-2' : 'grid-cols-3',
        )}
      >
        {outcomes.map((s) => (
          <div key={s} className="flex min-w-0 flex-col">
            <dt className="truncate text-sm text-kumo-subtle">{tm(jobStates[s].word)}</dt>
            <dd className="text-heading font-semibold text-kumo-strong tabular">
              {fmt.number(totals.jobs[s] ?? 0)}
            </dd>
          </div>
        ))}
      </dl>
    </>
  );
}

export function MovedCard() {
  const t = useT(overviewMessages);
  const q = useOverview();
  let body;
  if (q.data) {
    body = <Moved totals={q.data.migrations} />;
  } else if (q.isError) {
    body = <PartFailed message={t('cardFailed')} onRetry={() => void q.refetch()} />;
  } else {
    body = <SummarySkeleton label={t('movedTitle')} />;
  }
  return <SummaryCard title={t('movedTitle')}>{body}</SummaryCard>;
}
