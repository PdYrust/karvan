import { describe, expect, it } from 'vitest';
import { changeOf, latestOf } from './change';
describe('changeOf', () => {
  it('compares the latest value with the first one Karvan holds', () => {
    expect(changeOf([100, null, 110])).toEqual({ by: 0.1, up: true, good: true });
    const down = changeOf([200, 150]);
    expect(down?.up).toBe(false);
    expect(down?.by).toBeCloseTo(0.25);
  });
  it('says a rise is bad where it is (CPU, memory)', () => {
    expect(changeOf([1, 2], { risingIsBad: true })).toMatchObject({ up: true, good: false });
    expect(changeOf([2, 1], { risingIsBad: true })).toMatchObject({ up: false, good: true });
  });
  it('moves a share by points', () => {
    const c = changeOf([0, 0.25], { points: true, risingIsBad: true });
    expect(c?.by).toBeCloseTo(0.25);
    expect(c?.good).toBe(false);
  });
  it('has nothing to say without two values, from zero, or with no change', () => {
    expect(changeOf([])).toBeUndefined();
    expect(changeOf([null, 5])).toBeUndefined();
    expect(changeOf([0, 5])).toBeUndefined();
    expect(changeOf([5, 5])).toBeUndefined();
  });
});
describe('latestOf', () => {
  it('skips missing values at the end', () => {
    expect(latestOf([1, 2, null])).toBe(2);
    expect(latestOf([])).toBeUndefined();
  });
});
