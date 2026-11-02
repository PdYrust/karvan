import { describe, expect, it } from 'vitest';
import type { BatchGroup } from '../../lib/api/batches';
import { oldPanels } from './old-panel-choice';

function group(edit: Partial<BatchGroup> = {}): BatchGroup {
  return {
    id: 1,
    latestId: 3,
    panel: { id: 2, name: 'pasarguard', kind: 'pasarguard' },
    source: { kind: 'pasarguard', origin: 'https://phonepanel.ir' },
    counts: { runs: 1, active: 1, inactive: 0, users: 3, verified: 0, closed: 0 },
    oldLinks: { path: '', hosts: [], port: 0 },
    rules: [],
    firstAt: '2026-11-01T00:00:00Z',
    lastAt: '2026-11-01T00:00:00Z',
    runs: [],
    ...edit,
  };
}

describe('the old-panel filter options', () => {
  it('lists each address once, in first-seen order', () => {
    expect(
      oldPanels([
        group(),
        group({ id: 2, source: { kind: 'marzban', origin: 'https://old.example.com' } }),
      ]),
    ).toEqual([
      { origin: 'https://phonepanel.ir', kind: 'pasarguard', runs: 1 },
      { origin: 'https://old.example.com', kind: 'marzban', runs: 1 },
    ]);
  });

  it('merges one address moved onto more than one new panel, adding its runs', () => {
    expect(
      oldPanels([
        group(),
        group({
          id: 2,
          panel: { id: 3, name: 'other', kind: 'marzban' },
          counts: { runs: 2, active: 2, inactive: 0, users: 5, verified: 0, closed: 0 },
        }),
      ]),
    ).toEqual([{ origin: 'https://phonepanel.ir', kind: 'pasarguard', runs: 3 }]);
  });

  it('lists nothing for no groups', () => {
    expect(oldPanels([])).toEqual([]);
  });
});
