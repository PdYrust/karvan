import { cn } from '@cloudflare/kumo/utils';
import { ArrowRightIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { cardLink } from '../../components/data/RowCard';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { originHost, type JobRow } from '../../lib/api/migrations';
import { useFormat } from '../../lib/preferences/region';
import { JobMove, JobOutcomeText } from './JobLabels';
import { listMessages } from './list-messages';

/** The move's logos, then the admin as the link to the job and the old panel's host under it. */
export function JobName({ job, to }: { job: Pick<JobRow, 'source' | 'panel'>; to: string }) {
  const host = originHost(job.source.origin);
  return (
    <span className="flex min-w-0 items-center gap-3">
      <JobMove job={job} />
      <span className="flex min-w-0 flex-col">
        <Link
          to={to}
          className="truncate font-medium text-kumo-strong underline decoration-kumo-line underline-offset-4 hover:decoration-current"
        >
          <bdi dir="ltr">{job.source.admin || host}</bdi>
        </Link>
        {job.source.admin ? (
          <bdi dir="ltr" className="truncate text-start text-sm text-kumo-subtle">
            {host}
          </bdi>
        ) : null}
      </span>
    </span>
  );
}

/**
 * A job on a phone card: the admin as the card's link, then the old panel's
 * host and, after an arrow, the new panel (left out where the card names it).
 */
export function JobCardName({
  job,
  to,
  destination = true,
}: {
  job: Pick<JobRow, 'source' | 'panel'>;
  to: string;
  destination?: boolean;
}) {
  const t = useT(listMessages);
  const host = originHost(job.source.origin);
  const dest = destination && job.panel.name;
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Link to={to} className={cn('truncate font-medium text-kumo-strong', cardLink)}>
        <bdi dir="ltr">{job.source.admin || host}</bdi>
      </Link>
      {job.source.admin || dest ? (
        <span className="flex min-w-0 items-center gap-1.5 text-sm text-kumo-subtle">
          {job.source.admin ? (
            <bdi dir="ltr" className="max-w-3/5 truncate">
              {host}
            </bdi>
          ) : null}
          {job.source.admin && dest ? (
            <ArrowRightIcon size={12} aria-hidden="true" className="shrink-0 rtl:-scale-x-100" />
          ) : null}
          {dest ? (
            <span className="min-w-0 truncate">
              <span className="sr-only">{`${t('colDestination')}: `}</span>
              {job.panel.name}
            </span>
          ) : null}
        </span>
      ) : null}
    </div>
  );
}

/** The destination panel with its logo. */
export function JobDestination({ job }: { job: JobRow }) {
  return (
    <PanelLabel kind={job.panel.kind || undefined} name={job.panel.name} className="min-w-0" />
  );
}

/** How many users moved and stayed, that a rollback took them all back, or that it has not run. */
export function JobUsers({ job }: { job: JobRow }) {
  const t = useT(listMessages);
  if (!job.counts) return <span className="text-kumo-subtle">{t('notRun')}</span>;
  if (job.state === 'rolled_back') return <span className="text-kumo-subtle">{t('noneLeft')}</span>;
  return <JobOutcomeText job={job} />;
}

/** When the job last changed: relative to the page's clock, with the full time on hover. */
export function JobWhen({ job, now }: { job: JobRow; now: Date }) {
  const fmt = useFormat();
  const at = new Date(job.endedAt ?? job.updatedAt);
  return (
    <time dateTime={at.toISOString()} title={fmt.dateTime(at)} className="whitespace-nowrap">
      {fmt.relative(at, now)}
    </time>
  );
}
