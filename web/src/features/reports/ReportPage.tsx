import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { PlayIcon, PrinterIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { ErrorState } from '../../components/feedback/ErrorState';
import { EmptyState } from '../../components/feedback/EmptyState';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { Bone } from '../../components/feedback/PageSkeleton';
import { FactList } from '../../components/layout/FactList';
import { PageHeader } from '../../components/layout/PageHeader';
import { Section } from '../../components/layout/Section';
import { usePageTitle } from '../../components/layout/use-page-title';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { errorOf } from '../../lib/errors/errors';
import { getReport, type Report } from '../../lib/api/report';
import { getJobRow, migrationKeys, type JobState } from '../../lib/api/migrations';
import { useFormat } from '../../lib/preferences/region';
import { Files } from '../live/Files';
import { mayRollBack, shownCounts } from '../live/live';
import { liveMessages } from '../live/messages';
import { ReloadLine } from '../live/ResultCard';
import { jobStates } from '../migrations/job-state';
import { migrationsMessages } from '../migrations/messages';
import { reportsMessages } from './messages';
import { Stages, UndoCard, useReportFacts } from './ReportParts';
import { ReportUsers } from './ReportUsers';

const linkButton = cn(buttonVariants({ variant: 'secondary' }), 'gap-2 pointer-coarse:h-11');

/** The state, the counts, the reload and the files. */
function Result({ id, state, report }: { id: string; state: JobState; report: Report }) {
  const t = useT(reportsMessages);
  const tl = useT(liveMessages);
  const tm = useT(migrationsMessages);
  const fmt = useFormat();
  const moved = report.counts.moved - report.counts.undone;
  const password =
    report.admin.password === 'generated' && (state === 'done' || state === 'failed')
      ? report.admin.username
      : null;
  return (
    <Section title={t('result')}>
      <div className="flex flex-col gap-5 px-4 pb-4">
        <StatusLabel tone={jobStates[state].tone}>{tm(jobStates[state].word)}</StatusLabel>
        <dl aria-label={tl('counts')} className="flex flex-wrap gap-x-8 gap-y-3">
          {shownCounts(report.counts).map(([k, n]) => (
            <div key={k} className="flex flex-col">
              <dt className="text-sm text-kumo-subtle">{tl(`count_${k}`)}</dt>
              <dd className="text-title font-semibold text-kumo-strong tabular">{fmt.number(n)}</dd>
            </div>
          ))}
        </dl>
        {state === 'done' ? <ReloadLine report={report} t={tl} /> : null}
        <Files jobId={id} links={state === 'done' && moved > 0} password={password} />
      </div>
    </Section>
  );
}

function Body({ id, state, report }: { id: string; state: JobState; report: Report }) {
  const t = useT(reportsMessages);
  const facts = useReportFacts(report);
  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <div className="flex flex-col gap-6">
          <Result id={id} state={state} report={report} />
          <UndoCard state={state} report={report} />
        </div>
        <div className="flex flex-col gap-6">
          <Section title={t('facts')}>
            <div className="px-4 pb-2">
              <FactList facts={facts} />
            </div>
          </Section>
          {report.stages?.length ? (
            <Section title={t('stages')}>
              <div className="px-4 pb-2">
                <Stages stages={report.stages} />
              </div>
            </Section>
          ) : null}
        </div>
      </div>
      <ReportUsers users={report.users ?? []} state={state} />
    </div>
  );
}

/** One finished job's report (4.7). */
export function ReportPage() {
  const { id = '' } = useParams();
  const t = useT(reportsMessages);
  const tm = useT(migrationsMessages);
  const report = useQuery({ queryKey: migrationKeys.report(id), queryFn: () => getReport(id) });
  const row = useQuery({ queryKey: migrationKeys.row(id), queryFn: () => getJobRow(id) });
  const r = report.data;
  const name = r?.source.admin || r?.source.origin || '';
  usePageTitle(r ? t('reportTitle', { name }) : t('title'));
  // The job's own state wins: a backup restore changes it without a new report.
  const state = row.data?.state ?? r?.state ?? 'done';
  const missing = report.isError && errorOf(report.error).code === 'not_found';
  const enc = encodeURIComponent(id);
  const actions = r ? (
    <>
      {mayRollBack(state, r.counts, r) ? (
        <Link to={`/migrations/${enc}/live`} className={linkButton}>
          <PlayIcon size={16} aria-hidden="true" />
          {t('openLive')}
        </Link>
      ) : null}
      <a href={`reports/${enc}/print`} target="_blank" rel="noopener" className={linkButton}>
        <PrinterIcon size={16} aria-hidden="true" />
        {t('print')}
      </a>
    </>
  ) : null;
  const move =
    r?.source.kind && r.destination.kind ? (
      <span className="inline-flex flex-wrap items-center gap-2">
        <MoveLabel
          from={r.source.kind}
          to={r.destination.kind}
          label={tm('move', {
            from: panelNames[r.source.kind],
            to: panelNames[r.destination.kind],
          })}
        />
        <Machine>{r.source.origin}</Machine>
      </span>
    ) : null;
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/reports"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-11"
          >
            {t('title')}
          </Link>
        }
        title={r ? <bdi>{t('reportTitle', { name })}</bdi> : t('printTitle')}
        description={move}
        actions={actions}
      />
      <div className="flex max-w-[1400px] flex-col gap-6 p-4 sm:p-8">
        {missing ? (
          <EmptyState icon={<PrinterIcon size={32} aria-hidden="true" />} title={t('noReport')} />
        ) : report.isError ? (
          <ErrorState
            message={t('reportFailed')}
            error={report.error}
            onRetry={() => void report.refetch()}
          />
        ) : r ? (
          <Body id={id} state={state} report={r} />
        ) : (
          <div role="status" aria-label={t('printLoading')} className="flex flex-col gap-4">
            <Bone className="h-40 w-full" />
            <Bone className="h-64 w-full" />
          </div>
        )}
      </div>
    </div>
  );
}
