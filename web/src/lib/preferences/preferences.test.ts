import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  applyTheme,
  readLocale,
  readThemePref,
  resolveMode,
  saveLocale,
  saveThemePref,
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
    vi.restoreAllMocks();
  });
});
