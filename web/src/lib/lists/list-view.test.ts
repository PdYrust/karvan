import { describe, expect, it } from 'vitest';
import { listView } from './list-view';

describe('listView', () => {
  it('shows only the skeleton until the first page is there', () => {
    expect(listView(undefined, false)).toBe('loading');
    expect(listView(undefined, true)).toBe('loading');
  });
  it('shows the empty card alone when nothing exists', () => {
    expect(listView([], false)).toBe('empty');
  });
  it('keeps the toolbar when a filter or search finds nothing', () => {
    expect(listView([], true)).toBe('list');
  });
  it('shows the toolbar with rows', () => {
    expect(listView([{ id: 1 }], false)).toBe('list');
  });
});
