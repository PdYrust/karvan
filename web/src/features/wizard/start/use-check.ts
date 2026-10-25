import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import {
  compareStatus,
  useJobEvents,
  type CompareStatus,
  type JobStream,
} from '../../../lib/api/events';
import { cancelJob, compareJob, migrationKeys } from '../../../lib/api/migrations';
import { getChanges, preflightKeys } from '../../../lib/api/preflight';
/**
 * Where a check that began in this visit stands. Older compares in the
 * stream's history began at or before `since` and do not count.
 */
export function checkStatus(stream: JobStream, since: number | null): CompareStatus {
  if (since === null) return 'idle';
  return stream.startSeq <= since ? 'reading' : compareStatus(stream);
}
/**
 * "Check again" (B3): reads both panels again through a compare, follows
 * it on the job's stream, then reads what changed. Start waits for a check
 * of this visit, and for the operator to accept any change.
 */
export function useCheck(jobId: string) {
  const qc = useQueryClient();
  const { stream } = useJobEvents(jobId);
  const [since, setSince] = useState<number | null>(null);
  const [accepted, setAccepted] = useState(false);
  const [offset, setOffset] = useState(0);
  const compare = useMutation({ mutationFn: () => compareJob(jobId) });
  const cancel = useMutation({ mutationFn: () => cancelJob(jobId) });
  const status = compare.isError ? 'idle' : checkStatus(stream, since);
  const built = status === 'built';
  const seq = stream.startSeq;
  // A new build has a new plan version and new counts: read the job again.
  useEffect(() => {
    if (built) void qc.invalidateQueries({ queryKey: migrationKeys.job(jobId) });
  }, [built, seq, jobId, qc]);
  const changes = useQuery({
    queryKey: [...preflightKeys.changes(jobId, offset), seq],
    queryFn: () => getChanges(jobId, offset),
    enabled: built,
    placeholderData: keepPreviousData,
  });
  return {
    status,
    stream,
    /** The stream's history has arrived, so a new compare can be told from an old one. */
    ready: stream.lastSeq > 0,
    failure: status === 'failed' ? stream.failure : null,
    compare,
    changes: built ? changes : null,
    offset,
    setOffset,
    accepted,
    setAccepted,
    run: () => {
      setSince(stream.lastSeq);
      setAccepted(false);
      setOffset(0);
      compare.mutate();
    },
    stop: () => cancel.mutate(),
    stopping: cancel.isPending,
    /** A run refused as stale needs a new check. */
    reset: () => setSince(null),
  };
}
