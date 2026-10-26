import { Button } from '@cloudflare/kumo/components/button';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { TypeToConfirm } from '../../../components/dialogs/TypeToConfirm';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { useT } from '../../../i18n/locale';
import { getGroups, getSummary, migrationKeys } from '../../../lib/api/migrations';
import { getReloadOptions, preflightKeys, type ReloadMethod } from '../../../lib/api/preflight';
import { errorOf, useErrorText } from '../../../lib/errors/errors';
import { usePanels } from '../../panels/use-panels';
import { useSafeMode } from '../../settings/use-safe-mode';
import { wizardMessages } from '../messages';
import { stepPath } from '../steps';
import { useWizard } from '../wizard-context';
import { CheckAgain } from './CheckAgain';
import { startMessages } from './messages';
import { ReloadChoice } from './ReloadChoice';
import { SafeModeCard } from './SafeModeCard';
import { useCheck } from './use-check';
import { startGate, useStart } from './use-start';
import { adminLine, UserEffects } from './UserEffects';
import { ImpactBar } from '../ImpactBar';
import { useFirstLoad } from '../../../lib/motion/use-first-load';
/**
 * Step 6: check again, the reload method, Safe Mode, what changes for
 * users, then Start (4.5, B3). Nothing is written before Start.
 */
export function StartStep() {
  const t = useT(startMessages);
  const tw = useT(wizardMessages);
  const errorText = useErrorText();
  const navigate = useNavigate();
  const { jobId, plan, password } = useWizard();
  const summary = useQuery({
    queryKey: migrationKeys.summary(jobId),
    queryFn: () => getSummary(jobId),
  });
  const groups = useQuery({
    queryKey: migrationKeys.groups(jobId),
    queryFn: () => getGroups(jobId),
  });
  const reloads = useQuery({
    queryKey: preflightKeys.reload(jobId),
    queryFn: () => getReloadOptions(jobId),
  });
  const safe = useSafeMode();
  const panels = usePanels();
  const check = useCheck(jobId);
  const start = useStart(jobId, check.reset);
  const [method, setMethod] = useState<ReloadMethod>();
  const load = useFirstLoad(summary.data && groups.data ? true : undefined);
  const failed = summary.error ?? groups.error;
  if (failed) {
    if (errorOf(failed).code === 'not_checked') {
      return <InlineAlert tone="warning">{tw('notChecked')}</InlineAlert>;
    }
    const retry = () => {
      void summary.refetch();
      void groups.refetch();
    };
    return <ErrorState message={t('failed')} error={failed} onRetry={retry} />;
  }
  if (!load.data || !summary.data || !groups.data) {
    return (
      <div role="status" aria-label={t('loading')} className="flex flex-col gap-4">
        <Bone className="h-28 w-full" />
        <Bone className="h-32 w-full" />
        <Bone className="h-48 w-full" />
      </div>
    );
  }
  const s = summary.data;
  // The server judges the password at Start (D11); Compare already checks it as it is typed.
  const valid = password.value !== '';
  const admin = adminLine(s, password, valid);
  const changes = check.changes?.data;
  const gate = startGate({
    blockers: groups.data.blockers,
    checked: changes !== undefined,
    changed: changes?.changed ?? false,
    accepted: check.accepted,
    reload: method,
    adminMissing: admin === 'adminMissing',
  });
  const ready = groups.data.blockers === 0 && gate === null && changes !== undefined;
  const control = panels.data?.find((p) => p.id === s.panelId)?.control;
  const go = () => {
    if (!changes || !method) return;
    start.run.mutate({ planVersion: changes.planVersion, reload: method, password });
  };
  const runError = start.run.error;
  return (
    <div className="flex flex-col gap-6">
      <ImpactBar impact={plan.impact} blockers={groups.data.blockers} saving={plan.saving} />
      <CheckAgain
        check={{
          ...check,
          run: () => {
            start.run.reset();
            check.run();
          },
        }}
      />
      <div className="rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-5">
        <ReloadChoice options={reloads} value={method} onChange={setMethod} />
      </div>
      <SafeModeCard safe={safe} control={control} />
      <UserEffects jobId={jobId} impact={plan.impact} summary={s} admin={admin} />
      {runError ? (
        <InlineAlert>
          {start.stale ? t('staleRun') : `${t('startFailed')} ${errorText(runError)}`}
        </InlineAlert>
      ) : null}
      <div className="flex flex-col gap-3 border-t border-kumo-hairline pt-5 sm:flex-row sm:items-center sm:justify-end">
        {gate ? (
          <p role="status" className="text-sm text-kumo-subtle sm:me-auto">
            {t(gate)}
          </p>
        ) : null}
        <div className="flex gap-2 sm:justify-end">
          <Button
            className="flex-1 justify-center sm:flex-none pointer-coarse:h-11"
            onClick={() => void navigate(stepPath('preview', jobId))}
          >
            {tw('back')}
          </Button>
          <Button
            variant="primary"
            disabled={!ready || !safe.data}
            loading={start.run.isPending && !start.confirming}
            className="flex-1 justify-center sm:flex-none pointer-coarse:h-11"
            onClick={() => (safe.data?.enabled ? go() : start.setConfirming(true))}
          >
            {t('start')}
          </Button>
        </div>
      </div>
      <TypeToConfirm
        open={start.confirming}
        onOpenChange={start.setConfirming}
        title={t('confirmTitle')}
        word={s.newAdmin.username}
        action={t('start')}
        pending={start.run.isPending}
        onConfirm={go}
      >
        {t('confirmBody')}
      </TypeToConfirm>
    </div>
  );
}
