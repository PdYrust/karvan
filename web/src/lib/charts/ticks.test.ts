import { describe, expect, it } from 'vitest';
import { axisTicks, byteTicks, niceStep, niceTicks } from './ticks';

describe('axisTicks', () => {
  it('keeps shares on fixed quarters', () => {
    expect(axisTicks('percent', 0.03)).toEqual([0, 0.25, 0.5, 0.75, 1]);
    expect(axisTicks('percent', 1.6)).toEqual([0, 0.5, 1, 1.5, 2]);
  });

  it('rounds bytes and rates in their unit, counts in tens', () => {
    expect(axisTicks('rate', 3000)).toEqual([0, 1024, 2048, 3072]);
    expect(axisTicks('bytes', 300)).toEqual([0, 100, 200, 300]);
    expect(axisTicks('count', 76)).toEqual([0, 20, 40, 60, 80]);
  });
  it('keeps counts whole, even with nothing counted yet', () => {
    expect(axisTicks('count', 0)).toEqual([0, 1]);
    expect(axisTicks('count', 3)).toEqual([0, 1, 2, 3]);
  });
});

describe('niceStep', () => {
  it('rounds up to 1, 2, 2.5 or 5 times a power of ten', () => {
    expect([0.7, 1, 1.3, 2.2, 3, 7, 23, 260].map(niceStep)).toEqual([1, 1, 2, 2.5, 5, 10, 25, 500]);
    expect(niceStep(0.03)).toBe(0.05);
  });

  it('falls back to 1 for nothing to scale', () => {
    expect(niceStep(0)).toBe(1);
    expect(niceStep(-4)).toBe(1);
    expect(niceStep(Number.NaN)).toBe(1);
  });
});

describe('niceTicks', () => {
  it('starts at zero and ends at or above the maximum', () => {
    expect(niceTicks(100)).toEqual([0, 25, 50, 75, 100]);
    expect(niceTicks(6.3)).toEqual([0, 2, 4, 6, 8]);
    expect(niceTicks(0.3)).toEqual([0, 0.1, 0.2, 0.3]);
  });

  it('gives an empty chart a unit axis', () => {
    expect(niceTicks(0)).toEqual([0, 0.25, 0.5, 0.75, 1]);
  });
});

describe('byteTicks', () => {
  const gb = 1024 ** 3;
  const mb = 1024 ** 2;

  it('is round in the unit the top value is shown in', () => {
    expect(byteTicks(11.6 * gb)).toEqual([0, 5 * gb, 10 * gb, 15 * gb]);
    expect(byteTicks(900 * mb)).toEqual([0, 250 * mb, 500 * mb, 750 * mb, 1000 * mb]);
  });

  it('keeps small numbers in bytes', () => {
    expect(byteTicks(300)).toEqual([0, 100, 200, 300]);
  });
});
