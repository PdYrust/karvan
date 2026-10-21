import { describe, expect, it } from 'vitest';
import type { PanelStats } from '../../../lib/api/analytics';
import { panelWindow } from './window';

const minute = 60_000;

function stats(at: string, lastPoint?: number): PanelStats {
  return {
    panelId: 1,
    name: 'p',
    at,
    qps: 0,
    cacheHit: null,
    points: lastPoint === undefined ? [] : [{ t: lastPoint } as PanelStats['points'][number]],
  };
}

describe('panelWindow', () => {
  it('ends at the last reading and spans the range', () => {
    const at = '2026-10-21T12:00:00Z';
    const w = panelWindow({ panel: stats(at), step: 300 }, '24h');
    expect(w.end).toBe(Date.parse(at));
    expect(w.step).toBe(300_000);
    expect(w.domain).toEqual([Date.parse(at) - 24 * 60 * minute, Date.parse(at)]);
  });

  it('reads live as the last hour, since databases are read once a minute', () => {
    const at = '2026-10-21T12:00:00Z';
    const [from, to] = panelWindow({ panel: stats(at), step: 60 }, 'live').domain;
    expect(to - from).toBe(60 * minute);
  });

  it('falls back to the last minute it has when the reading has no time', () => {
    expect(panelWindow({ panel: stats('', 1_800_000_000), step: 60 }, '1h').end).toBe(
      1_800_000_000_000,
    );
  });
});
