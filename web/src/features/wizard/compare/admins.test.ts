import { describe, expect, it } from 'vitest';
import type { AdminInfo, Conflict } from '../../../lib/api/migrations';
import { adminConflicts, roleChoosers, sharedRole } from './admins';

const admin = (username: string, extra: Partial<AdminInfo> = {}): AdminInfo => ({
  username,
  isLogin: false,
  users: 4,
  moves: true,
  roleId: 3,
  roleFixed: false,
  newAdmin: { username, merged: false, roleId: 3 },
  userTargets: null,
  ...extra,
});

describe('roleChoosers', () => {
  it('keeps the admins that move and whose role can change', () => {
    const list = [
      admin('root', { isLogin: true }),
      admin('idle', { moves: false }),
      admin('north', { roleFixed: true }),
      admin('shop'),
    ];
    expect(roleChoosers(list).map((a) => a.username)).toEqual(['root', 'shop']);
  });
});

describe('sharedRole', () => {
  it('is the role every admin has, or null when they differ', () => {
    expect(sharedRole([admin('a'), admin('b')])).toBe(3);
    expect(sharedRole([admin('a'), admin('b', { roleId: 5 })])).toBeNull();
    expect(sharedRole([])).toBeNull();
  });
});

describe('adminConflicts', () => {
  const c = (type: string, who?: string): Conflict => ({
    type,
    severity: 'blocker',
    ...(who ? { detail: { admin: who } } : {}),
  });
  const all = [c('C01', 'shop'), c('C35', 'root'), c('C30'), c('C29'), c('C33', 'shop')];

  it("gives each admin its own conflicts, the login's admin those that name none, and leaves the job's", () => {
    expect(adminConflicts(all, admin('shop')).map((x) => x.type)).toEqual(['C01', 'C33']);
    expect(adminConflicts(all, admin('root', { isLogin: true })).map((x) => x.type)).toEqual([
      'C35',
      'C30',
    ]);
  });
});
