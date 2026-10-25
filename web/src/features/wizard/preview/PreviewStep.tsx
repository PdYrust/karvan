import { useQuery } from '@tanstack/react-query';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { useT } from '../../../i18n/locale';
import { getGroups, migrationKeys } from '../../../lib/api/migrations';
import { errorOf } from '../../../lib/errors/errors';
import { wizardMessages } from '../messages';
import { useWizard } from '../wizard-context';
import { previewMessages } from './messages';
import { PreviewList } from './PreviewList';
import { ReportCard } from './ReportCard';
import { ImpactBar } from '../ImpactBar';
import { StepActions } from '../StepActions';
/**
 * Step 5: what the plan does in numbers, the pre-flight report (B7), and
 * every user as they will land, paged on the server (4.5, D5).
 */
export function PreviewStep() {
  const t = useT(previewMessages);
  const tw = useT(wizardMessages);
  const { jobId, plan } = useWizard();
  const groups = useQuery({
    queryKey: migrationKeys.groups(jobId),
    queryFn: () => getGroups(jobId),
  });
  const failed = groups.error ?? plan.query.error;
  if (failed) {
    if (errorOf(failed).code === 'not_checked') {
      return <InlineAlert tone="warning">{tw('notChecked')}</InlineAlert>;
    }
    const retry = () => {
      void groups.refetch();
      void plan.query.refetch();
    };
    return <ErrorState message={t('failed')} error={failed} onRetry={retry} />;
  }
  if (!groups.data || !plan.plan) {
    return (
      <div role="status" aria-label={t('loading')} className="flex flex-col gap-4">
        <Bone className="h-28 w-full" />
        <Bone className="h-32 w-full" />
        <Bone className="h-64 w-full" />
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-6">
      <ImpactBar impact={plan.impact} blockers={groups.data.blockers} saving={plan.saving} />
      <ReportCard jobId={jobId} />
      <PreviewList jobId={jobId} impact={plan.impact} />
      <StepActions jobId={jobId} back="changes" next="start" />
    </div>
  );
}
