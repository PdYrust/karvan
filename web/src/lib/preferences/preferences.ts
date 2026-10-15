import { directionOf, type Locale } from '../../i18n/core';

// Same keys as public/boot.js, which applies them before first paint.
const langKey = 'karvan.lang';
const themeKey = 'karvan.theme';

export type ThemePref = 'system' | 'light' | 'dark';
export type Mode = 'light' | 'dark';

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Blocked storage only means the choice is not remembered.
  }
}

export function readLocale(): Locale {
  return document.documentElement.lang === 'en' ? 'en' : 'fa';
}

export function saveLocale(locale: Locale): void {
  write(langKey, locale);
  document.documentElement.lang = locale;
  document.documentElement.dir = directionOf(locale);
}

export function readThemePref(): ThemePref {
  const v = read(themeKey);
  return v === 'light' || v === 'dark' ? v : 'system';
}

const darkQuery = '(prefers-color-scheme: dark)';

export function resolveMode(pref: ThemePref): Mode {
  if (pref !== 'system') return pref;
  return window.matchMedia(darkQuery).matches ? 'dark' : 'light';
}

export function applyTheme(pref: ThemePref): Mode {
  const mode = resolveMode(pref);
  document.documentElement.dataset.mode = mode;
  return mode;
}

export function saveThemePref(pref: ThemePref): Mode {
  write(themeKey, pref);
  return applyTheme(pref);
}

/** Calls onChange when the system theme flips; returns the unsubscribe. */
export function watchSystemTheme(onChange: () => void): () => void {
  const mq = window.matchMedia(darkQuery);
  mq.addEventListener('change', onChange);
  return () => mq.removeEventListener('change', onChange);
}
