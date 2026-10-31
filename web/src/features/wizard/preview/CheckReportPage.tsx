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
import { panelNames, type PanelKind } from '../../../components/panels/PanelLogo';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import { getSummary, migrationKeys } from '../../../lib/api/migrations';
import { getCheckReport, preflightKeys, type CheckReport } from '../../../lib/api/preflight';
import { useFormat } from '../../../lib/preferences/region';
import { migrationsMessages } from '../../migrations/messages';
import { stepPath } from '../steps';
import { previewMessages, type PreviewKey } from './messages';
import { ReportTable } from './ReportTable';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';
const isKind = (k: string): k is PanelKind => k === 'marzban' || k === 'pasarguard';
const totals: readonly [keyof CheckReport['totals'], PreviewKey][] = [
  ['users', 'totalUsers'],
  ['moving', 'totalMoving'],
  ['skipped', 'totalSkipped'],
  ['alreadyMoved', 'totalAlreadyMoved'],
  ['adopted', 'totalAdopted'],
  ['renamed', 'totalRenamed'],
  ['configChanged', 'totalConfig'],
  ['changed', 'totalChanged'],
];
function Head({ report }: { report: CheckReport }) {
  const t = useT(previewMessages);
  const tm = useT(migrationsMessages);
  const fmt = useFormat();
  const { source, destination, admin } = report;
  const name = admin.username || source.admin;
  return (
    <header className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-title font-semibold text-kumo-strong">{t('printTitle')}</h1>
        {isKind(source.kind) && isKind(destination.kind) ? (
          <MoveLabel
            from={source.kind}
            to={destination.kind}
            label={tm('move', { from: panelNames[source.kind], to: panelNames[destination.kind] })}
          />
        ) : null}
      </div>
      <FactList
        facts={[
          { label: t('oldPanel'), value: <Machine>{source.origin}</Machine> },
          {
            label: t('admin'),
            value: admin.merged ? t('adminMerged', { name }) : <bdi dir="ltr">{name}</bdi>,
          },
          { label: t('destination'), value: <bdi dir="ltr">{destination.name}</bdi> },
          { label: t('checkedAt'), value: fmt.dateTime(new Date(report.checkedAt)) },
          { label: t('planVersion'), value: fmt.number(report.planVersion) },
        ]}
      />
      <dl className="grid grid-cols-2 gap-4 rounded-lg p-4 ring ring-kumo-hairline sm:grid-cols-4">
        {totals.map(([k, word]) => (
          <div key={k} className="flex flex-col">
            <dt className="text-sm text-kumo-subtle">{t(word)}</dt>
            <dd className="text-heading font-semibold text-kumo-strong tabular">
              {fmt.number(report.totals[k])}
            </dd>
          </div>
        ))}
      </dl>
      <p>{report.blockers > 0 ? t('blockers', { count: report.blockers }) : t('noBlockers')}</p>
    </header>
  );
}
/**
 * The pre-flight report as a page to print: outside the shell,
 * one column, a bar that does not print.
 */
export function CheckReportPage() {
  const { id = '' } = useParams();
  const t = useT(previewMessages);
  usePageTitle(t('printTitle'));
  const report = useQuery({
    queryKey: preflightKeys.checkReport(id),
    queryFn: () => getCheckReport(id),
  });
  // Target names; the report holds their ids.
  const load = useFirstLoad(report.data);
  const summary = useQuery({
    queryKey: migrationKeys.summary(id),
    queryFn: () => getSummary(id),
  });
  const names = new Map((summary.data?.targets ?? []).map((g) => [g.id, g.name]));
  return (
    <div className="min-h-dvh bg-kumo-base text-kumo-default">
      <div className="sticky top-0 flex items-center justify-end gap-2 border-b border-kumo-hairline bg-kumo-base px-4 py-3 print:hidden">
        <Link
          to={stepPath('preview', id)}
          className={cn(buttonVariants({ variant: 'ghost' }), 'pointer-coarse:h-9')}
        >
          {t('backToPreview')}
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
            message={t('printFailed')}
            error={report.error}
            onRetry={() => void report.refetch()}
          />
        ) : load.data ? (
          <Reveal on={load.reveal}>
            <Head report={load.data} />
            <ReportTable users={load.data.users} names={names} />
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
