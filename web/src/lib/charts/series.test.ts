import { describe, expect, it } from 'vitest';
import {
  areaPath,
  linePath,
  lonePoints,
  maxValue,
  nearestIndex,
  runs,
  spreadIndexes,
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
