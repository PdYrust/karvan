import { useQuery } from '@tanstack/react-query';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { useT } from '../../../i18n/locale';
import { getGroups, getSummary, migrationKeys } from '../../../lib/api/migrations';
import { errorOf, useErrorText } from '../../../lib/errors/errors';
import { ImpactBar } from '../ImpactBar';
import { wizardMessages } from '../messages';
import { StepActions } from '../StepActions';
import { useWizard } from '../wizard-context';
import { AddChange } from './AddChange';
import { ChangeList } from './ChangeList';
import { changesMessages } from './messages';
import { Templates } from './Templates';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

function Loading() {
  const t = useT(changesMessages);
  return (
    <div role="status" aria-label={t('loading')} className="flex flex-col gap-4">
      <Bone className="h-28 w-full" />
      <Bone className="h-40 w-full" />
      <Bone className="h-64 w-full" />
    </div>
  );
}

/**
 * Step 4: bulk changes on all users, a category, a filter or picked users,
 * each with how many users it reaches; then templates (4.5, D6).
 */
export function ChangesStep() {
  const t = useT(changesMessages);
  const tw = useT(wizardMessages);
  const errorText = useErrorText();
  const { jobId, plan } = useWizard();
  const summary = useQuery({
    queryKey: migrationKeys.summary(jobId),
    queryFn: () => getSummary(jobId),
  });
  const groups = useQuery({
    queryKey: migrationKeys.groups(jobId),
    queryFn: () => getGroups(jobId),
  });
  const load = useFirstLoad(summary.data && groups.data && plan.plan ? true : undefined);
  const failed = summary.error ?? groups.error ?? plan.query.error;
  if (failed) {
    if (errorOf(failed).code === 'not_checked') {
      return <InlineAlert tone="warning">{tw('notChecked')}</InlineAlert>;
    }
    const retry = () => {
      void summary.refetch();
      void groups.refetch();
      void plan.query.refetch();
    };
    return <ErrorState message={t('failed')} error={failed} onRetry={retry} />;
  }
  if (!load.data || !summary.data || !groups.data || !plan.plan) return <Loading />;
  const counts = groups.data.categories;
  return (
    <div className="flex flex-col gap-6">
      <ImpactBar impact={plan.impact} blockers={groups.data.blockers} saving={plan.saving} />
      {plan.failure ? (
        <InlineAlert>{`${tw('saveFailed')} ${errorText(plan.failure)}`}</InlineAlert>
      ) : null}
      <ChangeList summary={summary.data} counts={counts} />
      <AddChange summary={summary.data} counts={counts} />
      <Templates />
      <StepActions jobId={jobId} back="compare" next="preview" />
    </div>
  );
}
