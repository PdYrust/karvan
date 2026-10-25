import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { DotsThreeIcon, DownloadSimpleIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { DataTable, type Column } from '../../components/data/DataTable';
import { useT } from '../../i18n/locale';
import { jobOutcome, originHost, type JobRow } from '../../lib/api/migrations';
import { reportFiles } from '../../lib/api/report';
import { useErrorText } from '../../lib/errors/errors';
import { liveMessages, type LiveKey } from '../live/messages';
import { JobDestination, JobName, JobUsers, JobWhen } from '../migrations/JobCells';
import { JobStateLabel } from '../migrations/JobLabels';
import { reportPath } from '../migrations/job-place';
import { listMessages } from '../migrations/list-messages';
import { reportsMessages } from './messages';

type File = 'json' | 'csv' | 'links';
const fileWords: Readonly<Record<File, LiveKey>> = {
  json: 'fileJson',
  csv: 'fileCsv',
  links: 'fileLinks',
};

/** The "..." menu of a report row: its files. New links only when users moved and stayed. */
function FilesMenu({ job }: { job: JobRow }) {
  const t = useT(reportsMessages);
  const tl = useT(liveMessages);
  const errorText = useErrorText();
  const toasts = useKumoToastManager();
  const file = useMutation({
    mutationFn: (f: File) => reportFiles[f](job.id),
    onError: (err) =>
      toasts.add({ title: tl('fileFailed'), description: errorText(err), variant: 'error' }),
  });
  const moved = (jobOutcome(job)?.moved ?? 0) > 0 && job.state === 'done';
  const files: File[] = moved ? ['json', 'csv', 'links'] : ['json', 'csv'];
  const name = t('filesFor', { name: job.source.admin || originHost(job.source.origin) });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={name}
            title={name}
            loading={file.isPending}
            icon={<DotsThreeIcon size={20} weight="bold" />}
            className="pointer-coarse:size-11"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-52">
        {files.map((f) => (
          <DropdownMenu.Item key={f} className="gap-2" onClick={() => file.mutate(f)}>
            <DownloadSimpleIcon size={16} aria-hidden="true" />
            {tl(fileWords[f])}
          </DropdownMenu.Item>
        ))}
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

function ReportCard({ job, now }: { job: JobRow; now: Date }) {
  return (
    <article className="flex flex-col gap-2 rounded-lg bg-kumo-base ps-4 pe-2 pt-3 pb-2 ring ring-kumo-hairline">
      <div className="flex items-start gap-2">
        <span className="min-w-0 flex-1">
          <JobName job={job} to={reportPath(job.id)} />
        </span>
        <FilesMenu job={job} />
      </div>
      <div className="text-sm">
        <JobDestination job={job} />
      </div>
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1 border-t border-kumo-hairline pt-2 pe-2 text-sm">
        <JobStateLabel state={job.state} />
        <span className="flex flex-col items-end text-kumo-subtle">
          <JobUsers job={job} />
          <JobWhen job={job} now={now} />
        </span>
      </div>
    </article>
  );
}

/** Finished jobs: the move, the new panel, the result, the users, when it ended, and its files. */
export function ReportsTable({ jobs, now }: { jobs: readonly JobRow[]; now: Date }) {
  const t = useT(reportsMessages);
  const tl = useT(listMessages);
  const columns: Column<JobRow>[] = [
    {
      key: 'job',
      header: t('colReport'),
      cell: (j) => <JobName job={j} to={reportPath(j.id)} />,
      className: 'max-w-80',
    },
    {
      key: 'dest',
      header: tl('colDestination'),
      cell: (j) => <JobDestination job={j} />,
      className: 'max-w-60',
    },
    { key: 'result', header: t('colResult'), cell: (j) => <JobStateLabel state={j.state} /> },
    { key: 'users', header: tl('colUsers'), cell: (j) => <JobUsers job={j} /> },
    { key: 'when', header: t('colEnded'), cell: (j) => <JobWhen job={j} now={now} /> },
    {
      key: 'files',
      header: t('colFiles'),
      hideHeader: true,
      cell: (j) => <FilesMenu job={j} />,
      className: 'w-14 py-1.5 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('table')}
      rows={jobs}
      rowKey={(j) => j.id}
      columns={columns}
      card={(j) => <ReportCard job={j} now={now} />}
    />
  );
}
