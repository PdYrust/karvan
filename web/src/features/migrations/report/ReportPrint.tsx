import { Button, buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { PrinterIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { FactList } from '../../../components/layout/FactList';
import { usePageTitle } from '../../../components/layout/use-page-title';
import { MoveLabel } from '../../../components/panels/PanelLabel';
import { panelNames } from '../../../components/panels/PanelLogo';
import { useT } from '../../../i18n/locale';
import { getJobRow, migrationKeys } from '../../../lib/api/migrations';
import { getReport, type Report } from '../../../lib/api/report';
import { useFormat } from '../../../lib/preferences/region';
import { shownCounts } from '../../live/live';
import { liveMessages } from '../../live/messages';
import { reportPath } from '../job-place';
import { jobStates } from '../job-state';
import { migrationsMessages } from '../messages';
import { reportsMessages } from './messages';
import { Stages, UndoCard, useReportFacts } from './ReportParts';
import { userOutcome } from './report-view';
import { UserName } from './ReportUsers';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

const cell = 'border-b border-kumo-hairline px-2 py-1.5 text-start align-top';

function Printed({ report, state }: { report: Report; state: Report['state'] }) {
  const t = useT(reportsMessages);
  const tl = useT(liveMessages);
  const tm = useT(migrationsMessages);
  const fmt = useFormat();
  const facts = useReportFacts(report);
  const { source: s, destination: d } = report;
  return (
    <>
      <header className="flex flex-col gap-2">
        <h1 className="text-title font-semibold text-kumo-strong">{t('printTitle')}</h1>
        <p className="flex flex-wrap items-center gap-3">
          {s.kind && d.kind ? (
            <MoveLabel
              from={s.kind}
              to={d.kind}
              label={tm('move', { from: panelNames[s.kind], to: panelNames[d.kind] })}
            />
          ) : null}
          <span className="font-medium">{tm(jobStates[state].word)}</span>
        </p>
      </header>
      <FactList facts={facts} />
      <dl className="grid grid-cols-2 gap-4 rounded-lg p-4 ring ring-kumo-hairline sm:grid-cols-4 print:grid-cols-7">
        {shownCounts(report.counts).map(([k, n]) => (
          <div key={k} className="flex flex-col">
            <dt className="text-sm text-kumo-subtle">{tl(`count_${k}`)}</dt>
            <dd className="text-heading font-semibold text-kumo-strong tabular">{fmt.number(n)}</dd>
          </div>
        ))}
      </dl>
      {report.stages?.length ? (
        <section className="flex flex-col gap-1">
          <h2 className="text-heading font-semibold text-kumo-strong">{t('stages')}</h2>
          <Stages stages={report.stages} />
        </section>
      ) : null}
      <UndoCard state={state} report={report} />
      <table className="w-full text-sm">
        <caption className="pb-2 text-start text-heading font-semibold text-kumo-strong">
          {t('users')}
        </caption>
        <thead>
          <tr>
            <th scope="col" className={cn(cell, 'font-medium text-kumo-subtle')}>
              {t('colUser')}
            </th>
            <th scope="col" className={cn(cell, 'font-medium text-kumo-subtle')}>
              {t('colOutcome')}
            </th>
            <th scope="col" className={cn(cell, 'font-medium text-kumo-subtle')}>
              {t('colChanges')}
            </th>
          </tr>
        </thead>
        <tbody>
          {(report.users ?? []).map((u) => (
            <tr key={u.key} className="break-inside-avoid">
              <td className={cell}>
                <UserName user={u} />
              </td>
              <td className={cell}>{t(userOutcome(u, state))}</td>
              <td className={cell}>
                {[
                  u.changesConfig ? t('configFlag') : '',
                  u.changesLink && u.moved ? t('linkFlag') : '',
                ]
                  .filter(Boolean)
                  .join(', ')}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}

/** A report as a page to print (4.7): outside the shell, one column, a bar that does not print. */
export function ReportPrint() {
  const { id = '' } = useParams();
  const t = useT(reportsMessages);
  usePageTitle(t('printTitle'));
  const report = useQuery({ queryKey: migrationKeys.report(id), queryFn: () => getReport(id) });
  const load = useFirstLoad(report.data);
  const row = useQuery({ queryKey: migrationKeys.row(id), queryFn: () => getJobRow(id) });
  return (
    <div className="min-h-dvh bg-kumo-base text-kumo-default">
      <div className="sticky top-0 flex items-center justify-end gap-2 border-b border-kumo-hairline bg-kumo-base px-4 py-3 print:hidden">
        <Link
          to={reportPath(id)}
          className={cn(buttonVariants({ variant: 'ghost' }), 'pointer-coarse:h-9')}
        >
          {t('back')}
        </Link>
        <Button
          variant="primary"
          icon={<PrinterIcon size={16} />}
          disabled={!load.data}
          className="pointer-coarse:h-9"
          onClick={() => window.print()}
        >
          {t('print')}
        </Button>
      </div>
      <main className="mx-auto flex max-w-[1100px] flex-col gap-6 p-4 sm:p-8 print:max-w-none print:p-0">
        {report.isError ? (
          <ErrorState
            message={t('reportFailed')}
            error={report.error}
            onRetry={() => void report.refetch()}
          />
        ) : load.data ? (
          <Reveal on={load.reveal}>
            <Printed report={load.data} state={row.data?.state ?? load.data.state} />
          </Reveal>
        ) : (
          <div role="status" aria-label={t('printLoading')} className="flex flex-col gap-4">
            <Bone className="h-8 w-64" />
            <Bone className="h-40 w-full" />
            <Bone className="h-64 w-full" />
          </div>
        )}
      </main>
    </div>
  );
}
