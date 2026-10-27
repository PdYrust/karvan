import type { JobStream, StageInfo } from '../../lib/api/events';
import type { Counts, JobRow, JobState } from '../../lib/api/migrations';
import type { Report, StageTime } from '../../lib/api/report';

/** The stages every run shows, in order (4.6); Undo and Restore join only when they run. */
export const runStages = ['check', 'backup', 'execute', 'verify', 'reload'] as const;
export const extraStages = ['undo', 'restore'] as const;
export type RunStage = (typeof runStages)[number] | (typeof extraStages)[number];

const isRunStage = (s: string): s is RunStage =>
  (runStages as readonly string[]).includes(s) || (extraStages as readonly string[]).includes(s);

/** What the page shows: a job that never ran, a run, an undo, a restore, or the result. */
export type Phase = 'idle' | 'run' | 'undo' | 'restore' | 'ended';

/** The job's row, as far as the live view reads it. */
export type LiveRow = Pick<JobRow, 'state' | 'updatedAt' | 'safeMode'>;

/** A stage a restart cut off: it started before the job was marked interrupted. */
function cutOff(s: StageInfo, row: LiveRow): boolean {
  return row.state === 'interrupted' && Date.parse(s.startedAt) <= Date.parse(row.updatedAt);
}

/** The stage running now, if any: the last one that started and has not ended. */
function liveStage(stream: JobStream, row: LiveRow): StageInfo | undefined {
  // A compare reads too, but it is not what this page follows.
  if (stream.kind !== 'run' || stream.done !== null) return undefined;
  const s = stream.stages.findLast((x) => x.state === 'start');
  return s && !cutOff(s, row) ? s : undefined;
}

export function phaseOf(stream: JobStream, row: LiveRow): Phase {
  const live = liveStage(stream, row);
  if (live) return live.stage === 'undo' || live.stage === 'restore' ? live.stage : 'run';
  if (stream.kind === 'run' && stream.done !== null) return 'ended';
  if (row.state === 'running') return 'run';
  if (row.state === 'created' || row.state === 'checked') return 'idle';
  return 'ended';
}

/** How a job ended: the stream's last word, else the row's state. "checked" is a run that wrote nothing. */
export type EndState = JobState;

export function endStateOf(stream: JobStream, row: LiveRow): EndState {
  if (stream.kind === 'run' && stream.done !== null) return stream.done as EndState;
  return row.state;
}

export type StepState = 'waiting' | 'running' | 'done' | 'failed' | 'off' | 'notRun' | 'stopped';

export interface RailStep {
  stage: RunStage;
  state: StepState;
  startedAt?: string;
  endedAt?: string;
}

/** The report's stage times as the stream's stages, for a job whose events were trimmed. */
export function stagesOfReport(stages: readonly StageTime[] | null | undefined): StageInfo[] {
  return (stages ?? []).map((s) => ({
    stage: s.stage,
    state: s.ok ? 'done' : 'failed',
    startedAt: s.started,
    endedAt: s.ended,
  }));
}

/**
 * The stage rail: the five run stages, then Undo and Restore when they ran.
 * Once a run failed or ended, a stage that never started did not run.
 */
export function railOf(stages: readonly StageInfo[], row: LiveRow, phase: Phase): RailStep[] {
  const over = phase !== 'run';
  const step = (stage: RunStage): RailStep => {
    const s = stages.find((x) => x.stage === stage);
    if (!s) {
      if (stage === 'backup' && !row.safeMode) return { stage, state: 'off' };
      return { stage, state: over ? 'notRun' : 'waiting' };
    }
    const at = { startedAt: s.startedAt, ...(s.endedAt ? { endedAt: s.endedAt } : {}) };
    if (s.state === 'done' || s.state === 'failed') return { stage, state: s.state, ...at };
    const running = !cutOff(s, row) && (phase === 'run' || phase === stage);
    return { stage, state: running ? 'running' : 'stopped', ...at };
  };
  const extras = stages
    .map((s) => s.stage)
    .filter((s): s is RunStage => isRunStage(s) && (extraStages as readonly string[]).includes(s));
  return [...runStages.map(step), ...extras.map(step)];
}

export interface Counter {
  kind: 'move' | 'undo';
  done: number;
  total: number;
  /** From the server; 0 when it has none. */
  etaSeconds: number;
}

/** The live counter: users written during a run, rows handled during an undo. */
export function counterOf(stream: JobStream, phase: Phase): Counter | null {
  const p = stream.progress;
  if (!p) return null;
  if (phase === 'run' && p.stage === 'execute') {
    return { kind: 'move', done: p.done, total: p.total, etaSeconds: p.etaSeconds };
  }
  if (phase === 'undo' && p.stage === 'undo') {
    return { kind: 'undo', done: p.done, total: p.total, etaSeconds: 0 };
  }
  return null;
}

/** The restore running now, or the last one: its stage on the stream. */
const lastRestore = (stream: JobStream) => stream.stages.findLast((s) => s.stage === 'restore');
/** Whether the panel came online again during the restore that runs now (B12). */
export function panelBack(stream: JobStream): boolean {
  const r = lastRestore(stream);
  if (!r) return false;
  const since = Date.parse(r.startedAt);
  return stream.log.some((l) => l.code === 'panel_started' && Date.parse(l.at) >= since);
}
/**
 * Whether the stream ended on a backup restore that failed and left the
 * job's state as it was, so the result alone would not say so.
 */
export function restoreFailed(stream: JobStream): boolean {
  if (stream.kind !== 'run' || stream.done === null || stream.done === 'rolled_back') return false;
  const last = stream.stages.at(-1);
  return last?.stage === 'restore' && last.state === 'failed';
}
/** Stop works between chunks, until the users are written; after that the run is nearly done. */
export function canStop(stream: JobStream, row: LiveRow, phase: Phase): boolean {
  if (phase !== 'run') return false;
  return !stream.stages.some(
    (s) => (s.stage === 'verify' || s.stage === 'reload') && !cutOff(s, row),
  );
}

/** Whether a rollback could remove something: an interrupted run, or a failed one that left users. */
export function mayRollBack(
  state: EndState,
  counts: Counts | null | undefined,
  report: Pick<Report, 'left'> | undefined,
): boolean {
  if (state === 'interrupted') return true;
  if (state !== 'failed') return false;
  const written = (counts?.moved ?? 0) - (counts?.undone ?? 0);
  return written > 0 || (report?.left?.length ?? 0) > 0;
}

export const countKeys = [
  'moved',
  'skipped',
  'adopted',
  'renamed',
  'configChanged',
  'verified',
  'undone',
] as const satisfies readonly (keyof Counts)[];

/** The counts a result shows: moved always, the others when they are not zero. */
export function shownCounts(c: Counts): [(typeof countKeys)[number], number][] {
  return countKeys.filter((k) => k === 'moved' || c[k] > 0).map((k) => [k, c[k]]);
}
