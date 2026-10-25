import { describe, expect, it } from 'vitest';
import {
  addChange,
  applyTemplate,
  changeTarget,
  filterOf,
  filterCondition,
  opFlags,
  removeChange,
  signedAmount,
  type Template,
} from './changes';
import type { Plan } from './plan';

const plan = (over: Partial<Plan> = {}): Plan =>
  Object.freeze({
    version: 0,
    now: '2026-10-17T09:06:00Z',
    admin: { username: 'shop' },
    targets: ['3'],
    tree: null,
    ...over,
  });

const vip = {
  id: 'vip',
  name: 'VIP',
  conditions: [{ field: 'note', op: 'contains', value: 'vip' }],
  ops: null,
};
const add10 = { kind: 'traffic.add', params: { gb: '10' } };

describe('signedAmount', () => {
  it.each([
    ['+10', false, '10'],
    ['10', false, '10'],
    ['-5', false, '-5'],
    [' 2.5 ', false, '2.5'],
    ['۱۰', false, '10'],
    ['-۵', false, '-5'],
    ['٢٫٥', false, '2.5'],
    ['-30', true, '-30'],
  ] as const)('reads %j', (text, whole, want) => {
    expect(signedAmount(text, whole)).toBe(want);
  });

  it.each([
    ['', false],
    ['0', false],
    ['-0', false],
    ['abc', false],
    ['1e3', false],
    ['10 GB', false],
    ['2.5', true],
    ['--5', false],
  ] as const)('refuses %j', (text, whole) => {
    expect(signedAmount(text, whole)).toBeNull();
  });
});

describe('filterCondition', () => {
  it('builds a condition from a filter and its value', () => {
    expect(filterCondition('name_contains', 'vip')).toEqual({
      field: 'username',
      op: 'contains',
      value: 'vip',
    });
    expect(filterCondition('used_more', '۵۰')).toEqual({ field: 'used_gb', op: 'gt', value: '50' });
    expect(filterCondition('status', 'disabled')).toEqual({
      field: 'status',
      op: 'in',
      list: ['disabled'],
    });
  });

  it('refuses an empty text, a number below 0 and an unknown status', () => {
    expect(filterCondition('note_contains', '  ')).toBeNull();
    expect(filterCondition('expires_within', '-3')).toBeNull();
    expect(filterCondition('offline_for', 'x')).toBeNull();
    expect(filterCondition('status', 'gone')).toBeNull();
  });
});

describe('changeTarget', () => {
  const name = { field: 'username', op: 'contains', value: 'x' };

  it('makes All users a category without conditions, once', () => {
    expect(changeTarget({ kind: 'all' }, 'All users', null)).toEqual({
      id: 'change-all',
      node: { id: 'change-all', name: 'All users', conditions: [], ops: null },
    });
    const all = { id: 'change-all', name: 'All users', conditions: [], ops: [add10] };
    expect(changeTarget({ kind: 'all' }, 'All users', [all])).toEqual({
      id: 'change-all',
      node: null,
    });
  });

  it('lands a filter the changes already made a category for on that category', () => {
    const made = { id: 'change-filter', name: 'Name has x', conditions: [name], ops: [add10] };
    expect(changeTarget({ kind: 'filter', condition: name }, 'n', [made])).toEqual({
      id: 'change-filter',
      node: null,
    });
    const other = { field: 'note', op: 'contains', value: 'x' };
    expect(changeTarget({ kind: 'filter', condition: other }, 'Note has x', [made])).toEqual({
      id: 'change-filter-2',
      node: { id: 'change-filter-2', name: 'Note has x', conditions: [other], ops: null },
    });
  });

  it('does not reuse a category of the operator with the same condition', () => {
    const theirs = { id: 'x-cat', name: 'X', conditions: [name], ops: null };
    expect(changeTarget({ kind: 'filter', condition: name }, 'n', [theirs]).id).toBe(
      'change-filter',
    );
  });

  it('picks users by key, and the same pick lands on the same category', () => {
    const t = changeTarget({ kind: 'users', keys: ['a', 'b'] }, 'Picked', null);
    expect(t.node).toEqual({
      id: 'change-users',
      name: 'Picked',
      conditions: [{ field: 'users', op: 'in', list: ['a', 'b'] }],
      ops: null,
    });
    const tree = t.node ? [{ ...t.node, ops: [add10] }] : [];
    expect(changeTarget({ kind: 'users', keys: ['a', 'b'] }, 'Picked', tree).node).toBeNull();
    expect(changeTarget({ kind: 'users', keys: ['b'] }, 'Picked', tree).id).toBe('change-users-2');
  });

  it('has no node of its own for a category', () => {
    expect(changeTarget({ kind: 'category', id: 'vip' }, 'VIP', [vip])).toEqual({
      id: 'vip',
      node: null,
    });
  });
});

