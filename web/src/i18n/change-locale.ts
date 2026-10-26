import { saveLocale } from '../lib/preferences/preferences';
import type { Locale } from './core';
/**
 * Saves a new language and reloads the page, so everything on it (dates,
 * direction, text set before the change) comes back in that language at once.
 */
export function changeLocale(
  next: Locale,
  current: Locale,
  reload: () => void = () => window.location.reload(),
): void {
  if (next === current) return;
  saveLocale(next);
  reload();
}
