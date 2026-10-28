import { describe, expect, it } from 'vitest';
import {
  addCategory,
  childrenOf,
  newCategoryId,
  pathTo,
  removeCategory,
  renameCategory,
  setAdmin,
  setFix,
  setRoleForAll,
  setUserFix,
  userFix,
  type Plan,
} from './plan';

const fresh = (): Plan => ({
  version: 0,
  now: '2026-10-17T09:06:00Z',
  targets: null,
  tree: null,
});

/** A plan frozen deep, so an edit that changes its input throws. */
function frozen(p: Plan): Plan {
  const freeze = (v: unknown): unknown => {
    if (v && typeof v === 'object') {
      Object.values(v).forEach(freeze);
      Object.freeze(v);
    }
    return v;
  };
  return freeze(structuredClone(p)) as Plan;
}

describe('setAdmin', () => {
  it('renames one admin and keeps its role and the other admins', () => {
    const p = setAdmin('shop', { username: 'shop_2', mode: 'create' })(
      frozen({ ...fresh(), admins: { shop: { roleId: 3 }, north: { mode: 'merge' } } }),
    );
    expect(p.admins).toEqual({
      shop: { username: 'shop_2', mode: 'create', roleId: 3 },
      north: { mode: 'merge' },
    });
  });

  it('drops a field set back to its default, and an entry left empty', () => {
    const p = setAdmin('shop', { username: '', roleId: 0 })(
      frozen({ ...fresh(), admins: { shop: { username: 'x', mode: 'merge', roleId: 2 } } }),
    );
    expect(p.admins).toEqual({ shop: { mode: 'merge' } });
    const q = setAdmin('shop', { roleId: 0 })(
      frozen({ ...fresh(), admins: { shop: { roleId: 2 } } }),
    );
    expect(q.admins).toBeUndefined();
  });

  it('never sends the old admin field, which the server would read over admins', () => {
    const p = setAdmin('shop', { roleId: 4 })(frozen({ ...fresh(), admin: { roleId: 2 } }));
    expect(p).not.toHaveProperty('admin');
    expect(p.admins).toEqual({ shop: { roleId: 4 } });
  });
});

describe('setRoleForAll', () => {
  it('gives every named admin the role and leaves the rest as they are', () => {
    const p = setRoleForAll(
      ['shop', 'east'],
      5,
    )(
      frozen({
        ...fresh(),
        admins: { shop: { username: 'shop_2', roleId: 3 }, north: { mode: 'merge' } },
      }),
    );
    expect(p.admins).toEqual({
      shop: { username: 'shop_2', roleId: 5 },
      east: { roleId: 5 },
      north: { mode: 'merge' },
    });
  });
});

describe('setFix', () => {
  it('chooses a fix for every user with the conflict', () => {
    const p = setFix('C02', { option: 'prefix', params: { text: 'old_' } })(frozen(fresh()));
    expect(p.resolutions).toEqual({ C02: { option: 'prefix', params: { text: 'old_' } } });
  });

  it('goes back to the default and leaves no empty map', () => {
    const p = setFix('C02', null)(frozen({ ...fresh(), resolutions: { C02: { option: 'skip' } } }));
    expect(p.resolutions).toBeUndefined();
  });
});

describe('setUserFix', () => {
  it("sets one user's fix next to the user's other choices", () => {
    const start = { ...fresh(), users: { bob: { ops: [{ kind: 'note.clear' }] } } };
    const p = setUserFix('bob', 'C02', { option: 'rename', params: { username: 'bob2' } })(
      frozen(start),
    );
    expect(p.users?.bob).toEqual({
      ops: [{ kind: 'note.clear' }],
      resolutions: { C02: { option: 'rename', params: { username: 'bob2' } } },
    });
    expect(userFix(p, 'bob', 'C02')).toEqual({ option: 'rename', params: { username: 'bob2' } });
  });

  it('removes the choice, then the user, then the map once nothing is left', () => {
    const start = { ...fresh(), users: { bob: { resolutions: { C02: { option: 'skip' } } } } };
    const p = setUserFix('bob', 'C02', null)(frozen(start));
    expect(p.users).toBeUndefined();
    expect(userFix(p, 'bob', 'C02')).toBeUndefined();
  });
});

describe('categories', () => {
  const tree = () =>
    frozen({
      ...fresh(),
      tree: [
        { id: 'vip', name: 'VIP', conditions: [], ops: null },
        { id: 'vip-old', parentId: 'vip', name: 'Old VIP', conditions: [], ops: null },
        { id: 'deep', parentId: 'vip-old', name: 'Deep', conditions: [], ops: null },
        { id: 'trial', name: 'Trial', conditions: [], ops: null },
      ],
    });

  it('adds a category at the top or inside another', () => {
    const cond = [{ field: 'expired', op: 'is', value: 'true' }];
    const top = addCategory('expired', { name: 'Expired', conditions: cond })(frozen(fresh()));
    expect(top.tree).toEqual([{ id: 'expired', name: 'Expired', conditions: cond, ops: null }]);
    const inside = addCategory('expired', { name: 'Expired', conditions: cond, parentId: 'vip' })(
      tree(),
    );
    expect(inside.tree?.at(-1)).toEqual({
      id: 'expired',
      parentId: 'vip',
      name: 'Expired',
      conditions: cond,
      ops: null,
    });
  });

  it('adds a category once when the same edit runs twice', () => {
    const add = addCategory('vip', { name: 'VIP again', conditions: [] });
    expect(add(tree()).tree).toHaveLength(4);
  });

  it('makes an id no other category has', () => {
    expect(newCategoryId(null, 'vip')).toBe('vip');
    expect(newCategoryId(tree().tree, 'vip')).toBe('vip-2');
  });

  it('renames a category', () => {
    expect(renameCategory('trial', 'Free trial')(tree()).tree?.at(-1)?.name).toBe('Free trial');
  });

  it('removes a category with everything inside it', () => {
    expect(removeCategory('vip')(tree()).tree?.map((n) => n.id)).toEqual(['trial']);
    const childFirst = frozen({ ...fresh(), tree: [...(tree().tree ?? [])].reverse() });
    expect(removeCategory('vip')(childFirst).tree?.map((n) => n.id)).toEqual(['trial']);
  });

  it('lists the categories inside one, in tree order', () => {
    expect(childrenOf(tree().tree, undefined).map((n) => n.id)).toEqual(['vip', 'trial']);
    expect(childrenOf(tree().tree, 'vip').map((n) => n.id)).toEqual(['vip-old']);
  });

  it('finds the path from the top to a category, for the breadcrumb', () => {
    expect(pathTo(tree().tree, 'deep').map((n) => n.id)).toEqual(['vip', 'vip-old', 'deep']);
    expect(pathTo(tree().tree, 'gone')).toEqual([]);
  });
});
