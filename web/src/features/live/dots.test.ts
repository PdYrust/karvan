import { describe, expect, it } from 'vitest';
import {
  batchDots,
  curvesDown,
  curvesFor,
  launch,
  maxDots,
  minBatchDots,
  pathOf,
  placeOf,
  pointOn,
  progressOf,
  sinkDots,
  sinkEnd,
  sinkLabel,
  staggerMs,
  travelMs,
} from './dots';

describe('batchDots', () => {
  it("sends a batch's share of the run in dots", () => {
    expect(batchDots(500, 1000, 0)).toBe(12);
    expect(batchDots(1000, 1000, 0)).toBe(maxDots);
  });

  it('sends a few dots for a batch too small for its share', () => {
    expect(batchDots(50, 100_000, 0)).toBe(minBatchDots);
  });

  it('never has more than the cap in flight', () => {
    expect(batchDots(1000, 1000, 20)).toBe(4);
    expect(batchDots(50, 100, maxDots)).toBe(0);
  });

  it('sends nothing without progress', () => {
    expect(batchDots(0, 100, 0)).toBe(0);
    expect(batchDots(10, 0, 0)).toBe(0);
  });
});

describe('sinkDots', () => {
  it('gives the sink its share of every dot sent over the run', () => {
    let sent = 0;
    let sunk = 0;
    for (let i = 0; i < 10; i++) {
      const n = sinkDots(sent, sunk, 4, 0.25);
      sent += 4;
      sunk += n;
    }
    expect(sunk).toBe(10);
  });

  it('sends none without skipped users, and never more than it has', () => {
    expect(sinkDots(0, 0, 5, 0)).toBe(0);
    expect(sinkDots(0, 0, 3, 1)).toBe(3);
  });
});

describe('dots in flight', () => {
  it('leave one after another', () => {
    const dots = launch(1000, 3, 'forward');
    expect(dots.map((d) => d.born)).toEqual([1000, 1000 + staggerMs, 1000 + 2 * staggerMs]);
  });

  it('ease in and out from 0 to 1', () => {
    const [d] = launch(0, 1, 'forward');
    if (!d) throw new Error('no dot');
    expect(progressOf(d, -100)).toBeLessThan(0);
    expect(progressOf(d, 0)).toBe(0);
    expect(progressOf(d, travelMs / 2)).toBeCloseTo(0.5);
    expect(progressOf(d, travelMs / 4)).toBeLessThan(0.25);
    expect(progressOf(d, travelMs)).toBe(1);
  });
});

describe('the scene curves', () => {
  it('run from the start edge to the end edge at mid height without a sink', () => {
    const { arc, sink } = curvesFor(400, 120, false);
    expect(sink).toBeNull();
    expect(pointOn(arc, 0)).toEqual({ x: 0, y: 60 });
    expect(pointOn(arc, 1)).toEqual({ x: 400, y: 60 });
    expect(pointOn(arc, 0.5).y).toBeLessThan(60);
  });

  it('branch down to the sink at the bottom middle', () => {
    const { sink } = curvesFor(400, 150, true);
    if (!sink) throw new Error('no sink');
    const end = pointOn(sink, 1);
    expect(end.x).toBe(200);
    expect(end.y).toBe(150 - sinkLabel);
  });

  it('write paths with a tenth of a pixel at most', () => {
    expect(pathOf(curvesFor(333, 100, false).arc)).toBe('M 0 50 C 99.9 26 233.1 26 333 50');
  });

  it('put a dot going back at the far end first, and a sink dot on the branch', () => {
    const { arc, sink } = curvesFor(400, 150, true);
    const [back] = launch(0, 1, 'back');
    const [sunk] = launch(0, 1, 'sink');
    if (!back || !sunk) throw new Error('no dot');
    expect(placeOf(back, 0, arc, sink)).toEqual({ x: 400, y: 75 });
    expect(placeOf(sunk, travelMs - 1, arc, sink)?.y).toBeGreaterThan(115);
    expect(placeOf(back, travelMs, arc, sink)).toBeNull();
    expect(placeOf(back, -1, arc, sink)).toBeNull();
  });
});

describe('the standing scene curves', () => {
  it('run straight down the middle from the top to the bottom', () => {
    const { arc, sink } = curvesDown(300, 120, false);
    expect(sink).toBeNull();
    expect(pointOn(arc, 0)).toEqual({ x: 150, y: 0 });
    expect(pointOn(arc, 0.5)).toEqual({ x: 150, y: 60 });
    expect(pointOn(arc, 1)).toEqual({ x: 150, y: 120 });
  });
  it('branch off the line toward the end edge, where the sink sits', () => {
    const { sink } = curvesDown(300, 120, true);
    if (!sink) throw new Error('no sink');
    expect(pointOn(sink, 0)).toEqual({ x: 150, y: 0 });
    expect(pointOn(sink, 1)).toEqual({ x: 300 * sinkEnd.x, y: 120 * sinkEnd.y });
  });
  it('send a dot going back up from the bottom', () => {
    const { arc } = curvesDown(300, 120, false);
    const [back] = launch(0, 1, 'back');
    if (!back) throw new Error('no dot');
    expect(placeOf(back, 0, arc, null)).toEqual({ x: 150, y: 120 });
  });
});
