import { describe, expect, it } from 'vitest';
import { impactCounts, isAlreadyMoved } from './impact';

describe('impactCounts', () => {
  it('shows what moves, what is left out, renames and config changes before the preview', () => {
    expect(impactCounts({ users: 9, moving: 5, skipped: 4, renamed: 1 }, 'plan')).toEqual([
      ['moving', 5],
      ['skipped', 4],
      ['renamed', 1],
      ['configChanged', 0],
    ]);
  });
  it('splits added and already moved users in the preview, so no user counts twice (D9)', () => {
    const impact = { users: 30, moving: 10, skipped: 20, alreadyMoved: 12, protocolsAdded: 3 };
    expect(impactCounts(impact, 'preview')).toEqual([
      ['added', 10],
      ['alreadyMoved', 12],
      ['skipped', 8],
      ['renamed', 0],
      ['configChanged', 0],
      ['protocolsAdded', 3],
    ]);
  });
  it('reads a missing impact as zeros', () => {
    expect(impactCounts(undefined, 'preview').every(([, n]) => n === 0)).toBe(true);
  });
});

describe('isAlreadyMoved', () => {
  it('finds users an earlier migration moved, by their C16 fix or open C16', () => {
    expect(isAlreadyMoved({ applied: ['added:wireguard', 'fix:C16:skip'], conflicts: [] })).toBe(
      true,
    );
    expect(isAlreadyMoved({ applied: [], conflicts: [{ type: 'C16' }] })).toBe(true);
    expect(isAlreadyMoved({ applied: ['excluded'], conflicts: [{ type: 'C16b' }] })).toBe(false);
  });
});
