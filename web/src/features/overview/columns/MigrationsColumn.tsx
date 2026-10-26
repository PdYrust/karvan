import { ArrowsLeftRightIcon } from '@phosphor-icons/react';
import { useT } from '../../../i18n/locale';
import { originHost, type JobRow } from '../../../lib/api/migrations';
import { useFormat } from '../../../lib/preferences/region';
import { JobMove, JobOutcomeText, JobStateLabel } from '../../migrations/JobLabels';
import { PartFailed } from '../PartFailed';
import { HomeColumn, HomeNote, HomeRow, HomeRows, HomeRowsSkeleton } from './HomeColumn';
import { overviewMessages } from '../messages';
import { useRecentJobs } from '../use-overview';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** The move, its admin and state, then what it moved and when. */
function JobItem({ job, now }: { job: JobRow; now: Date }) {
  const fmt = useFormat();
  const at = new Date(job.endedAt ?? job.updatedAt);
  return (
    <HomeRow to={`/migrations/${encodeURIComponent(job.id)}`}>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="flex min-w-0 items-center gap-2">
          <JobMove job={job} />
          <bdi dir="ltr" className="min-w-0 truncate font-medium text-kumo-strong">
            {job.source.admin || originHost(job.source.origin)}
          </bdi>
          <span className="ms-auto shrink-0 text-sm">
            <JobStateLabel state={job.state} />
          </span>
        </span>
        <span className="flex min-w-0 items-center gap-3 text-sm text-kumo-subtle">
          <span className="min-w-0 truncate">
            <JobOutcomeText job={job} />
          </span>
          <time dateTime={at.toISOString()} title={fmt.dateTime(at)} className="ms-auto shrink-0">
            {fmt.relative(at, now)}
          </time>
        </span>
      </span>
    </HomeRow>
  );
}

/** The latest jobs, newest first. */
export function MigrationsColumn({ now }: { now: Date }) {
  const t = useT(overviewMessages);
  const jobs = useRecentJobs();
  const load = useFirstLoad(jobs.data);
  let body;
  if (load.data) {
    body =
      load.data.jobs.length > 0 ? (
        <HomeRows>
          {load.data.jobs.map((j) => (
            <JobItem key={j.id} job={j} now={now} />
          ))}
        </HomeRows>
      ) : (
        <HomeNote icon={<ArrowsLeftRightIcon size={16} aria-hidden="true" />}>
          {t('noMigrations')}
        </HomeNote>
      );
  } else if (jobs.isError) {
    body = <PartFailed message={t('listFailed')} onRetry={() => void jobs.refetch()} />;
  } else {
    body = <HomeRowsSkeleton label={t('migrationsTitle')} />;
  }
  return (
    <HomeColumn title={t('migrationsTitle')} to="/migrations">
      <Reveal on={load.reveal}>{body}</Reveal>
    </HomeColumn>
  );
}
