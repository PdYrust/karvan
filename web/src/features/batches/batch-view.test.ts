import { describe, expect, it } from 'vitest';
import type { Batch } from '../../lib/api/batches';
import { batchActions, batchLook, matchesBatch } from './batch-view';

function batch(edit: Partial<Batch> = {}): Batch {
  return {
    id: 3,
    name: 'root 2',
    state: 'active',
    inactiveReason: '',
    inactiveAt: null,
    jobId: 'j1',
    panel: { id: 2, name: 'karvan-test-pasarguard', kind: 'pasarguard' },
    source: { kind: 'marzban', origin: 'https://old.example.com:8443' },
    admins: [{ source: 'root', dest: 'root_2', users: 9 }],
    oldLinks: { path: '', hosts: [], port: 0 },
    counts: { users: 9, verified: 0, closed: 0 },
    rules: [],
    createdAt: '2026-10-25T02:14:13Z',
    updatedAt: '2026-10-25T02:14:13Z',
    ...edit,
  };
}

const inactive = (inactiveReason: Batch['inactiveReason']) =>
  batch({ state: 'inactive', inactiveReason, inactiveAt: '2026-10-26T00:00:00Z' });

describe('what a batch offers', () => {
  it('lets an active batch only be renamed', () => {
    expect(batchActions(batch())).toEqual({ remove: false, reactivate: false });
  });
  it('lets a rolled back batch be deleted', () => {
    expect(batchActions(inactive('rolled_back'))).toEqual({ remove: true, reactivate: false });
  });
  it('lets a batch whose restore failed be deleted or made active again', () => {
    expect(batchActions(inactive('restore_failed'))).toEqual({ remove: true, reactivate: true });
  });
  it('offers nothing while a restore is under way', () => {
    expect(batchActions(inactive('restoring'))).toEqual({ remove: false, reactivate: false });
  });
});

describe('how a batch state reads', () => {
  it('shows active as success with no reason', () => {
    expect(batchLook(batch())).toEqual({ tone: 'success', word: 'stateActive' });
  });
  it('shows an inactive batch with its reason', () => {
    expect(batchLook(inactive('restoring'))).toEqual({
      tone: 'active',
      word: 'stateInactive',
      reason: 'reason_restoring',
    });
    expect(batchLook(inactive('rolled_back'))).toMatchObject({ tone: 'neutral' });
    expect(batchLook(inactive('restore_failed'))).toMatchObject({
      tone: 'danger',
      reason: 'reason_restore_failed',
    });
  });
});

describe('the batch search', () => {
  it('finds a batch by name, admin, address or new panel', () => {
    const b = batch();
    for (const term of ['ROOT 2', 'root_2', 'old.example', 'pasarguard', ' ']) {
      expect(matchesBatch(b, term)).toBe(true);
    }
  });
  it('leaves out a batch that does not match', () => {
    expect(matchesBatch(batch(), 'reseller')).toBe(false);
  });
});
