import { describe, expect, it } from 'vitest';
import { activeFilters, matchesTerm, sortRows } from './choices';

describe('activeFilters', () => {
  it('counts filters off their all value and never a sort', () => {
    expect(
      activeFilters([
        { value: 'done', all: 'all' },
        { value: 'all', all: 'all' },
        { value: 'name' },
      ]),
    ).toBe(1);
    expect(activeFilters([])).toBe(0);
  });
});

describe('sortRows', () => {
  const rows = [
    { n: 'beta', t: '2026-10-01T00:00:00Z', s: 5 },
    { n: 'Alpha', t: '2026-10-03T00:00:00Z', s: 1 },
    { n: 'alpha 10', t: '2026-10-02T00:00:00Z', s: 9 },
    { n: 'alpha 9', t: 'not a time', s: 3 },
  ];
  const keys = { name: (r: (typeof rows)[number]) => r.n, time: (r: (typeof rows)[number]) => r.t };
  it('sorts by name in the page language, numbers as numbers', () => {
    expect(sortRows(rows, 'name', keys, 'en').map((r) => r.n)).toEqual([
      'Alpha',
      'alpha 9',
      'alpha 10',
      'beta',
    ]);
  });
  it('puts the newest first, a time it cannot read last', () => {
    expect(sortRows(rows, 'newest', keys, 'en').map((r) => r.n)).toEqual([
      'Alpha',
      'alpha 10',
      'beta',
      'alpha 9',
    ]);
    expect(sortRows([{ id: 1 }, { id: 3 }], 'newest', { time: (r) => r.id }, 'en')).toEqual([
      { id: 3 },
      { id: 1 },
    ]);
  });
  it('puts the largest first', () => {
    expect(sortRows(rows, 'largest', { size: (r) => r.s }, 'en').map((r) => r.s)).toEqual([
      9, 5, 3, 1,
    ]);
  });
  it('keeps the order for a sort the rows lack, without changing the input', () => {
    const out = sortRows(rows, 'largest', keys, 'en');
    expect(out).toEqual(rows);
    expect(out).not.toBe(rows);
  });
});

describe('matchesTerm', () => {
  it('matches any word, ignoring case and spaces around the term', () => {
    expect(matchesTerm(' ALP ', ['x', 'alpha'])).toBe(true);
    expect(matchesTerm('zz', ['x', undefined, null])).toBe(false);
    expect(matchesTerm('', [])).toBe(true);
  });
});
