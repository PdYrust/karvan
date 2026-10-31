import { latinDigits } from '../format/number';
import {
  newCategoryId,
  removeCategory,
  type Condition,
  type Op,
  type PlanEdit,
  type PlanNode,
} from './plan';

/**
 * The Changes step's edits: an
 * operation lands on a category's ops. All users, a filter and picked users
 * get a category of their own, with an id that starts with "change-", so
 * removing its last operation removes it too.
 */
export type Scope =
  | { kind: 'all' }
  | { kind: 'category'; id: string }
  | { kind: 'filter'; condition: Condition }
  | { kind: 'users'; keys: readonly string[] };

const own = 'change-';

/** "+10", "-5" or "۱۰" as the API's signed number ("10", "-5"); null when it is not one or is 0. */
export function signedAmount(text: string, whole: boolean): string | null {
  const t = latinDigits(text);
  const shape = whole ? /^[+-]?\d+$/ : /^[+-]?(\d+(\.\d*)?|\.\d+)$/;
  if (!shape.test(t)) return null;
  const n = Number(t);
  if (!Number.isFinite(n) || n === 0) return null;
  return String(n);
}

/** A number of zero or more as the API reads it; null otherwise. */
function amount(text: string): string | null {
  const t = latinDigits(text);
  return /^(\d+(\.\d*)?|\.\d+)$/.test(t) ? String(Number(t)) : null;
}

export const statuses = ['active', 'disabled', 'limited', 'expired', 'on_hold'] as const;

/** The filters the Changes step offers, as conditions. */
export const filters = {
  name_contains: { field: 'username', op: 'contains', value: 'text' },
  name_prefix: { field: 'username', op: 'prefix', value: 'text' },
  name_suffix: { field: 'username', op: 'suffix', value: 'text' },
  note_contains: { field: 'note', op: 'contains', value: 'text' },
  used_more: { field: 'used_gb', op: 'gt', value: 'number' },
  used_less: { field: 'used_gb', op: 'lt', value: 'number' },
  expires_within: { field: 'expires_within_days', op: 'lte', value: 'number' },
  offline_for: { field: 'not_online_days', op: 'gte', value: 'number' },
  status: { field: 'status', op: 'in', value: 'status' },
} as const;

export type FilterKey = keyof typeof filters;

/** The condition a filter with this value makes, or null when the value does not fit it. */
export function filterCondition(key: FilterKey, value: string): Condition | null {
  const f = filters[key];
  if (f.value === 'status') {
    return (statuses as readonly string[]).includes(value)
      ? { field: f.field, op: f.op, list: [value] }
      : null;
  }
  const v = f.value === 'number' ? amount(value) : value.trim() || null;
  return v === null ? null : { field: f.field, op: f.op, value: v };
}

const sameCondition = (a: Condition | undefined, b: Condition) =>
  JSON.stringify(a) === JSON.stringify(b);

/** A category the changes made whose only condition is c. */
function madeFor(tree: readonly PlanNode[] | null, c: Condition): PlanNode | undefined {
  return tree?.find(
    (n) => n.id.startsWith(own) && n.conditions?.length === 1 && sameCondition(n.conditions[0], c),
  );
}

/**
 * Where a change for scope lands: the category's id, and the category to
 * add first when the plan has none for it yet. A filter or a pick the
 * changes already made a category for lands there again.
 */
export function changeTarget(
  scope: Scope,
  name: string,
  tree: readonly PlanNode[] | null,
): { id: string; node: PlanNode | null } {
  if (scope.kind === 'category') return { id: scope.id, node: null };
  const condition: Condition | null =
    scope.kind === 'filter'
      ? scope.condition
      : scope.kind === 'users'
        ? { field: 'users', op: 'in', list: [...scope.keys] }
        : null;
  const there = condition ? madeFor(tree, condition) : tree?.find((n) => n.id === `${own}all`);
  if (there) return { id: there.id, node: null };
  const id = condition ? newCategoryId(tree, `${own}${scope.kind}`) : `${own}all`;
  return { id, node: { id, name, conditions: condition ? [condition] : [], ops: null } };
}

/** Whether the changes made this category (All users, a filter or a pick). */
export const isChangeNode = (n: PlanNode) => n.id.startsWith(own);

/** The filter a category the changes made stands for, read back from its condition. */
export function filterOf(n: PlanNode): { key: FilterKey; value: string } | null {
  const c = n.conditions?.[0];
  if (!isChangeNode(n) || n.conditions?.length !== 1 || !c) return null;
  const key = (Object.keys(filters) as FilterKey[]).find(
    (k) => filters[k].field === c.field && filters[k].op === c.op,
  );
  return key ? { key, value: c.list?.[0] ?? c.value ?? '' } : null;
}

/** Puts op on the category id; node, when given, is added first with op as its only operation. */
export function addChange(id: string, node: PlanNode | null, op: Op): PlanEdit {
  return (p) => {
    const tree = p.tree ?? [];
    if (node) {
      // Applied twice (a save that lands while the screen still shows it), it adds one.
      if (tree.some((n) => n.id === node.id)) return p;
      return { ...p, tree: [...tree, { ...node, ops: [op] }] };
    }
    return {
      ...p,
      tree: tree.map((n) => (n.id === id ? { ...n, ops: [...(n.ops ?? []), op] } : n)),
    };
  };
}

/** Removes the operation at index of category id, and the category when the changes made it and it is empty now. */
export function removeChange(id: string, index: number): PlanEdit {
  return (p) => {
    const node = p.tree?.find((n) => n.id === id);
    if (!node?.ops || index >= node.ops.length) return p;
    const ops = node.ops.filter((_, i) => i !== index);
    const alone = !p.tree?.some((n) => n.parentId === id);
    if (!ops.length && id.startsWith(own) && alone) return removeCategory(id)(p);
    const next: PlanNode = { ...node, ops: ops.length ? ops : null };
    return { ...p, tree: (p.tree ?? []).map((n) => (n.id === id ? next : n)) };
  };
}

/** Whether an operation changes users' configs (new credentials) or links (a new name). */
export function opFlags(kind: string): { config: boolean; link: boolean } {
  return { config: kind === 'cred.new', link: kind.startsWith('name.') };
}
