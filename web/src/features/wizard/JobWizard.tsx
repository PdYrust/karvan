import { useQuery } from '@tanstack/react-query';
import { Suspense } from 'react';
import { Outlet, useLocation, useParams } from 'react-router';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { getJob, migrationKeys, originHost, type JobSummary } from '../../lib/api/migrations';
import { migrationsMessages } from '../migrations/messages';
import { wizardMessages } from './messages';
import { stepOf } from './steps';
import { WizardFrame } from './WizardFrame';
import { WizardProvider } from './wizard-context';

/** The move's logos and the old panel's host, once a compare has read the old panel. */
function JobLine({ summary }: { summary: JobSummary | undefined }) {
  const tm = useT(migrationsMessages);
  if (!summary?.sourceKind || !summary.destKind) return null;
  const from = summary.sourceKind;
  const to = summary.destKind;
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      <MoveLabel
        from={from}
        to={to}
        label={tm('move', { from: panelNames[from], to: panelNames[to] })}
      />
      <bdi dir="ltr">{originHost(summary.origin)}</bdi>
    </span>
  );
}

/** The steps after Connect: one job, its header, and the open step. */
export function JobWizard() {
  const { id = '' } = useParams();
  const { pathname } = useLocation();
  const t = useT(wizardMessages);
  const job = useQuery({ queryKey: migrationKeys.job(id), queryFn: () => getJob(id) });
  return (
    <WizardFrame
      step={stepOf(pathname)}
      jobId={id}
      description={<JobLine summary={job.data?.summary} />}
    >
      {job.isError ? (
        <ErrorState message={t('jobFailed')} error={job.error} onRetry={() => void job.refetch()} />
      ) : (
        // The step's own chunk loads under the header and the steps.
        <WizardProvider jobId={id}>
          <Suspense fallback={<Bone className="h-24 w-full max-w-[720px]" />}>
            {/* Keyed by step, so a new step's content fades in. */}
            <div key={pathname} className="k-reveal">
              <Outlet />
            </div>
          </Suspense>
        </WizardProvider>
      )}
    </WizardFrame>
  );
}
