import { describe, expect, it } from 'vitest';
import type { Counts } from '../../../lib/api/migrations';
import type { Report, ReportUser } from '../../../lib/api/report';
import { filterCounts, knownUndoCodes, undoWindow, userMatches, userOutcome } from './report-view';

const user = (patch: Partial<ReportUser>): ReportUser => ({
  key: 'k',
  sourceUsername: 'alice',
  username: 'alice',
  moved: true,
  changesConfig: false,
  changesLink: true,
  ...patch,
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

const report = (patch: Partial<Report>): Report => ({
  schema: 'karvan.report/v1',
  jobId: 'j',
  state: 'done',
  safeMode: true,
  startedAt: '2026-10-17T09:06:00Z',
  endedAt: '2026-10-17T09:12:41Z',
  source: { kind: 'marzban', origin: 'https://old.example.com', admin: 'reseller1' },
  destination: { kind: 'pasarguard', name: 'pg', db: 'sqlite' },
  admin: { username: 'reseller1', merged: false },
  counts: counts({ moved: 2 }),
  users: null,
  notUndoable: null,
  reload: { method: '', ok: 0 },
  stages: null,
  ...patch,
});

describe('report users', () => {
  const alice = user({});
  const bob = user({ key: 'b', sourceUsername: 'bob', username: 'bob_2', changesConfig: true });
  const carol = user({
    key: 'c',
    sourceUsername: 'carol',
    username: 'carol',
    moved: false,
    changesLink: false,
  });
  const users = [alice, bob, carol];
  it('counts what each filter keeps', () => {
    expect(filterCounts(users)).toEqual({
      all: 3,
      moved: 2,
      left: 1,
      renamed: 1,
      config: 1,
      link: 2,
    });
  });
  it('searches the old and the new name within a filter', () => {
    expect(userMatches(bob, 'renamed', 'BOB_2')).toBe(true);
    expect(userMatches(bob, 'renamed', 'bob')).toBe(true);
    expect(userMatches(bob, 'left', '')).toBe(false);
    expect(userMatches(carol, 'all', 'ali')).toBe(false);
  });
});

describe('undo', () => {
  it('keeps the codes Karvan has words for, once each', () => {
    const r = report({
      notUndoable: ['notifications_sent', 'something_new', 'notifications_sent'],
    });
    expect(knownUndoCodes(r)).toEqual(['notifications_sent']);
  });
  it('is open for a failed run that left users behind', () => {
    expect(undoWindow('failed', report({ counts: counts({ moved: 5 }) }))).toEqual({
      kind: 'open',
    });
    expect(undoWindow('failed', report({ counts: counts({ moved: 5, undone: 5 }) }))).toEqual({
      kind: 'closed',
    });
  });
  it('closes when the panel reloaded, at the end of the reload', () => {
    const r = report({
      notUndoable: ['after_reload'],
      stages: [
        {
          stage: 'reload',
          started: '2026-10-17T09:12:00Z',
          ended: '2026-10-17T09:12:40Z',
          ok: true,
        },
      ],
    });
    expect(undoWindow('done', r)).toEqual({ kind: 'reloaded', at: '2026-10-17T09:12:40Z' });
    expect(undoWindow('done', report({}))).toEqual({ kind: 'closed' });
  });
});

describe('userOutcome', () => {
  it('says a rollback took a moved user back', () => {
    expect(userOutcome({ moved: true }, 'done')).toBe('outcomeMoved');
    expect(userOutcome({ moved: true }, 'rolled_back')).toBe('outcomeUndone');
    expect(userOutcome({ moved: false }, 'rolled_back')).toBe('outcomeLeft');
  });
});
