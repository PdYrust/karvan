import { useMutation, useMutationState, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { getPlan, migrationKeys, putPlan, type PlanState } from '../../lib/api/migrations';
import type { Plan, PlanEdit } from '../../lib/plan/plan';
import { PlanConflict, saveEdit } from '../../lib/plan/save';

/**
 * A job's plan and the way to change it. Every edit is a pure function
 * (D6); saves run one after another, and the screen shows the edits still
 * on their way on top of the saved plan, so a choice shows at once. After
 * a save the job's other reads (groups, users, summary) load again.
 */
export function usePlan(jobId: string) {
  const qc = useQueryClient();
  const key = migrationKeys.plan(jobId);
  const load = () => qc.fetchQuery({ queryKey: key, queryFn: () => getPlan(jobId), staleTime: 0 });
  const query = useQuery({ queryKey: key, queryFn: () => getPlan(jobId) });
  // The edit a second conflict stopped, until the operator decides (D7).
  const [conflict, setConflict] = useState<PlanEdit | null>(null);
  const mutationKey = ['migrations', 'plan-edit', jobId];
  const save = useMutation({
    mutationKey,
    scope: { id: `plan-${jobId}` },
    mutationFn: async (edit: PlanEdit) => {
      const current = qc.getQueryData<PlanState>(key) ?? (await load());
      return saveEdit(edit, current, { load, save: (v, p) => putPlan(jobId, v, p) });
    },
    onSuccess: ({ plan, saved }) => {
      qc.setQueryData<PlanState>(key, {
        planVersion: saved.planVersion,
        plan,
        impact: saved.impact,
      });
      void qc.invalidateQueries({
        queryKey: migrationKeys.job(jobId),
        predicate: (q) => q.queryKey[3] !== 'plan',
      });
    },
    onError: (err, edit) => {
      if (err instanceof PlanConflict) setConflict(() => edit);
    },
  });
  const pending = useMutationState({
    filters: { mutationKey, status: 'pending' },
    select: (m) => m.state.variables as PlanEdit,
  });
  const saved = query.data;
  const plan: Plan | undefined = saved ? pending.reduce((p, e) => e(p), saved.plan) : undefined;
  const failure = save.error instanceof PlanConflict ? null : save.error;
  return {
    query,
    plan,
    impact: saved?.impact,
    edit: (e: PlanEdit) => save.mutate(e),
    saving: pending.length > 0,
    failure,
    conflict: conflict !== null,
    /** Applies the stopped edit on the plan as it is now. */
    replay: () => {
      if (conflict) save.mutate(conflict);
      setConflict(null);
    },
    dismiss: () => setConflict(null),
  };
}

export type PlanApi = ReturnType<typeof usePlan>;
