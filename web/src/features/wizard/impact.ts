import type { Impact } from '../../lib/api/migrations';
import type { WizardKey } from './messages';

/** A count the impact bar shows: its words and its number. */
export type ImpactCount = [WizardKey, number];

/**
 * The counts of a plan. The preview splits new users from those an earlier
 * migration moved, and leaves those out of "Left out", so no user counts
 * twice (phase 4.1 D9); it also counts users who get protocols they lacked.
 */
export function impactCounts(impact: Impact | undefined, view: 'plan' | 'preview'): ImpactCount[] {
  const n = (k: keyof Impact) => impact?.[k] ?? 0;
  if (view === 'plan') {
    return [
      ['moving', n('moving')],
      ['skipped', n('skipped')],
      ['renamed', n('renamed')],
      ['configChanged', n('configChanged')],
    ];
  }
  return [
    ['added', n('moving')],
    ['alreadyMoved', n('alreadyMoved')],
    ['skipped', Math.max(0, n('skipped') - n('alreadyMoved'))],
    ['renamed', n('renamed')],
    ['configChanged', n('configChanged')],
    ['protocolsAdded', n('protocolsAdded')],
  ];
}

/** Whether an earlier migration moved the user (C16), so the run leaves it as it is (D9). */
export function isAlreadyMoved(row: {
  applied: readonly string[];
  conflicts: readonly { type: string }[];
}): boolean {
  return row.applied.includes('fix:C16:skip') || row.conflicts.some((c) => c.type === 'C16');
}
