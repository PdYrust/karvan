import type { FixOption, ProblemGroup, Severity, UserRow } from '../../../lib/api/migrations';
import type { Plan, Resolution } from '../../../lib/plan/plan';
import type { ConflictKey } from './conflict-messages';

/** How serious each per-user conflict is before a fix (api-migration.md section 4). */
const severities: Readonly<Record<string, Severity>> = {
  C02: 'blocker',
  C03: 'blocker',
  C04: 'blocker',
  C05: 'blocker',
  C06: 'blocker',
  C07: 'blocker',
  C11: 'blocker',
  C12: 'blocker',
  C19: 'blocker',
  C24: 'blocker',
  C10: 'warning',
  C16b: 'warning',
  C18: 'warning',
  C28: 'warning',
  C31: 'warning',
  C32: 'warning',
  C16: 'info',
  C26: 'info',
};

/** Types whose first fix applies when nothing is chosen, unless it changes configs. */
const withDefault = new Set(
  Object.keys(severities).filter((t) => !['C11', 'C12', 'C19'].includes(t)),
);

export function severityOf(type: string): Severity {
  return severities[type] ?? 'warning';
}

const rank: Record<Severity, number> = { blocker: 0, warning: 1, info: 2 };

/**
 * Groups most relevant first: the ones that still need a fix, then the most
 * serious kind, then the largest.
 */
export function byRelevance(groups: readonly ProblemGroup[]): ProblemGroup[] {
  return [...groups].sort(
    (a, b) =>
      Number(b.open > 0) - Number(a.open > 0) ||
      rank[severityOf(a.type)] - rank[severityOf(b.type)] ||
      b.count - a.count ||
      a.type.localeCompare(b.type),
  );
}

/** The fix Karvan uses for a type when nothing is chosen, if there is one. */
export function defaultFix(group: ProblemGroup): FixOption | undefined {
  const first = group.options[0];
  return withDefault.has(group.type) && first && !first.changesConfig ? first : undefined;
}

/** The fix every user of a group gets: the one chosen for all, else the default. */
export function fixForAll(plan: Plan, group: ProblemGroup): Resolution | undefined {
  const chosen = plan.resolutions?.[group.type];
  if (chosen) return chosen;
  const d = defaultFix(group);
  return d ? { option: d.id } : undefined;
}

/** The fix one user got for a type in the last build, from the row's conflicts or what was applied. */
export function appliedFix(row: UserRow, type: string): string | undefined {
  const open = row.conflicts.find((c) => c.type === type)?.fix;
  if (open) return open;
  const prefix = `fix:${type}:`;
  return row.applied.find((a) => a.startsWith(prefix))?.slice(prefix.length);
}

const typeKeys = new Set(Object.keys(severities));
const fixes = new Set([
  'suffix',
  'prefix',
  'rename',
  'sanitize',
  'skip',
  'new_credential',
  'keep',
  'accept',
  'change_method',
  'drop_protocol',
  'move_again',
  'normalize',
  'adopt',
  'fit',
  'remove',
]);

/** The words of a per-user conflict type; an unknown one names its code. */
export function typeWords(type: string): ConflictKey {
  return typeKeys.has(type) ? (`type_${type}` as ConflictKey) : 'type_unknown';
}

export function fixWords(fix: string): ConflictKey {
  return fixes.has(fix) ? (`fix_${fix}` as ConflictKey) : 'fix_unknown';
}

/** Fixes that need a value: the text of a prefix or suffix, a method. A new name is per user. */
export const fixParam: Readonly<Record<string, string>> = {
  prefix: 'text',
  suffix: 'text',
  change_method: 'method',
  rename: 'username',
};

/** Fixes one user can get on their own: those without a value, a numbered suffix, a new name. */
export function userFixes(group: ProblemGroup): FixOption[] {
  return group.options.filter((o) => !['prefix', 'change_method'].includes(o.id));
}
