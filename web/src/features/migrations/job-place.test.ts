import { describe, expect, it } from 'vitest';
import { sitePrefill } from '../../lib/api/links';
import type { JobRow } from '../../lib/api/migrations';
import { againPath, jobPlace, loginDropped, matchesJob, statesOf } from './job-place';

const row = (patch: Partial<JobRow> = {}): JobRow => ({
  id: '9f1c2b3a4d5e6f7081920a1b',
  state: 'checked',
  panel: { id: 2, name: 'karvan-test-pasarguard', kind: 'pasarguard' },
  source: { kind: 'marzban', origin: 'https://old.example.com:8443', admin: 'reseller1' },
  safeMode: true,
  counts: null,
  createdAt: '2026-10-17T09:06:00Z',
  updatedAt: '2026-10-17T09:06:00Z',
  endedAt: null,
  loginHeld: false,
  ...patch,
});

describe('jobPlace', () => {
  const id = '9f1c2b3a4d5e6f7081920a1b';
  it('sends a draft to its steps, and a read still going to its counter', () => {
    expect(jobPlace(row({ loginHeld: true }))).toBe(`/migrations/${id}/summary`);
    expect(jobPlace(row({ state: 'created', loginHeld: true }))).toBe(`/migrations/new?job=${id}`);
  });
  it('starts a draft Karvan signed out of again from its address', () => {
    expect(jobPlace(row())).toBe(againPath(row()));
    expect(jobPlace(row({ state: 'created' }))).toBe(againPath(row()));
    expect(jobPlace(row({ source: { kind: 'marzban', origin: '', admin: '' } }))).toBe(
      `/migrations/${id}/summary`,
    );
  });
  it('sends a run and a job that may need a rollback to the live view', () => {
    for (const state of ['running', 'failed', 'interrupted'] as const) {
      expect(jobPlace(row({ state }))).toBe(`/migrations/${id}/live`);
    }
  });
  it('sends a finished job to its report', () => {
    expect(jobPlace(row({ state: 'done' }))).toBe(`/reports/${id}`);
    expect(jobPlace(row({ state: 'rolled_back' }))).toBe(`/reports/${id}`);
  });
});

describe('loginDropped', () => {
  it('holds only for drafts without the login', () => {
    expect(loginDropped(row())).toBe(true);
    expect(loginDropped(row({ loginHeld: true }))).toBe(false);
    expect(loginDropped(row({ state: 'done' }))).toBe(false);
  });
});

describe('againPath', () => {
  it('fills in the destination, the address and the kind, and nothing else', () => {
    const path = againPath(row());
    expect(path.startsWith('/migrations/new?')).toBe(true);
    expect(path).not.toContain('reseller1');
    expect(sitePrefill(new URL(path, 'http://x').searchParams)).toEqual({
      panelId: 2,
      sourceKind: 'marzban',
      sourceUrl: 'https://old.example.com:8443',
    });
  });
});

describe('statesOf', () => {
  it('asks for both draft states as one filter, and for nothing on all', () => {
    expect(statesOf('all')).toEqual([]);
    expect(statesOf('draft')).toEqual(['created', 'checked']);
    expect(statesOf('failed')).toEqual(['failed']);
  });
});

describe('matchesJob', () => {
  it('finds a job by its admin, address or panel, in either language', () => {
    expect(matchesJob(row(), '')).toBe(true);
    expect(matchesJob(row(), 'RESELLER')).toBe(true);
    expect(matchesJob(row(), 'old.example')).toBe(true);
    expect(matchesJob(row(), 'test-pasarguard')).toBe(true);
    expect(matchesJob(row(), 'مرزبان')).toBe(true);
    expect(matchesJob(row(), 'someone else')).toBe(false);
  });
});
