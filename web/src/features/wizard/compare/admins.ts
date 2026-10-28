import type { AdminInfo, Conflict } from '../../../lib/api/migrations';

/** The admins "Same role for all" sets: those that move and whose role is not fixed by a merge. */
export function roleChoosers(admins: readonly AdminInfo[]): AdminInfo[] {
  return admins.filter((a) => a.moves && !a.roleFixed);
}

/** The role every one of these admins has, or null when they differ. */
export function sharedRole(admins: readonly AdminInfo[]): number | null {
  const [first, ...rest] = admins;
  return first && rest.every((a) => a.roleId === first.roleId) ? first.roleId : null;
}

/** The job conflicts about one admin (api-migration.md section 4); the rest are the whole job's. */
const adminTypes = new Set(['C01', 'C09', 'C30', 'C33', 'C34', 'C35', 'C37']);

/** One admin's conflicts: those naming it, and for the login's admin those naming none. */
export function adminConflicts(conflicts: readonly Conflict[], admin: AdminInfo): Conflict[] {
  return conflicts.filter(
    (c) =>
      adminTypes.has(c.type) &&
      (c.detail?.admin ? c.detail.admin === admin.username : admin.isLogin),
  );
}
