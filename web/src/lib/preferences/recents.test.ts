import { afterEach, describe, expect, it, vi } from 'vitest';
import { maxRecents, readRecents, recordRecent } from './recents';

const key = 'karvan.recents';

afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
});

describe('recents', () => {
  it('starts empty', () => {
    expect(readRecents()).toEqual([]);
  });

  it('puts the newest visit first and keeps each page once', () => {
    recordRecent('/panels');
    recordRecent('/analytics/server');
    expect(recordRecent('/panels')).toEqual(['/panels', '/analytics/server']);
    expect(readRecents()).toEqual(['/panels', '/analytics/server']);
  });

  it('keeps only the latest pages', () => {
    for (let i = 1; i <= maxRecents + 2; i++) recordRecent(`/panels/${String(i)}`);
    const kept = readRecents();
    expect(kept).toHaveLength(maxRecents);
    expect(kept[0]).toBe(`/panels/${String(maxRecents + 2)}`);
    expect(kept).not.toContain('/panels/1');
  });

  it('records only paths inside the panel', () => {
    for (const bad of ['', 'panels', '//example.com', 'https://example.com', '/a b', '/x?y=1']) {
      expect(recordRecent(bad)).toEqual([]);
    }
    expect(localStorage.getItem(key)).toBeNull();
  });

  it('ignores whatever else is stored', () => {
    localStorage.setItem(key, 'not json');
    expect(readRecents()).toEqual([]);
    localStorage.setItem(key, '{"path":"/panels"}');
    expect(readRecents()).toEqual([]);
    localStorage.setItem(key, JSON.stringify(['/panels', 7, '//example.com', '/panels', null]));
    expect(readRecents()).toEqual(['/panels']);
  });

  it('works without storage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    expect(readRecents()).toEqual([]);
    expect(recordRecent('/panels')).toEqual(['/panels']);
  });
});
