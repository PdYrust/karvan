import type { Template } from '../../lib/plan/changes';
import type { Op, PlanNode } from '../../lib/plan/plan';

/** What a template does, one line each, in the order the engine applies it. */
export type TemplateLine =
  | { kind: 'category'; node: PlanNode; ops: Op[] }
  | { kind: 'problem'; type: string; ops: Op[] }
  | { kind: 'fix'; type: string; option: string }
  | { kind: 'reset'; schedule: 'estimate' | 'restart' };

const byKey = <V>(m: Record<string, V> | undefined) =>
  Object.entries(m ?? {}).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));

/**
 * The template as lines: categories with changes (a category that only
 * groups others changes nothing by itself), changes on problem groups,
 * fixes chosen for all users of a problem, and how reset times are kept.
 */
export function templateLines(t: Template): TemplateLine[] {
  const lines: TemplateLine[] = [];
  for (const node of t.tree ?? []) {
    if (node.ops?.length) lines.push({ kind: 'category', node, ops: node.ops });
  }
  for (const [type, ops] of byKey(t.problems)) {
    if (ops.length) lines.push({ kind: 'problem', type, ops });
  }
  for (const [type, r] of byKey(t.resolutions)) {
    lines.push({ kind: 'fix', type, option: r.option });
  }
  if (t.resetSchedule === 'estimate' || t.resetSchedule === 'restart') {
    lines.push({ kind: 'reset', schedule: t.resetSchedule });
  }
  return lines;
}
