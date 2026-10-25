import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useJobEvents } from '../../lib/api/events';
import { getJobRow, getPlan, migrationKeys, originHost } from '../../lib/api/migrations';
import { getReport } from '../../lib/api/report';
import type { SceneEnds } from './FlowScene';
import { canStop, counterOf, endStateOf, phaseOf, railOf, stagesOfReport } from './live';

/**
 * Everything the live view reads (4.6, D3): the job's row by id (it answers
 * while the job runs and after a restart), its event stream, the plan's
 * skipped users while it runs, and the report once it ends (and while a
 * backup goes back, for the backup's id). `after` is the last event before
 * a backup restore this page follows (B12).
 */
export function useLive(id: string, after = 0) {
  const qc = useQueryClient();
  const row = useQuery({ queryKey: migrationKeys.row(id), queryFn: () => getJobRow(id) });
  const { stream, lost } = useJobEvents(id, after);
  const r = row.data ?? null;
  const phase = r ? phaseOf(stream, r) : null;
  const end = r ? endStateOf(stream, r) : null;
  const hasReport =
    phase === 'ended' && (end === 'done' || end === 'failed' || end === 'rolled_back');
  const report = useQuery({
    queryKey: migrationKeys.report(id),
    queryFn: () => getReport(id),
    enabled: hasReport || phase === 'restore',
  });
  // Skipped users never pass through a batch; the plan the run took counts them.
  const plan = useQuery({
    queryKey: migrationKeys.plan(id),
    queryFn: () => getPlan(id),
    enabled: phase === 'run',
    staleTime: Infinity,
  });
  // A run or a rollback ended: its state, counts and report are new.
  const ended = stream.done !== null ? stream.lastSeq : 0;
  useEffect(() => {
    if (ended) void qc.invalidateQueries({ queryKey: migrationKeys.all });
  }, [ended, qc]);

  const stages =
    stream.kind === 'run' && stream.stages.length > 0
      ? stream.stages
      : stagesOfReport(report.data?.stages);
  const ends: SceneEnds | null = r
    ? {
        from: { kind: r.source.kind, host: originHost(r.source.origin) },
        to: { kind: r.panel.kind, name: r.panel.name },
      }
    : null;
  return {
    row,
    stream,
    lost,
    phase,
    end,
    report,
    ends,
    rail: r && phase ? railOf(stages, r, phase) : [],
    counter: phase ? counterOf(stream, phase) : null,
    skipped: report.data?.counts.skipped ?? plan.data?.impact?.skipped ?? 0,
    stoppable: r && phase ? canStop(stream, r, phase) : false,
  };
}
