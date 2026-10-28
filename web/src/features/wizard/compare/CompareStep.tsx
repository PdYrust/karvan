import { useQuery } from '@tanstack/react-query';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import { getGroups, getSummary, migrationKeys } from '../../../lib/api/migrations';
import { errorOf, useErrorText } from '../../../lib/errors/errors';
import { wizardMessages } from '../messages';
import { StepActions } from '../StepActions';
import { useWizard } from '../wizard-context';
import { AdminList } from './AdminList';
import { Categories } from './Categories';
import { conflictMessages } from './conflict-messages';
import { ImpactBar } from '../ImpactBar';
import { compareMessages } from './messages';
import { ProblemList } from './ProblemList';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

function Loading() {
  const t = useT(compareMessages);
  return (
    <div role="status" aria-label={t('loading')} className="flex flex-col gap-4">
      <Bone className="h-28 w-full" />
      <Bone className="h-64 w-full" />
      <Bone className="h-40 w-full" />
    </div>
  );
}

/**
 * Step 3: the admins first, each with its role, then the problem groups with
 * their fixes, and the categories. Every choice saves the whole plan and
 * shows what it does (D6).
 */
export function CompareStep() {
  const t = useT(compareMessages);
  const tw = useT(wizardMessages);
  const tc = useT(conflictMessages);
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
  const g = groups.data;
  const kind = summary.data.destKind || 'pasarguard';
  return (
    <div className="flex flex-col gap-6">
      <ImpactBar impact={plan.impact} blockers={g.blockers} saving={plan.saving} />
      {plan.failure ? (
        <InlineAlert>{`${tw('saveFailed')} ${errorText(plan.failure)}`}</InlineAlert>
      ) : null}
      {g.conflicts.some((c) => c.type === 'C29') ? (
        <InlineAlert>{tc('job_C29')}</InlineAlert>
      ) : null}
      {g.conflicts
        .filter((c) => c.type === 'C27')
        .map((c) => (
          <InlineAlert key={c.type} tone="warning">
            <CodeText text={tc('job_C27', { skew: c.detail?.skew ?? '' })} />
          </InlineAlert>
        ))}
      {g.conflicts.some((c) => c.type === 'source_groups_unknown') ? (
        <InlineAlert tone="info">{tc('job_source_groups_unknown')}</InlineAlert>
      ) : null}
      <AdminList summary={summary.data} conflicts={g.conflicts} />
      <ProblemList groups={g.groups} kind={kind} />
      <Categories counts={g.categories} total={summary.data.users} />
      <StepActions jobId={jobId} back="summary" next="changes" />
    </div>
  );
}
