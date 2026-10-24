import { useEffect, useReducer } from 'react';
import { apiURL } from './client';

/** One event of a migration's stream (api-migration.md section 3). */
export interface MigrationEvent {
  seq: number;
  at: string;
  /** check, backup, execute, verify, reload, undo, restore for a run; read, build for a compare; "" on done. */
  stage: string;
  type: 'stage' | 'progress' | 'log' | 'done';
  data: {
    state?: string;
    done?: number;
    total?: number;
    etaSeconds?: number;
    pass?: number;
    code?: string;
    [extra: string]: unknown;
  };
}

export type StageState = 'start' | 'done' | 'failed';

export interface StageInfo {
  stage: string;
  state: StageState;
  startedAt: string;
  endedAt?: string;
}

export interface StreamProgress {
  stage: string;
  done: number;
  total: number;
  /** The read pass of a compare (1, then 2 to confirm); 0 for a run. */
  pass: number;
  /** From the server; 0 when it has no estimate. */
  etaSeconds: number;
}

export interface LogLine {
  seq: number;
  at: string;
  stage: string;
  code: string;
  /** Extras next to the code, such as `left` or a certificate's `sha256`. */
  detail: Readonly<Record<string, unknown>>;
}

/** What the stream says about the job's latest compare or run. */
export interface JobStream {
  lastSeq: number;
  kind: 'none' | 'compare' | 'run';
  /** In the order they started. */
  stages: StageInfo[];
  progress: StreamProgress | null;
  log: LogLine[];
  /** The log line before the stage that failed. */
  failure: { code: string; detail: Readonly<Record<string, unknown>> } | null;
  /** The state a run ended in; null until it ends. */
  done: string | null;
}

export const emptyStream: JobStream = {
  lastSeq: 0,
  kind: 'none',
  stages: [],
  progress: null,
  log: [],
  failure: null,
  done: null,
};

const stageStates: readonly string[] = ['start', 'done', 'failed'];

function onStage(s: JobStream, e: MigrationEvent): JobStream {
  const state = e.data.state;
  if (typeof state !== 'string' || !stageStates.includes(state)) return s;
  // One stream carries every compare and run; a new one starts afresh.
  if (state === 'start' && (e.stage === 'read' || e.stage === 'check')) {
    s = { ...emptyStream, lastSeq: s.lastSeq, kind: e.stage === 'read' ? 'compare' : 'run' };
  }
  const i = s.stages.findIndex((x) => x.stage === e.stage);
  const next: StageInfo =
    state === 'start'
      ? { stage: e.stage, state, startedAt: e.at }
      : {
          stage: e.stage,
          state: state as StageState,
          startedAt: s.stages[i]?.startedAt ?? e.at,
          endedAt: e.at,
        };
  const stages = i < 0 ? [...s.stages, next] : s.stages.map((x, j) => (j === i ? next : x));
  const last = s.log.at(-1);
  const failure =
    state === 'failed'
      ? last
        ? { code: last.code, detail: last.detail }
        : { code: 'internal', detail: {} }
      : s.failure;
  return { ...s, stages, failure };
}

function onLog(s: JobStream, e: MigrationEvent): JobStream {
  const { code, ...detail } = e.data;
  if (typeof code !== 'string') return s;
  return { ...s, log: [...s.log, { seq: e.seq, at: e.at, stage: e.stage, code, detail }] };
}

/** Folds one event into the stream. Events already seen (a reconnect replays some) change nothing. */
export function reduceStream(s: JobStream, e: MigrationEvent): JobStream {
  if (e.seq <= s.lastSeq) return s;
  s = { ...s, lastSeq: e.seq };
  switch (e.type) {
    case 'stage':
      return onStage(s, e);
    case 'progress':
      return {
        ...s,
        progress: {
          stage: e.stage,
          done: e.data.done ?? 0,
          total: e.data.total ?? 0,
          pass: e.data.pass ?? 0,
          etaSeconds: e.data.etaSeconds ?? 0,
        },
      };
    case 'log':
      return onLog(s, e);
    case 'done':
      return { ...s, done: e.data.state ?? 'done' };
    default:
      return s;
  }
}

export function reduceEvents(events: readonly MigrationEvent[], from = emptyStream): JobStream {
  return events.reduce(reduceStream, from);
}

export type CompareStatus = 'idle' | 'reading' | 'building' | 'built' | 'failed';

/** Where the latest compare stands; idle when the stream shows no compare. */
export function compareStatus(s: JobStream): CompareStatus {
  if (s.kind !== 'compare') return 'idle';
  if (s.stages.some((x) => x.state === 'failed')) return 'failed';
  const build = s.stages.find((x) => x.stage === 'build');
  if (build?.state === 'done') return 'built';
  return build ? 'building' : 'reading';
}

export interface Followed {
  id: string;
  stream: JobStream;
  lost: boolean;
}

export type FollowAction = { id: string; event: MigrationEvent } | { id: string; lost: true };

/** The hook's state: one job's stream at a time. */
export function follow(s: Followed, a: FollowAction): Followed {
  // A new job starts from an empty stream.
  const base = s.id === a.id ? s : { id: a.id, stream: emptyStream, lost: false };
  if ('lost' in a) return { ...base, lost: true };
  const stream = reduceStream(base.stream, a.event);
  return stream === base.stream && base === s ? s : { ...base, stream };
}

const eventTypes = ['stage', 'progress', 'log', 'done'] as const;

/**
 * Follows a job's event stream (D3). EventSource resumes with Last-Event-ID
 * by itself after a dropped connection; the stream is closed once a run is
 * done. `lost` is true when the browser gave up on the stream.
 */
export function useJobEvents(id: string | undefined): { stream: JobStream; lost: boolean } {
  const [state, dispatch] = useReducer(follow, { id: '', stream: emptyStream, lost: false });
  useEffect(() => {
    if (!id) return;
    const source = new EventSource(apiURL(`migrations/${encodeURIComponent(id)}/events`));
    const onEvent = (msg: Event) => {
      let event: MigrationEvent;
      try {
        event = JSON.parse((msg as MessageEvent<string>).data) as MigrationEvent;
      } catch {
        return;
      }
      dispatch({ id, event });
      if (event.type === 'done') source.close();
    };
    for (const type of eventTypes) source.addEventListener(type, onEvent);
    source.onerror = () => {
      if (source.readyState === EventSource.CLOSED) dispatch({ id, lost: true });
    };
    return () => source.close();
  }, [id]);
  if (!id || state.id !== id) return { stream: emptyStream, lost: false };
  return { stream: state.stream, lost: state.lost };
}
