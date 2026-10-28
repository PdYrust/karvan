import { describe, expect, it } from 'vitest';
import type { FixOption, ProblemGroup, UserRow } from '../../../lib/api/migrations';
import type { Plan } from '../../../lib/plan/plan';
import {
  appliedFix,
  byRelevance,
  defaultFix,
  fixForAll,
  fixWords,
  typeWords,
  userFixes,
} from './problems';

const opt = (id: string, changesConfig = false): FixOption => ({
  id,
  changesConfig,
  changesLink: false,
});
const group = (type: string, count: number, open = 0, options = [opt('skip')]): ProblemGroup => ({
  type,
  count,
  open,
  options,
});
const plan: Plan = {
  version: 0,
  now: '2026-10-17T09:06:00Z',
  targets: null,
  tree: null,
};

describe('byRelevance', () => {
  it('puts groups that still need a fix first, then blockers, warnings and notes, then size', () => {
    const order = byRelevance([
      group('C26', 900),
      group('C18', 40),
      group('C02', 3),
      group('C11', 2, 2),
      group('C07', 50),
    ]).map((g) => g.type);
    expect(order).toEqual(['C11', 'C07', 'C02', 'C18', 'C26']);
  });
  it('ranks C10 and C38 as blockers (phase 4.1 D37, D38)', () => {
    const order = byRelevance([group('C18', 90), group('C38', 2), group('C10', 5)]);
    expect(order.map((g) => g.type)).toEqual(['C10', 'C38', 'C18']);
  });
});

describe('defaultFix and fixForAll', () => {
  it('uses the first fix when the type has a default that keeps configs', () => {
    const g = group('C02', 3, 0, [opt('suffix'), opt('skip')]);
    expect(defaultFix(g)?.id).toBe('suffix');
    expect(fixForAll(plan, g)).toEqual({ option: 'suffix' });
  });

  it('has no default for a type that needs a choice, or a first fix that changes configs', () => {
    expect(defaultFix(group('C11', 2, 2, [opt('accept', true), opt('skip')]))).toBeUndefined();
    expect(defaultFix(group('C24', 2, 0, [opt('new_credential', true)]))).toBeUndefined();
  });
  it('has no default for C10 and C38, whose first fix leaves the user out', () => {
    for (const type of ['C10', 'C38']) {
      expect(defaultFix(group(type, 2, 2, [opt('skip'), opt('accept', true)]))).toBeUndefined();
    }
  });

  it('prefers the fix chosen for all', () => {
    const g = group('C02', 3, 0, [opt('suffix'), opt('prefix')]);
    const chosen = { option: 'prefix', params: { text: 'old_' } };
    expect(fixForAll({ ...plan, resolutions: { C02: chosen } }, g)).toEqual(chosen);
  });
});

describe('appliedFix', () => {
  const row = (r: Partial<UserRow>): UserRow => ({
    key: 'a',
    sourceUsername: 'a',
    username: 'a',
    include: true,
    status: 'active',
    usedTraffic: 0,
    dataLimit: 0,
    expire: null,
    categories: [],
    conflicts: [],
    applied: [],
    changesConfig: false,
    changesLink: false,
    ...r,
  });

  it('reads an open conflict with its fix, or a fix that already applied', () => {
    expect(
      appliedFix(row({ conflicts: [{ type: 'C10', severity: 'warning', fix: 'accept' }] }), 'C10'),
    ).toBe('accept');
    expect(appliedFix(row({ applied: ['op:note.set', 'fix:C02:suffix'] }), 'C02')).toBe('suffix');
    expect(appliedFix(row({}), 'C02')).toBeUndefined();
  });
});

describe('userFixes', () => {
  it('leaves out the fixes that need a value shared by all', () => {
    const g = group('C02', 3, 0, [opt('suffix'), opt('prefix'), opt('rename'), opt('skip')]);
    expect(userFixes(g).map((o) => o.id)).toEqual(['suffix', 'rename', 'skip']);
  });
});

describe('words', () => {
  it('names known types and fixes, and falls back for new ones', () => {
    expect(typeWords('C16b')).toBe('type_C16b');
    expect(typeWords('C99')).toBe('type_unknown');
    expect(fixWords('new_credential')).toBe('fix_new_credential');
    expect(fixWords('magic')).toBe('fix_unknown');
    expect(typeWords('C38')).toBe('type_C38');
    expect(typeWords('C36')).toBe('type_unknown');
  });
  it('calls accept on C10 and C38 a move anyway', () => {
    expect(fixWords('accept', 'C10')).toBe('fix_move_anyway');
    expect(fixWords('accept', 'C38')).toBe('fix_move_anyway');
    expect(fixWords('accept', 'C11')).toBe('fix_accept');
    expect(fixWords('skip', 'C38')).toBe('fix_skip');
  });
});
