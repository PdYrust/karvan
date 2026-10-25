import { Button } from '@cloudflare/kumo/components/button';
import { ArrowCounterClockwiseIcon } from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { DataTable, type Column } from '../../components/data/DataTable';
import { useT } from '../../i18n/locale';
import type { JobRow } from '../../lib/api/migrations';
import { JobDestination, JobName, JobUsers, JobWhen } from './JobCells';
import { JobStateLabel } from './JobLabels';
import { againPath, loginDropped } from './job-place';
import { listMessages } from './list-messages';

const jobPath = (j: JobRow) => `/migrations/${encodeURIComponent(j.id)}`;

/** The state, and for a draft Karvan signed out of, that it did. */
function JobState({ job }: { job: JobRow }) {
  const t = useT(listMessages);
  return (
    <span className="flex flex-col gap-0.5">
      <JobStateLabel state={job.state} />
      {loginDropped(job) ? (
        <span className="text-sm text-kumo-subtle">{t('loginDropped')}</span>
      ) : null}
    </span>
  );
}

/** A dropped draft cannot read the old panel again by itself: a new one starts from its address. */
function StartAgain({ job }: { job: JobRow }) {
  const t = useT(listMessages);
  const navigate = useNavigate();
  if (!loginDropped(job) || !job.source.origin) return null;
  return (
    <Button
      size="sm"
      icon={<ArrowCounterClockwiseIcon size={16} />}
      className="pointer-coarse:h-11"
      onClick={() => void navigate(againPath(job))}
    >
      {t('startAgain')}
    </Button>
  );
}

/** On a phone a job is a card: the move and admin first, then the panel, the state and the users. */
function JobCard({ job, now }: { job: JobRow; now: Date }) {
  return (
    <article className="flex flex-col gap-2 rounded-lg bg-kumo-base px-4 py-3 ring ring-kumo-hairline">
      <JobName job={job} to={jobPath(job)} />
      <div className="text-sm">
        <JobDestination job={job} />
      </div>
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1 border-t border-kumo-hairline pt-2 text-sm">
        <JobState job={job} />
        <span className="flex flex-col items-end text-kumo-subtle">
          <JobUsers job={job} />
          <JobWhen job={job} now={now} />
        </span>
      </div>
      <StartAgain job={job} />
    </article>
  );
}

/** Jobs newest first: the move, the new panel, the state, the users and when. */
export function MigrationsTable({ jobs, now }: { jobs: readonly JobRow[]; now: Date }) {
  const t = useT(listMessages);
  const columns: Column<JobRow>[] = [
    {
      key: 'job',
      header: t('colMigration'),
      cell: (j) => <JobName job={j} to={jobPath(j)} />,
      className: 'max-w-80',
    },
    {
      key: 'panel',
      header: t('colDestination'),
      cell: (j) => <JobDestination job={j} />,
      className: 'max-w-60',
    },
    { key: 'state', header: t('colState'), cell: (j) => <JobState job={j} /> },
    { key: 'users', header: t('colUsers'), cell: (j) => <JobUsers job={j} /> },
    { key: 'when', header: t('colWhen'), cell: (j) => <JobWhen job={j} now={now} /> },
    {
      key: 'actions',
      header: t('colActions'),
      hideHeader: true,
      cell: (j) => <StartAgain job={j} />,
      className: 'py-1.5 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('table')}
      rows={jobs}
      rowKey={(j) => j.id}
      columns={columns}
      card={(j) => <JobCard job={j} now={now} />}
    />
  );
}
