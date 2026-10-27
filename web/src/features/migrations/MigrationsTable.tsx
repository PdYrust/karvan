import { Button } from '@cloudflare/kumo/components/button';
import { ArrowCounterClockwiseIcon } from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { DataTable, type Column } from '../../components/data/DataTable';
import { RowCard } from '../../components/data/RowCard';
import { useT } from '../../i18n/locale';
import type { JobRow } from '../../lib/api/migrations';
import { JobCardName, JobDestination, JobName, JobUsers, JobWhen } from './JobCells';
import { JobMenu, type DraftJob } from './DeleteDraft';
import { JobMove, JobStateBadge, JobStateLabel } from './JobLabels';
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
      className="pointer-coarse:h-9"
      onClick={() => void navigate(againPath(job))}
    >
      {t('startAgain')}
    </Button>
  );
}

type OnDelete = (job: DraftJob) => void;

/** On a phone a job is a card: the move and its state, the admin, then the users and when. */
function JobCard({ job, now, onDelete }: { job: JobRow; now: Date; onDelete: OnDelete }) {
  const t = useT(listMessages);
  const dropped = loginDropped(job);
  return (
    <RowCard
      lead={<JobMove job={job} size={20} />}
      badge={<JobStateBadge state={job.state} />}
      menu={<JobMenu job={job} onDelete={onDelete} />}
      footer={
        <>
          <JobUsers job={job} />
          <JobWhen job={job} now={now} />
        </>
      }
    >
      <JobCardName job={job} to={jobPath(job)} />
      {dropped ? (
        <div className="flex flex-col items-start gap-2">
          <p className="text-sm text-kumo-subtle">{t('loginDropped')}</p>
          <span className="relative z-10">
            <StartAgain job={job} />
          </span>
        </div>
      ) : null}
    </RowCard>
  );
}

/** Jobs newest first: the move, the new panel, the state, the users and when; a draft's menu deletes it. */
export function MigrationsTable({
  jobs,
  now,
  onDelete,
}: {
  jobs: readonly JobRow[];
  now: Date;
  onDelete: OnDelete;
}) {
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
      cell: (j) => (
        <span className="flex items-center justify-end gap-2">
          <StartAgain job={j} />
          <JobMenu job={j} onDelete={onDelete} />
        </span>
      ),
      className: 'py-1.5 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('table')}
      rows={jobs}
      rowKey={(j) => j.id}
      columns={columns}
      card={(j) => <JobCard job={j} now={now} onDelete={onDelete} />}
    />
  );
}
