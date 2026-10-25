import { describe, expect, it } from 'vitest';
import { reduceEvents, type MigrationEvent } from '../../lib/api/events';
import type { Counts } from '../../lib/api/migrations';
import { codeSentence, eventSentence } from './log';
import {
  canStop,
  counterOf,
  endStateOf,
  mayRollBack,
  phaseOf,
  railOf,
  shownCounts,
  stagesOfReport,
  type LiveRow,
} from './live';

let seq = 0;
const at = (s: number) => new Date(Date.UTC(2026, 9, 24, 10, 0, s)).toISOString();
function ev(
  stage: string,
  type: MigrationEvent['type'],
  data: MigrationEvent['data'] = {},
  second = seq,
): MigrationEvent {
  seq += 1;
  return { seq, at: at(second), stage, type, data };
}
const stage = (name: string, state: string) => ev(name, 'stage', { state });
const row = (r: Partial<LiveRow> = {}): LiveRow => ({
  state: 'running',
  updatedAt: at(0),
  safeMode: true,
  ...r,
});
const counts = (c: Partial<Counts>): Counts => ({
  read: 0,
  moved: 0,
  skipped: 0,
  adopted: 0,
  undone: 0,
  configChanged: 0,
  renamed: 0,
  protocolsAdded: 0,
  verified: 0,
  ...c,
});

describe('phaseOf', () => {
  it('follows a run through its stages to the end', () => {
    seq = 0;
    const writing = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
    ]);
    expect(phaseOf(writing, row())).toBe('run');
    const ended = reduceEvents(
      [stage('execute', 'done'), ev('', 'done', { state: 'done' })],
      writing,
    );
    expect(phaseOf(ended, row())).toBe('ended');
    expect(endStateOf(ended, row())).toBe('done');
  });

  it('shows a running job as running before its stream arrives, and a draft as not started', () => {
    seq = 0;
    expect(phaseOf(reduceEvents([]), row())).toBe('run');
    expect(
      phaseOf(reduceEvents([ev('read', 'stage', { state: 'start' })]), row({ state: 'checked' })),
    ).toBe('idle');
  });

  it('tells an undo and a restore from the run', () => {
    seq = 0;
    const undo = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
      stage('execute', 'failed'),
      stage('undo', 'start'),
    ]);
    expect(phaseOf(undo, row())).toBe('undo');
    const restore = reduceEvents([stage('undo', 'done'), stage('restore', 'start')], undo);
    expect(phaseOf(restore, row())).toBe('restore');
  });

  it('ends an interrupted job whose stage a restart cut off, but follows a rollback after it', () => {
    seq = 0;
    const cut = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
    ]);
    const interrupted = row({ state: 'interrupted', updatedAt: at(30) });
    expect(phaseOf(cut, interrupted)).toBe('ended');
    expect(endStateOf(cut, interrupted)).toBe('interrupted');
    const rolling = reduceEvents([ev('undo', 'stage', { state: 'start' }, 40)], cut);
    expect(phaseOf(rolling, interrupted)).toBe('undo');
  });

  it('reads a job whose events were trimmed from its row', () => {
    expect(phaseOf(reduceEvents([]), row({ state: 'done' }))).toBe('ended');
    expect(endStateOf(reduceEvents([]), row({ state: 'rolled_back' }))).toBe('rolled_back');
  });

  it('ends a run the re-check sent back with nothing written', () => {
    seq = 0;
    const s = reduceEvents([
      stage('check', 'start'),
      stage('check', 'failed'),
      ev('check', 'log', { code: 'plan_stale' }),
      ev('', 'done', { state: 'checked' }),
    ]);
    expect(phaseOf(s, row({ state: 'checked' }))).toBe('ended');
    expect(endStateOf(s, row({ state: 'checked' }))).toBe('checked');
  });
});

describe('railOf', () => {
  it('lists the five run stages with Backup off when Safe Mode is off', () => {
    seq = 0;
    const s = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
    ]);
    const rail = railOf(s.stages, row({ safeMode: false }), 'run');
    expect(rail.map((x) => [x.stage, x.state])).toEqual([
      ['check', 'done'],
      ['backup', 'off'],
      ['execute', 'running'],
      ['verify', 'waiting'],
      ['reload', 'waiting'],
    ]);
    expect(rail[0]?.endedAt).toBe(at(1));
  });

  it('adds Undo once it runs, and marks what a failure skipped as not run', () => {
    seq = 0;
    const s = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('backup', 'start'),
      stage('backup', 'done'),
      stage('execute', 'start'),
      stage('execute', 'failed'),
      stage('undo', 'start'),
    ]);
    expect(railOf(s.stages, row(), 'undo').map((x) => [x.stage, x.state])).toEqual([
      ['check', 'done'],
      ['backup', 'done'],
      ['execute', 'failed'],
      ['verify', 'notRun'],
      ['reload', 'notRun'],
      ['undo', 'running'],
    ]);
  });

  it('marks a stage a restart cut off as stopped', () => {
    seq = 0;
    const s = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
    ]);
    const rail = railOf(s.stages, row({ state: 'interrupted', updatedAt: at(30) }), 'ended');
    expect(rail[2]?.state).toBe('stopped');
  });

  it("reads the report's stage times when the events are gone", () => {
    const stages = stagesOfReport([
      { stage: 'check', started: at(0), ended: at(2), ok: true },
      { stage: 'execute', started: at(2), ended: at(9), ok: false },
    ]);
    expect(
      railOf(stages, row({ state: 'failed', safeMode: false }), 'ended').map((x) => x.state),
    ).toEqual(['done', 'off', 'failed', 'notRun', 'notRun']);
    expect(stagesOfReport(null)).toEqual([]);
  });
});

