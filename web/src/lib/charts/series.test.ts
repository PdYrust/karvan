import { describe, expect, it } from 'vitest';
import {
  areaPath,
  fitDomain,
  gapLimit,
  linePath,
  lonePoints,
  maxValue,
  nearestIndex,
  runs,
  splitEdges,
  spreadIndexes,
  sumValues,
} from './series';

describe('runs', () => {
  it('breaks at a missing value and at a long gap', () => {
    const times = [0, 5, 10, 15, 40, 45, 50];
    const values = [1, 2, null, 4, 5, 6, 7];
    expect(runs(times, values, 10)).toEqual([
      [
        [0, 1],
        [5, 2],
      ],
      [[15, 4]],
      [
        [40, 5],
        [45, 6],
        [50, 7],
      ],
    ]);
  });

  it('skips values that are not numbers', () => {
    expect(runs([0, 1], [Number.NaN, undefined], 5)).toEqual([]);
  });

  it('takes a limit for each point where the points are not evenly spaced', () => {
    // Minutes 60 apart, then samples 5 apart; 0 before 130 breaks the line there.
    const times = [0, 60, 120, 130, 135];
    expect(
      runs(times, [1, 2, 3, 4, 5], gapLimit(5, [60, 60, 60, 0, 5])).map((r) => r.length),
    ).toEqual([3, 2]);
    expect(gapLimit(5)).toBe(9.5);
  });
});

describe('fitDomain', () => {
  // A panel read every 60 s: buckets start on the minute, the window ends at the last reading.
  const parts = runs([35, 95, 155, 3515, 3575], [1, 2, 3, 4, 5], 114);
  it('moves each edge onto a point that is only off the window grid', () => {
    expect(fitDomain(runs([35, 95, 155, 215], [1, 2, 3, 4], 114), [0, 240])).toEqual([35, 215]);
  });
  it('keeps an edge where no point covers a longer wait', () => {
    expect(fitDomain(parts, [-200, 3600])).toEqual([-200, 3575]);
    expect(fitDomain(runs([35, 95], [1, 2], 114), [0, 3600])).toEqual([35, 3600]);
  });
  it('keeps the window with a lone point, nothing drawn or points outside it', () => {
    expect(fitDomain(runs([35], [1], 114), [0, 100])).toEqual([0, 100]);
    expect(fitDomain([], [0, 100])).toEqual([0, 100]);
    expect(fitDomain(runs([-10, 50, 110], [1, 2, 3], 114), [0, 100])).toEqual([0, 100]);
  });
});
describe('paths', () => {
  const x = (t: number) => t * 10;
  const y = (v: number) => 100 - v;
  const parts = runs([0, 1, 2, 5, 9], [10, 20, 30, 40, 50], 1);

  it('moves to the start of each run and draws only runs of two or more', () => {
    expect(linePath(parts, x, y)).toBe('M0 90L10 80L20 70');
  });

  it('closes the area of each run down to the base', () => {
    expect(areaPath(parts, x, y, 100)).toBe('M0 100L0 90L10 80L20 70L20 100Z');
  });

  it('gives lone points their own dot', () => {
    expect(lonePoints(parts)).toEqual([
      [5, 40],
      [9, 50],
    ]);
  });
});

describe('nearestIndex', () => {
  const times = [0, 10, 20, 30];

  it('finds the closest time', () => {
    expect(nearestIndex(times, 14)).toBe(1);
    expect(nearestIndex(times, 16)).toBe(2);
    expect(nearestIndex(times, -5)).toBe(0);
    expect(nearestIndex(times, 99)).toBe(3);
    expect(nearestIndex([7], 0)).toBe(0);
  });

  it('is -1 without times', () => {
    expect(nearestIndex([], 3)).toBe(-1);
  });
});

describe('maxValue and spreadIndexes', () => {
  it('ignores missing values', () => {
    expect(
      maxValue([
        [1, null, 3],
        [undefined, 2],
      ]),
    ).toBe(3);
    expect(maxValue([[]])).toBe(0);
  });

  it('spreads indexes from first to last', () => {
    expect(spreadIndexes(3, 5)).toEqual([0, 1, 2]);
    expect(spreadIndexes(101, 5)).toEqual([0, 25, 50, 75, 100]);
    expect(spreadIndexes(10, 1)).toEqual([9]);
  });
});
describe('splitEdges', () => {
  it('dashes the first and last segment and keeps the middle solid', () => {
    const run = [1, 2, 3, 4].map((t) => [t, t] as const);
    expect(splitEdges([run])).toEqual({
      solid: [
        [
          [2, 2],
          [3, 3],
        ],
      ],
      dashed: [
        [
          [1, 1],
          [2, 2],
        ],
        [
          [3, 3],
          [4, 4],
        ],
      ],
    });
  });
  it('leaves only dashes when a series has two points', () => {
    const { solid, dashed } = splitEdges([
      [
        [1, 1],
        [2, 2],
      ],
    ]);
    expect(solid).toEqual([]);
    expect(dashed).toHaveLength(1);
  });
  it('has nothing to split without segments', () => {
    expect(splitEdges([[[1, 1]]])).toEqual({ solid: [], dashed: [] });
    expect(splitEdges([])).toEqual({ solid: [], dashed: [] });
  });
});
describe('sumValues', () => {
  it('adds what is there and keeps missing times missing', () => {
    expect(
      sumValues([
        [1, null, null],
        [2, 3, undefined],
      ]),
    ).toEqual([3, 3, null]);
  });
});
