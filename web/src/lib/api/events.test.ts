import { describe, expect, it } from 'vitest';
import {
  compareStatus,
  emptyStream,
  follow,
  reduceEvents,
  reduceStream,
  type MigrationEvent,
} from './events';

let seq = 0;
function ev(
  stage: string,
  type: MigrationEvent['type'],
  data: MigrationEvent['data'] = {},
  at = '2026-10-21T10:00:00.000Z',
): MigrationEvent {
  seq += 1;
  return { seq, at, stage, type, data };
}
const fresh = () => {
  seq = 0;
};

describe('reduceStream', () => {
  it('follows a compare from the first read to the finished build', () => {
    fresh();
    const s = reduceEvents([
      ev('read', 'stage', { state: 'start' }, '2026-10-21T10:00:00.000Z'),
      ev('read', 'progress', { done: 500, total: 1200, pass: 1 }),
      ev('read', 'progress', { done: 1000, total: 1200, pass: 1 }),
      ev('read', 'stage', { state: 'done' }, '2026-10-21T10:00:04.000Z'),
      ev('build', 'stage', { state: 'start' }),
      ev('build', 'stage', { state: 'done' }),
    ]);
    expect(s.kind).toBe('compare');
    expect(s.stages.map((x) => [x.stage, x.state])).toEqual([
      ['read', 'done'],
      ['build', 'done'],
    ]);
    expect(s.stages[0]).toMatchObject({
      startedAt: '2026-10-21T10:00:00.000Z',
      endedAt: '2026-10-21T10:00:04.000Z',
    });
    expect(s.progress).toEqual({ stage: 'read', done: 1000, total: 1200, pass: 1, etaSeconds: 0 });
    expect(compareStatus(s)).toBe('built');
  });

  it('counts a second pass from the start again', () => {
    fresh();
    const s = reduceEvents([
      ev('read', 'stage', { state: 'start' }),
      ev('read', 'progress', { done: 1200, total: 1200, pass: 1 }),
      ev('read', 'progress', { done: 500, total: 1200, pass: 2 }),
    ]);
    expect(s.progress).toMatchObject({ done: 500, pass: 2 });
    expect(compareStatus(s)).toBe('reading');
  });

  it('keeps only the last compare when the history holds several', () => {
    fresh();
    const s = reduceEvents([
      ev('read', 'stage', { state: 'start' }),
      ev('read', 'log', { code: 'source_timeout' }),
      ev('read', 'stage', { state: 'failed' }),
      ev('read', 'stage', { state: 'start' }),
      ev('read', 'progress', { done: 9, total: 9, pass: 1 }),
    ]);
    expect(s.log).toEqual([]);
    expect(s.stages).toHaveLength(1);
    // The last compare began with the fourth event, so a page can tell it from an older one.
    expect(s.startSeq).toBe(4);
    expect(compareStatus(s)).toBe('reading');
  });

  it('names the failure of a compare with its extras', () => {
    fresh();
    const s = reduceEvents([
      ev('read', 'stage', { state: 'start' }),
      ev('read', 'log', { code: 'source_tls_untrusted', sha256: 'ab12', subject: 'CN=old' }),
      ev('read', 'stage', { state: 'failed' }),
    ]);
    expect(compareStatus(s)).toBe('failed');
    expect(s.failure).toEqual({
      code: 'source_tls_untrusted',
      detail: { sha256: 'ab12', subject: 'CN=old' },
    });
  });

  it('reads a run: stages in order, the batch counter with its ETA, then done', () => {
    fresh();
    const s = reduceEvents([
      ev('read', 'stage', { state: 'start' }),
      ev('read', 'stage', { state: 'done' }),
      ev('check', 'stage', { state: 'start' }),
      ev('check', 'stage', { state: 'done' }),
      ev('execute', 'stage', { state: 'start' }),
      ev('execute', 'progress', { done: 500, total: 1170, etaSeconds: 42 }),
      ev('execute', 'stage', { state: 'done' }),
      ev('', 'done', { state: 'done' }),
    ]);
    expect(s.kind).toBe('run');
    expect(s.stages.map((x) => x.stage)).toEqual(['check', 'execute']);
    expect(s.progress).toEqual({
      stage: 'execute',
      done: 500,
      total: 1170,
      pass: 0,
      etaSeconds: 42,
    });
    expect(s.done).toBe('done');
  });

  it('ignores an event it has already seen, as a reconnect may send it twice', () => {
    fresh();
    const start = ev('read', 'stage', { state: 'start' });
    const page = ev('read', 'progress', { done: 500, total: 1200, pass: 1 });
    const once = reduceEvents([start, page]);
    const twice = reduceStream(reduceStream(once, start), page);
    expect(twice).toBe(once);
    expect(twice.lastSeq).toBe(2);
  });

  it('keeps log lines in order and leaves the counter of another stage alone', () => {
    fresh();
    const s = reduceEvents([
      ev('check', 'stage', { state: 'start' }),
      ev('execute', 'stage', { state: 'start' }),
      ev('execute', 'progress', { done: 50, total: 100, etaSeconds: 3 }),
      ev('execute', 'log', { code: 'verify_failed' }),
      ev('undo', 'stage', { state: 'start' }),
      ev('undo', 'log', { code: 'undo_incomplete', left: 3 }),
    ]);
    expect(s.log.map((l) => l.code)).toEqual(['verify_failed', 'undo_incomplete']);
    expect(s.log[1]?.detail).toEqual({ left: 3 });
    expect(s.progress?.stage).toBe('execute');
  });

  it('starts empty', () => {
    expect(compareStatus(emptyStream)).toBe('idle');
    expect(emptyStream.lastSeq).toBe(0);
  });
});

describe('follow', () => {
  it('starts afresh when the page follows another job', () => {
    fresh();
    const start = ev('read', 'stage', { state: 'start' });
    const a = follow({ id: '', stream: emptyStream, lost: false }, { id: 'a', event: start });
    expect(a.stream.kind).toBe('compare');
    const lost = follow(a, { id: 'a', lost: true });
    expect(lost.lost).toBe(true);
    const b = follow(lost, { id: 'b', event: { ...start, seq: 1 } });
    expect(b).toMatchObject({ id: 'b', lost: false });
    expect(b.stream.lastSeq).toBe(1);
  });

  it('keeps the same state for an event it has seen', () => {
    fresh();
    const start = ev('read', 'stage', { state: 'start' });
    const a = follow({ id: '', stream: emptyStream, lost: false }, { id: 'a', event: start });
    expect(follow(a, { id: 'a', event: start })).toBe(a);
  });
});
