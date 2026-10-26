import { describe, expect, it } from 'vitest';
import { topRows, type TopItem } from './top-list';

const item = (name: string, value: number): TopItem => ({
  key: name,
  name,
  label: name,
  value,
  display: String(value),
});
const items = [item('root', 40), item('alice', 5), item('bob', 12), item('alina', 3)];

describe('topRows', () => {
  it('lists the largest first and stops at the count', () => {
    expect(topRows(items, { sort: 'share', count: 2 }).map((i) => i.name)).toEqual(['root', 'bob']);
  });
  it('sorts by name', () => {
    expect(topRows(items, { sort: 'name', count: 9 }).map((i) => i.name)).toEqual([
      'alice',
      'alina',
      'bob',
      'root',
    ]);
  });
  it('keeps only the rows the search finds, in any case', () => {
    expect(topRows(items, { query: ' AL', sort: 'share', count: 9 }).map((i) => i.name)).toEqual([
      'alice',
      'alina',
    ]);
  });
});
