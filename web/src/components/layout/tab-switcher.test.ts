import { describe, expect, it } from 'vitest';
import { currentTab, tabsFit } from './tab-switcher';

describe('tabsFit', () => {
  it('keeps the segments while they fit on a wide screen', () => {
    expect(tabsFit({ phone: false, need: 300, room: 300 })).toBe(true);
    expect(tabsFit({ phone: false, need: 301, room: 300 })).toBe(false);
  });
  it('always uses the button on a phone', () => {
    expect(tabsFit({ phone: true, need: 100, room: 300 })).toBe(false);
  });
  it('uses the button until the segments are measured', () => {
    expect(tabsFit({ phone: false, need: 0, room: 300 })).toBe(false);
  });
});

describe('currentTab', () => {
  const tabs = [
    { to: '/analytics/server?range=24h', label: 'Server' },
    { to: '/analytics/panels/2?range=24h', label: 'b' },
    { to: '/analytics/panels/12?range=24h', label: 'c' },
  ];
  it('matches the path without the range', () => {
    expect(currentTab(tabs, '/analytics/panels/2')?.label).toBe('b');
    expect(currentTab(tabs, '/analytics/panels/12')?.label).toBe('c');
  });
  it('matches a page under a tab', () => {
    expect(currentTab(tabs, '/analytics/server/cpu')?.label).toBe('Server');
  });
  it('finds nothing for another page', () => {
    expect(currentTab(tabs, '/analytics/panels/3')).toBeUndefined();
  });
});
