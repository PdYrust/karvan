import { describe, expect, it } from 'vitest';
import { nextScrollHide, type ScrollHide } from './scroll-hide';

const run = (ys: number[], start: ScrollHide = { hidden: false, anchor: 0 }) =>
  ys.reduce((s, y) => nextScrollHide(s, y, { after: 56 }), start);

describe('nextScrollHide', () => {
  it('stays shown until the page is past the bar', () => {
    expect(run([10, 30, 56]).hidden).toBe(false);
  });
  it('hides on the way down past the bar', () => {
    expect(run([30, 80, 140]).hidden).toBe(true);
  });
  it('comes back on any move up', () => {
    expect(run([80, 300, 290]).hidden).toBe(false);
  });
  it('ignores jitters of a few px both ways', () => {
    const down = run([80, 300]);
    expect(nextScrollHide(down, 296, { after: 56 })).toBe(down);
    expect(run([80, 300, 296, 301, 297]).hidden).toBe(true);
    const up = run([80, 300, 200]);
    expect(run([203, 199, 204], up).hidden).toBe(false);
  });
  it('shows again at the top of the page', () => {
    expect(run([80, 600, 700, 0]).hidden).toBe(false);
  });
});
