import type { StatusTone } from '../../components/feedback/StatusBadge';
import { panelSearchNames } from '../../components/panels/PanelLogo';
import type { Batch, BatchGroup, InactiveReason } from '../../lib/api/batches';
import { matchesTerm } from '../../lib/lists/choices';
import type { BatchesKey } from './messages';

const reasonTones: Readonly<Record<InactiveReason, StatusTone>> = {
  restoring: 'active',
  rolled_back: 'neutral',
  restore_failed: 'danger',
};

/** How a batch's state reads: its tone, its word and, when inactive, why. */
export function batchLook(b: Pick<Batch, 'state' | 'inactiveReason'>): {
  tone: StatusTone;
  word: BatchesKey;
  reason?: BatchesKey;
} {
  if (b.state === 'active' || !b.inactiveReason) return { tone: 'success', word: 'stateActive' };
  return {
    tone: reasonTones[b.inactiveReason],
    word: 'stateInactive',
    reason: `reason_${b.inactiveReason}`,
  };
}

/**
 * What the operator may do besides renaming: delete an inactive batch
 * once no restore runs, and make one active again after a failed restore.
 */
export function batchActions(b: Pick<Batch, 'state' | 'inactiveReason'>) {
  const inactive = b.state === 'inactive';
  return {
    remove: inactive && b.inactiveReason !== 'restoring',
    reactivate: inactive && b.inactiveReason === 'restore_failed',
  };
}

/** The search over a page of batches: name, admins, old address and both panels. */
export function matchesBatch(b: Batch, term: string): boolean {
  return matchesTerm(term, [
    b.name,
    ...b.admins.flatMap((a) => [a.source, a.dest]),
    b.source.origin,
    b.panel.name,
    panelSearchNames[b.source.kind],
    b.panel.kind ? panelSearchNames[b.panel.kind] : undefined,
  ]);
}

/** The search over a page of old panels: the address, both panels, the rules and the runs shown. */
export function matchesGroup(g: BatchGroup, term: string): boolean {
  return (
    matchesTerm(term, [
      g.source.origin,
      g.panel.name,
      panelSearchNames[g.source.kind],
      g.panel.kind ? panelSearchNames[g.panel.kind] : undefined,
      ...g.rules.map((r) => r.name),
    ]) || g.runs.some((b) => matchesBatch(b, term))
  );
}

export const batchPath = (id: number) => `/batches/${String(id)}`;

/**
 * Batches newest first by the date they were made, as the table's Made column
 * shows. A stable sort, so batches made at the same time keep their page order.
 */
export function newestBatches(batches: readonly Batch[]): Batch[] {
  return [...batches].sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
}
