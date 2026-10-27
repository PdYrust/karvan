import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  applyTheme,
  isTimeZone,
  readCalendarPref,
  readCurrentPanel,
  readLocale,
  readSidebarCollapsed,
  readThemePref,
  readTimeZone,
  resolveMode,
  saveCalendarPref,
  saveCurrentPanel,
  saveLocale,
  saveSidebarCollapsed,
  saveThemePref,
  saveTimeZone,
  watchSystemTheme,
} from './preferences';

function mockSystemDark(dark: boolean) {
  const listeners = new Set<() => void>();
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({
      matches: dark,
      addEventListener: (_: string, fn: () => void) => listeners.add(fn),
      removeEventListener: (_: string, fn: () => void) => listeners.delete(fn),
    })),
  );
  return listeners;
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe('language', () => {
  it('saves and applies the language and direction', () => {
    saveLocale('en');
    expect(document.documentElement.dir).toBe('ltr');
    expect(readLocale()).toBe('en');
    expect(localStorage.getItem('karvan.lang')).toBe('en');
    saveLocale('fa');
    expect(document.documentElement.dir).toBe('rtl');
    expect(readLocale()).toBe('fa');
  });
});

describe('theme', () => {
  it('follows the system until a choice is saved', () => {
    mockSystemDark(true);
    expect(readThemePref()).toBe('system');
    expect(applyTheme('system')).toBe('dark');
    expect(document.documentElement.dataset.mode).toBe('dark');
  });

  it('remembers an explicit choice', () => {
    mockSystemDark(true);
    expect(saveThemePref('light')).toBe('light');
    expect(readThemePref()).toBe('light');
    expect(resolveMode('dark')).toBe('dark');
  });

  it('points the status bar color at the chosen theme', () => {
    mockSystemDark(false);
    document.head.innerHTML =
      '<meta name="theme-color" data-for="light" media="(prefers-color-scheme: light)" />' +
      '<meta name="theme-color" data-for="dark" media="(prefers-color-scheme: dark)" />';
    const media = () =>
      [...document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')].map(
        (m) => m.media,
      );
    applyTheme('dark');
    expect(media()).toEqual(['not all', 'all']);
    applyTheme('system');
    expect(media()).toEqual(['(prefers-color-scheme: light)', '(prefers-color-scheme: dark)']);
    document.head.innerHTML = '';
  });
  it('ignores garbage in storage', () => {
    localStorage.setItem('karvan.theme', 'purple');
    expect(readThemePref()).toBe('system');
  });

  it('watches system changes and stops watching', () => {
    const listeners = mockSystemDark(false);
    const stop = watchSystemTheme(() => undefined);
    expect(listeners.size).toBe(1);
    stop();
    expect(listeners.size).toBe(0);
  });

  it('reads defaults when storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    expect(readTimeZone()).toBe('Asia/Tehran');
    expect(readCurrentPanel()).toBeNull();
    expect(readCalendarPref()).toBe('auto');
  });

  it('survives blocked storage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    mockSystemDark(false);
    expect(readThemePref()).toBe('system');
    expect(saveThemePref('dark')).toBe('dark');
  });
});

describe('region', () => {
  it('defaults to Tehran and keeps only real zones', () => {
    expect(readTimeZone()).toBe('Asia/Tehran');
    saveTimeZone('Europe/Berlin');
    expect(readTimeZone()).toBe('Europe/Berlin');
    localStorage.setItem('karvan.timeZone', 'Mars/Olympus');
    expect(readTimeZone()).toBe('Asia/Tehran');
    expect(isTimeZone('UTC')).toBe(true);
    saveTimeZone('Asia/Tehran');
    expect(localStorage.getItem('karvan.timeZone')).toBeNull();
  });

  it('follows the language until a calendar is chosen', () => {
    expect(readCalendarPref()).toBe('auto');
    saveCalendarPref('gregory');
    expect(readCalendarPref()).toBe('gregory');
    saveCalendarPref('auto');
    expect(localStorage.getItem('karvan.calendar')).toBeNull();
  });
});

describe('layout and panel', () => {
  it('remembers a collapsed sidebar', () => {
    expect(readSidebarCollapsed()).toBe(false);
    saveSidebarCollapsed(true);
    expect(readSidebarCollapsed()).toBe(true);
    saveSidebarCollapsed(false);
    expect(readSidebarCollapsed()).toBe(false);
  });

  it('remembers the current panel and rejects junk', () => {
    expect(readCurrentPanel()).toBeNull();
    saveCurrentPanel(3);
    expect(readCurrentPanel()).toBe(3);
    localStorage.setItem('karvan.panel', '-1');
    expect(readCurrentPanel()).toBeNull();
    saveCurrentPanel(null);
    expect(localStorage.getItem('karvan.panel')).toBeNull();
  });
});
