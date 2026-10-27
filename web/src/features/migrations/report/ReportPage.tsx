import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { ArrowRightIcon, PlayIcon, PrinterIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { EmptyState } from '../../../components/feedback/EmptyState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { PageHeader } from '../../../components/layout/PageHeader';
import { Section } from '../../../components/layout/Section';
import { usePageTitle } from '../../../components/layout/use-page-title';
import { MoveLabel } from '../../../components/panels/PanelLabel';
import { panelNames } from '../../../components/panels/PanelLogo';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import { errorOf } from '../../../lib/errors/errors';
import { getReport, type Report } from '../../../lib/api/report';
import { getJobRow, migrationKeys, type JobState } from '../../../lib/api/migrations';
import { useFormat } from '../../../lib/preferences/region';
import { mayRollBack } from '../../live/live';
import { liveMessages } from '../../live/messages';
import { JobStateBadge } from '../JobLabels';
import { listMessages } from '../list-messages';
import { migrationsMessages } from '../messages';
import { reportsMessages } from './messages';
import { ReportAside } from './ReportAside';
import { Stages, UndoCard } from './ReportParts';
import { ReportResult } from './ReportResult';
import { ReportUsers } from './ReportUsers';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

const linkButton = cn(buttonVariants({ variant: 'secondary' }), 'gap-2 pointer-coarse:h-9');

/**
 * The detail page's body: the result, the stages, what cannot be undone and
 * every user in the main column, the admin, Safe Mode, the reload and the
 * details in the aside. On phones the aside follows the result.
 */
function Body({ id, state, report }: { id: string; state: JobState; report: Report }) {
  const t = useT(reportsMessages);
  return (
    <div className="grid max-w-[1400px] lg:grid-cols-[minmax(0,1fr)_22rem] lg:grid-rows-[auto_1fr]">
      <div className="p-4 sm:p-8 lg:pb-0">
        <ReportResult id={id} state={state} report={report} />
      </div>
      <aside
        aria-label={t('facts')}
        className="border-y border-kumo-hairline p-4 sm:p-8 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:border-s lg:border-y-0"
      >
        <ReportAside state={state} report={report} />
      </aside>
      <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8 lg:pt-6">
        {report.stages?.length ? (
          <Section title={t('stages')}>
            <div className="px-4 pt-2 pb-5">
              <Stages stages={report.stages} />
            </div>
          </Section>
        ) : null}
        <UndoCard state={state} report={report} />
        <ReportUsers users={report.users ?? []} state={state} />
      </div>
    </div>
  );
}

/** The move, the old panel's address and the new panel, then when it ran and how long it took. */
function HeaderLine({ report }: { report: Report }) {
  const tm = useT(migrationsMessages);
  const tl = useT(liveMessages);
  const fmt = useFormat();
  const { source: from, destination: to } = report;
  const took = (Date.parse(report.endedAt) - Date.parse(report.startedAt)) / 1000;
  return (
    <span className="flex flex-col gap-1.5">
      <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        {from.kind && to.kind ? (
          <MoveLabel
            from={from.kind}
            to={to.kind}
            size={20}
            label={tm('move', { from: panelNames[from.kind], to: panelNames[to.kind] })}
          />
        ) : null}
        <Machine>{from.origin}</Machine>
        {to.name ? (
          <>
            <ArrowRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
            <bdi>{to.name}</bdi>
          </>
        ) : null}
      </span>
      <span className="text-sm">
        {fmt.dateTime(new Date(report.startedAt))}
        {took >= 0 ? (
          <>
            <span aria-hidden="true" className="px-2">
              ·
            </span>
            {tl('took', { time: took >= 1 ? fmt.duration(took) : tl('underSecond') })}
          </>
        ) : null}
      </span>
    </span>
  );
}

/** One finished job's report (4.7). */
export function ReportPage() {
  const { id = '' } = useParams();
  const t = useT(reportsMessages);
  const tl = useT(listMessages);
  const report = useQuery({ queryKey: migrationKeys.report(id), queryFn: () => getReport(id) });
  const row = useQuery({ queryKey: migrationKeys.row(id), queryFn: () => getJobRow(id) });
  const load = useFirstLoad(report.data);
  const r = load.data;
  const name = r?.source.admin || r?.source.origin || '';
  usePageTitle(r ? t('reportTitle', { name }) : t('printTitle'));
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
      <a
        href={`migrations/${enc}/report/print`}
        target="_blank"
        rel="noopener"
        className={linkButton}
      >
        <PrinterIcon size={16} aria-hidden="true" />
        {t('print')}
      </a>
    </>
  ) : null;
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/migrations"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
          >
            {tl('title')}
          </Link>
        }
        title={
          r ? (
            <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <bdi className="min-w-0 break-all">{t('reportTitle', { name })}</bdi>
              <JobStateBadge state={state} />
            </span>
          ) : (
            t('printTitle')
          )
        }
        description={r ? <HeaderLine report={r} /> : null}
        actions={actions}
      />
      {r && !report.isError ? (
        <Reveal on={load.reveal}>
          <Body id={id} state={state} report={r} />
        </Reveal>
      ) : (
        <div className="flex max-w-[1400px] flex-col gap-6 p-4 sm:p-8">
          {missing ? (
            <EmptyState icon={<PrinterIcon size={32} aria-hidden="true" />} title={t('noReport')} />
          ) : report.isError ? (
            <ErrorState
              message={t('reportFailed')}
              error={report.error}
              onRetry={() => void report.refetch()}
            />
          ) : (
            <div role="status" aria-label={t('printLoading')} className="flex flex-col gap-4">
              <Bone className="h-40 w-full" />
              <Bone className="h-64 w-full" />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
