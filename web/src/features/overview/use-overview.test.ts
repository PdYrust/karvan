import { describe, expect, it } from 'vitest';
import type { Panel } from '../../lib/api/panels';
import { homePanels, homeRowsMax, recentJobsLimit } from './use-overview';

const panel = (id: number) => ({ id, name: `p${String(id)}` }) as Panel;

describe('overview columns', () => {
  it('show three rows each', () => {
    expect(homeRowsMax).toBe(3);
    expect(recentJobsLimit).toBe(3);
  });

  it('keep the last three registered panels in list order', () => {
    const shown = homePanels([1, 0, 2, 3, 4, 5].map(panel));
    expect(shown.map((p) => p.id)).toEqual([3, 4, 5]);
  });

  it('leave out folders a scan could not use', () => {
    expect(homePanels([0, 1].map(panel)).map((p) => p.id)).toEqual([1]);
  });
});
