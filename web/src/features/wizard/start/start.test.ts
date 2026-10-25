import { describe, expect, it } from 'vitest';
import { emptyStream, reduceEvents, type MigrationEvent } from '../../../lib/api/events';
import type { JobSummary } from '../../../lib/api/migrations';
import { checkStatus } from './use-check';
import { startGate } from './use-start';
import { adminLine } from './UserEffects';

const ev = (seq: number, stage: string, state: string): MigrationEvent => ({
  seq,
  at: '2026-10-21T10:00:00.000Z',
  stage,
  type: 'stage',
  data: { state },
});
const compare = (from: number) => [
  ev(from, 'read', 'start'),
  ev(from + 1, 'read', 'done'),
  ev(from + 2, 'build', 'start'),
  ev(from + 3, 'build', 'done'),
];

describe('checkStatus', () => {
  it('is idle until the operator asks for a check', () => {
    expect(checkStatus(reduceEvents(compare(1)), null)).toBe('idle');
  });
  it('does not take the compare from the history for the new one', () => {
    const old = reduceEvents(compare(1));
    expect(checkStatus(old, old.lastSeq)).toBe('reading');
  });
  it('follows the compare that began after the click', () => {
    const old = reduceEvents(compare(1));
    const since = old.lastSeq;
    expect(checkStatus(reduceEvents(compare(5).slice(0, 2), old), since)).toBe('reading');
    expect(checkStatus(reduceEvents(compare(5), old), since)).toBe('built');
  });
  it('says when the new read failed', () => {
    const s = reduceEvents([ev(1, 'read', 'start'), ev(2, 'read', 'failed')], emptyStream);
    expect(checkStatus(s, 0)).toBe('failed');
  });
});

const open = {
  blockers: 0,
  checked: true,
  changed: false,
  accepted: false,
  reload: 'manual' as const,
  adminMissing: false,
};

describe('startGate', () => {
  it('lets the run start once everything is in place', () => {
    expect(startGate(open)).toBeNull();
  });
  it('asks for the check, then the acceptance, then the reload, then the password', () => {
    expect(startGate({ ...open, checked: false })).toBe('needCheck');
    expect(startGate({ ...open, changed: true })).toBe('needAccept');
    expect(startGate({ ...open, changed: true, accepted: true })).toBeNull();
    expect(startGate({ ...open, reload: undefined })).toBe('needReload');
    expect(startGate({ ...open, adminMissing: true })).toBe('adminMissing');
  });
});

const summary = (merged: boolean, canKeep: boolean) =>
  ({
    newAdmin: { username: 'reseller', merged, roleId: 3 },
    canKeepPassword: canKeep,
  }) as JobSummary;

describe('adminLine', () => {
  it('says how the moved admin gets its password', () => {
    expect(adminLine(summary(true, true), { choice: 'new', value: '' }, false)).toBe('adminMerged');
    expect(adminLine(summary(false, true), { choice: 'keep', value: '' }, false)).toBe('adminKeep');
    expect(adminLine(summary(false, false), { choice: 'keep', value: '' }, false)).toBe(
      'adminMade',
    );
    expect(adminLine(summary(false, true), { choice: 'new', value: 'x' }, true)).toBe('adminTyped');
  });
  it('flags a new password that is empty or breaks a rule', () => {
    expect(adminLine(summary(false, true), { choice: 'new', value: '' }, false)).toBe(
      'adminMissing',
    );
  });
});
