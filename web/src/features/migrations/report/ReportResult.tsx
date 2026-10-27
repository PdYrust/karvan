import { cn } from '@cloudflare/kumo/utils';
import { useId } from 'react';
import { StatusIcon } from '../../../components/feedback/StatusBadge';
import { useT } from '../../../i18n/locale';
import type { Counts, JobState } from '../../../lib/api/migrations';
import type { Report } from '../../../lib/api/report';
import { useFormat } from '../../../lib/preferences/region';
import { Files } from '../../live/Files';
import { resultHeads, useMeaning } from '../../live/meaning';
import { liveMessages } from '../../live/messages';
import { jobStates } from '../job-state';

/** The five figures a report always shows, the main one first. */
const figures = [
  'moved',
  'skipped',
  'renamed',
  'configChanged',
  'undone',
] as const satisfies readonly (keyof Counts)[];

/** Each figure on its own tile; a zero stays quiet so the numbers that happened stand out. */
function Figures({ counts }: { counts: Counts }) {
  const t = useT(liveMessages);
  const fmt = useFormat();
  return (
    <dl
      aria-label={t('counts')}
      className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fit,minmax(7.5rem,1fr))]"
    >
      {figures.map((k, i) => (
        <div
          key={k}
          className={cn(
            'flex flex-col gap-1 rounded-lg px-4 py-3 ring ring-kumo-hairline',
            i === 0 && 'col-span-2 sm:col-span-1',
          )}
        >
          <dt className="text-sm text-kumo-subtle">{t(`count_${k}`)}</dt>
          {/* Not through cn: tailwind-merge reads text-title as a color and would drop it. */}
          <dd
            className={`text-title font-semibold tabular ${counts[k] > 0 ? 'text-kumo-strong' : 'text-kumo-subtle'}`}
          >
            {fmt.number(counts[k])}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/** The outcome in words with its icon, what it means, the figures, then the files. */
export function ReportResult({
  id,
  state,
  report,
}: {
  id: string;
  state: JobState;
  report: Report;
}) {
  const t = useT(liveMessages);
  const titleId = useId();
  const meaning = useMeaning(state, report);
  const moved = report.counts.moved - report.counts.undone;
  const password =
    report.admin.password === 'generated' && (state === 'done' || state === 'failed')
      ? report.admin.username
      : null;
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-5 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-6"
    >
      <div className="flex items-start gap-3">
        <span className="grid size-10 shrink-0 place-items-center rounded-full bg-kumo-tint">
          <StatusIcon tone={jobStates[state].tone} size={22} />
        </span>
        <div className="flex min-w-0 flex-col gap-1 pt-0.5">
          <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
            {t(resultHeads[state])}
          </h2>
          {meaning.map((m) => (
            <p key={m} className="max-w-[75ch] text-kumo-default">
              {m}
            </p>
          ))}
        </div>
      </div>
      <Figures counts={report.counts} />
      <div className="border-t border-kumo-hairline pt-4">
        <Files jobId={id} links={state === 'done' && moved > 0} password={password} />
      </div>
    </section>
  );
}
