/**
 * A migration's plan as the API sends it (api-migration.md section 6),
 * and the edits the wizard makes to it. Every edit is a pure function
 * from one plan to the next (D6), so a conflicting save can replay it on
 * the plan another tab wrote (D7).
 */
export interface Op {
  kind: string;
  params?: Record<string, string>;
}

export interface Condition {
  field: string;
  op: string;
  value?: string;
  list?: string[];
}

/** A category: its members are its parent's members that match all its conditions. */
export interface PlanNode {
  id: string;
  parentId?: string;
  name: string;
  conditions: Condition[] | null;
  ops: Op[] | null;
}

/** A fix for one conflict type, with its parameters (a prefix's text, a new name). */
export interface Resolution {
  option: string;
  params?: Record<string, string>;
}

export interface UserPlan {
  exclude?: boolean;
  resolutions?: Record<string, Resolution>;
  ops?: Op[];
}

export interface AdminPlan {
  /** Empty keeps the old panel's name. */
  username?: string;
  mode?: '' | 'create' | 'merge';
  /** PasarGuard; 0 or left out picks the default role. */
  roleId?: number;
}

export interface Plan {
  version: number;
  now: string;
  admin: AdminPlan;
  /** Where moved users go; null means the destination's defaults. */
  targets: string[] | null;
  tree: PlanNode[] | null;
  problems?: Record<string, Op[]>;
  /** The fix every user with a conflict type gets, unless they have their own. */
  resolutions?: Record<string, Resolution>;
  users?: Record<string, UserPlan>;
  resetSchedule?: string;
}

export type PlanEdit = (plan: Plan) => Plan;

/** A copy of map without key, or undefined once nothing is left in it. */
function without<V>(
  map: Record<string, V> | undefined,
  key: string,
): Record<string, V> | undefined {
  if (!map) return undefined;
  const rest = Object.fromEntries(Object.entries(map).filter(([k]) => k !== key));
  return Object.keys(rest).length ? rest : undefined;
}

function withKey<V>(
  map: Record<string, V> | undefined,
  key: string,
  value: V | undefined,
): Record<string, V> | undefined {
  return value === undefined ? without(map, key) : { ...map, [key]: value };
}

/** Renames the admin, merges it or picks its role. An empty name or role 0 goes back to the default. */
export function setAdmin(patch: AdminPlan): PlanEdit {
  return (p) => {
    const next: AdminPlan = { ...p.admin, ...patch };
    if (!next.username) delete next.username;
    if (!next.mode) delete next.mode;
    if (!next.roleId) delete next.roleId;
    return { ...p, admin: next };
  };
}

/** The targets moved users get; null for the destination's defaults. */
export function setTargets(ids: readonly string[] | null): PlanEdit {
  return (p) => ({ ...p, targets: ids ? [...ids] : null });
}

/** The fix every user with this conflict gets ("apply to all similar"); null for the default. */
export function setFix(type: string, fix: Resolution | null): PlanEdit {
  return (p) => ({ ...p, resolutions: withKey(p.resolutions, type, fix ?? undefined) });
}

/** One user's own fix for one conflict; null to follow the fix for all. */
export function setUserFix(key: string, type: string, fix: Resolution | null): PlanEdit {
  return (p) => {
    const user = p.users?.[key] ?? {};
    const resolutions = withKey(user.resolutions, type, fix ?? undefined);
    const next: UserPlan = { ...user };
    if (resolutions) next.resolutions = resolutions;
    else delete next.resolutions;
    return { ...p, users: withKey(p.users, key, Object.keys(next).length ? next : undefined) };
  };
}

/** The fix a user chose for a conflict, when they have their own. */
export function userFix(p: Plan, key: string, type: string): Resolution | undefined {
  return p.users?.[key]?.resolutions?.[type];
}

/** An id no category in tree has yet: base, then base-2, base-3 and on. */
export function newCategoryId(tree: readonly PlanNode[] | null, base: string): string {
  const taken = new Set((tree ?? []).map((n) => n.id));
  if (!taken.has(base)) return base;
  let n = 2;
  while (taken.has(`${base}-${n}`)) n++;
  return `${base}-${n}`;
}

/** Adds a category at the end of the tree, inside parentId when it is set. */
export function addCategory(
  id: string,
  node: { name: string; conditions: Condition[]; parentId?: string | undefined },
): PlanEdit {
  return (p) => {
    // Applied twice (a save that lands while the screen still shows it), it adds one.
    if (p.tree?.some((n) => n.id === id)) return p;
    const added: PlanNode = { id, name: node.name, conditions: node.conditions, ops: null };
    if (node.parentId) added.parentId = node.parentId;
    return { ...p, tree: [...(p.tree ?? []), added] };
  };
}

export function renameCategory(id: string, name: string): PlanEdit {
  return (p) => ({ ...p, tree: (p.tree ?? []).map((n) => (n.id === id ? { ...n, name } : n)) });
}

/** Removes a category and every category inside it. */
export function removeCategory(id: string): PlanEdit {
  return (p) => {
    const tree = p.tree ?? [];
    const gone = new Set([id]);
    // A child may come before its parent in the list, so repeat until nothing is added.
    for (let size = 0; size !== gone.size;) {
      size = gone.size;
      for (const n of tree) if (n.parentId && gone.has(n.parentId)) gone.add(n.id);
    }
    return { ...p, tree: tree.filter((n) => !gone.has(n.id)) };
  };
}

/** The categories directly inside parentId (undefined for the top), in tree order. */
export function childrenOf(tree: readonly PlanNode[] | null, parentId: string | undefined) {
  return (tree ?? []).filter((n) => (n.parentId || undefined) === parentId);
}

/** The categories from the top down to id, or [] when id is not in the tree. */
export function pathTo(tree: readonly PlanNode[] | null, id: string): PlanNode[] {
  const byId = new Map((tree ?? []).map((n) => [n.id, n]));
  const out: PlanNode[] = [];
  for (
    let n = byId.get(id);
    n && !out.includes(n);
    n = n.parentId ? byId.get(n.parentId) : undefined
  ) {
    out.unshift(n);
  }
  return out;
}