describe('filterOf', () => {
  it('reads the filter back from a category the changes made', () => {
    const made = {
      id: 'change-filter',
      name: 'n',
      conditions: [{ field: 'status', op: 'in', list: ['disabled'] }],
      ops: null,
    };
    expect(filterOf(made)).toEqual({ key: 'status', value: 'disabled' });
    expect(filterOf({ ...made, id: 'vip' })).toBeNull();
    expect(filterOf({ ...made, conditions: [] })).toBeNull();
  });
});

describe('addChange', () => {
  it('adds an operation to a category', () => {
    const p = addChange('vip', null, add10)(plan({ tree: [vip] }));
    expect(p.tree).toEqual([{ ...vip, ops: [add10] }]);
  });

  it('adds a new node with the operation, once', () => {
    const { id, node } = changeTarget({ kind: 'all' }, 'All users', null);
    const edit = addChange(id, node, add10);
    const once = edit(plan());
    expect(once.tree).toEqual([{ ...node, ops: [add10] }]);
    expect(edit(once)).toBe(once);
  });
});

describe('removeChange', () => {
  const note = { kind: 'note.clear' };

  it('removes one operation of a category and keeps the category', () => {
    const p = removeChange('vip', 0)(plan({ tree: [{ ...vip, ops: [add10, note] }] }));
    expect(p.tree).toEqual([{ ...vip, ops: [note] }]);
    const empty = removeChange('vip', 0)(plan({ tree: [{ ...vip, ops: [note] }] }));
    expect(empty.tree).toEqual([{ ...vip, ops: null }]);
  });

  it('drops a node the changes made once its last operation goes', () => {
    const own = { id: 'change-users', name: 'Picked', conditions: [], ops: [note] };
    expect(removeChange('change-users', 0)(plan({ tree: [vip, own] })).tree).toEqual([vip]);
  });

  it('keeps a node the changes made when a category sits inside it', () => {
    const own = { id: 'change-all', name: 'All', conditions: [], ops: [note] };
    const inside = { ...vip, parentId: 'change-all' };
    const p = removeChange('change-all', 0)(plan({ tree: [own, inside] }));
    expect(p.tree).toEqual([{ ...own, ops: null }, inside]);
  });

  it('does nothing when the operation is gone already', () => {
    const p = plan({ tree: [{ ...vip, ops: [note] }] });
    expect(removeChange('vip', 3)(p)).toBe(p);
    expect(removeChange('nobody', 0)(p)).toBe(p);
  });
});

describe('applyTemplate', () => {
  const tpl: Template = {
    name: 'Renew',
    tree: [{ ...vip, ops: [add10] }],
    problems: { C10: [{ kind: 'skip' }] },
    resolutions: { C02: { option: 'suffix' } },
    resetSchedule: 'restart',
  };

  it('replaces categories, fixes and the reset schedule, and keeps the rest', () => {
    const before = plan({
      tree: [{ id: 'old', name: 'Old', conditions: null, ops: null }],
      resolutions: { C05: { option: 'skip' } },
      users: { bob: { exclude: true } },
    });
    expect(applyTemplate(tpl)(before)).toEqual({
      ...before,
      tree: tpl.tree,
      problems: tpl.problems,
      resolutions: tpl.resolutions,
      resetSchedule: 'restart',
    });
  });

  it('sends an empty tree and no fixes for a template without them', () => {
    const p = applyTemplate({ name: 'Empty', tree: null })(
      plan({ tree: [vip], resolutions: { C02: { option: 'skip' } }, resetSchedule: 'restart' }),
    );
    expect(p.tree).toEqual([]);
    expect(p).not.toHaveProperty('resolutions');
    expect(p).not.toHaveProperty('problems');
    expect(p).not.toHaveProperty('resetSchedule');
  });
});

describe('opFlags', () => {
  it('says which operations change configs or links', () => {
    expect(opFlags('cred.new')).toEqual({ config: true, link: false });
    expect(opFlags('name.prefix')).toEqual({ config: false, link: true });
    expect(opFlags('traffic.add')).toEqual({ config: false, link: false });
  });
});
