import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '../../../lib/api/client';
import { migrationKeys } from '../../../lib/api/migrations';
import { runJob, type ReloadMethod } from '../../../lib/api/preflight';
import type { AdminPassword } from '../wizard-context';
import type { StartKey } from './messages';
/** What still stands before Start, first first; null when nothing does. */
export function startGate(s: {
  blockers: number;
  checked: boolean;
  changed: boolean;
  accepted: boolean;
  reload: ReloadMethod | undefined;
  adminMissing: boolean;
}): StartKey | null {
  if (s.blockers > 0) return null;
  if (!s.checked) return 'needCheck';
  if (s.changed && !s.accepted) return 'needAccept';
  if (!s.reload) return 'needReload';
  if (s.adminMissing) return 'adminMissing';
  return null;
}
/** The live view of a job (4.6). */
export const livePath = (jobId: string) => `/migrations/${encodeURIComponent(jobId)}/live`;
/**
 * Starts the run (POST run) with the plan version the check returned, the
 * chosen reload and the admin password typed in Compare (D10); then opens
 * the live view. A 412 means the outcome moved after the check.
 */
export function useStart(jobId: string, onStale: () => void) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const run = useMutation({
    // The body can carry the admin password: never keep it in the cache (D10).
    gcTime: 0,
    mutationFn: (v: { planVersion: number; reload: ReloadMethod; password: AdminPassword }) =>
      runJob(jobId, {
        planVersion: v.planVersion,
        reload: v.reload,
        ...(v.password.choice === 'new' ? { adminPassword: v.password.value } : {}),
      }),
    onSuccess: () => {
      // Marked stale only: while the run runs, the job's reads answer 409 busy.
      void qc.invalidateQueries({ queryKey: migrationKeys.all, refetchType: 'none' });
      void navigate(livePath(jobId));
    },
    onError: (err) => {
      setConfirming(false);
      if (err instanceof ApiError && err.status === 412) onStale();
    },
  });
  const stale = run.error instanceof ApiError && run.error.status === 412;
  return { run, stale, confirming, setConfirming };
}