describe('counterOf and canStop', () => {
  it('counts users written during a run, with the ETA, and rows handled during an undo', () => {
    seq = 0;
    const run = reduceEvents([
      stage('execute', 'start'),
      ev('execute', 'progress', { done: 500, total: 1170, etaSeconds: 42 }),
    ]);
    expect(counterOf(run, 'run')).toEqual({ kind: 'move', done: 500, total: 1170, etaSeconds: 42 });
    const undo = reduceEvents(
      [
        stage('execute', 'failed'),
        stage('undo', 'start'),
        ev('undo', 'progress', { done: 0, total: 500 }),
      ],
      run,
    );
    expect(counterOf(undo, 'undo')).toEqual({ kind: 'undo', done: 0, total: 500, etaSeconds: 0 });
    expect(counterOf(reduceEvents([]), 'run')).toBeNull();
  });

  it('lets the operator stop a run until its users are written', () => {
    seq = 0;
    const writing = reduceEvents([
      stage('check', 'start'),
      stage('check', 'done'),
      stage('execute', 'start'),
    ]);
    expect(canStop(writing, row(), 'run')).toBe(true);
    const verifying = reduceEvents([stage('execute', 'done'), stage('verify', 'start')], writing);
    expect(canStop(verifying, row(), 'run')).toBe(false);
    expect(canStop(writing, row(), 'undo')).toBe(false);
  });
});

describe('mayRollBack', () => {
  it('offers a rollback when a failed run left users, and always for an interrupted one', () => {
    expect(mayRollBack('failed', counts({ moved: 50 }), undefined)).toBe(true);
    expect(mayRollBack('failed', counts({ moved: 50, undone: 50 }), undefined)).toBe(false);
    expect(mayRollBack('failed', counts({ moved: 50, undone: 50 }), { left: [7] })).toBe(true);
    expect(mayRollBack('failed', counts({}), undefined)).toBe(false);
    expect(mayRollBack('interrupted', null, undefined)).toBe(true);
    expect(mayRollBack('done', counts({ moved: 9 }), undefined)).toBe(false);
  });
});

describe('shownCounts', () => {
  it('always shows moved and the other counts only when they are not zero', () => {
    expect(shownCounts(counts({ skipped: 3, renamed: 2 }))).toEqual([
      ['moved', 0],
      ['skipped', 3],
      ['renamed', 2],
    ]);
  });
});

describe('log sentences', () => {
  it('gives each stage event and end its own sentence', () => {
    expect(eventSentence(stage('execute', 'start'))).toEqual({
      from: 'live',
      key: 'start_execute',
    });
    expect(eventSentence(stage('undo', 'failed'))).toEqual({ from: 'live', key: 'failed_undo' });
    expect(eventSentence(ev('', 'done', { state: 'rolled_back' }))).toEqual({
      from: 'live',
      key: 'end_rolled_back',
    });
  });

  it('leaves out progress and what it does not know', () => {
    expect(eventSentence(ev('execute', 'progress', { done: 1, total: 2 }))).toBeNull();
    expect(eventSentence(stage('teleport', 'start'))).toBeNull();
    expect(eventSentence(ev('', 'done', { state: 'odd' }))).toBeNull();
    expect(eventSentence(stage('constructor', 'start'))).toBeNull();
  });

  it("says a run's own codes in the live words, and shared codes in the error words", () => {
    expect(codeSentence('undo_incomplete', { left: 3 })).toEqual({
      from: 'live',
      key: 'code_undo_incomplete',
      params: { count: 3 },
    });
    expect(codeSentence('cancelled')).toEqual({ from: 'error', key: 'cancelled' });
    expect(codeSentence('destination_locked')).toEqual({
      from: 'error',
      key: 'destination_locked',
    });
    expect(codeSentence('brand_new')).toEqual({ from: 'error', key: 'unknown' });
  });
});
