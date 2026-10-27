import { directionOf, type Locale } from '../../i18n/core';
import { defaultTimeZone } from '../format/date';

// Same keys as public/boot.js, which applies them before first paint.
const langKey = 'karvan.lang';
const themeKey = 'karvan.theme';
const timeZoneKey = 'karvan.timeZone';
const calendarKey = 'karvan.calendar';
const sidebarKey = 'karvan.sidebar';
const panelKey = 'karvan.panel';

export type ThemePref = 'system' | 'light' | 'dark';
export type Mode = 'light' | 'dark';
/** "auto" follows the language: Jalali in Persian, Gregorian in English. */
export type CalendarPref = 'auto' | 'persian' | 'gregory';

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string | null): void {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    // Blocked storage only means the choice is not remembered.
  }
}

export function isTimeZone(tz: string): boolean {
  try {
    new Intl.DateTimeFormat('en', { timeZone: tz });
    return true;
  } catch {
    return false;
  }
}

export function readTimeZone(): string {
  const v = read(timeZoneKey);
  return v && isTimeZone(v) ? v : defaultTimeZone;
}

export function saveTimeZone(tz: string): void {
  write(timeZoneKey, tz === defaultTimeZone ? null : tz);
}

export function readCalendarPref(): CalendarPref {
  const v = read(calendarKey);
  return v === 'persian' || v === 'gregory' ? v : 'auto';
}

export function saveCalendarPref(pref: CalendarPref): void {
  write(calendarKey, pref === 'auto' ? null : pref);
}

export function readSidebarCollapsed(): boolean {
  return read(sidebarKey) === 'collapsed';
}

export function saveSidebarCollapsed(collapsed: boolean): void {
  write(sidebarKey, collapsed ? 'collapsed' : null);
}

/** The panel the switcher points at; null is "All panels". */
export function readCurrentPanel(): number | null {
  const n = Number(read(panelKey));
  return Number.isInteger(n) && n > 0 ? n : null;
}

export function saveCurrentPanel(id: number | null): void {
  write(panelKey, id === null ? null : String(id));
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
  syncThemeColor(pref);
  return mode;
}

/**
 * The page has a theme-color tag per theme (index.html). A saved choice wins
 * over the system's, so an installed app's status bar matches the page;
 * boot.js does the same before first paint.
 */
function syncThemeColor(pref: ThemePref) {
  for (const meta of document.querySelectorAll<HTMLMetaElement>(
    'meta[name="theme-color"][data-for]',
  )) {
    const own = meta.dataset.for;
    meta.media =
      pref === 'system' ? `(prefers-color-scheme: ${own})` : own === pref ? 'all' : 'not all';
  }
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
