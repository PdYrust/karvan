import type { Locale } from '../../i18n/core';
import type { CalendarPref } from '../preferences/preferences';
import {
  defaultCalendar,
  formatDateTime,
  formatLongDate,
  formatRelative,
  formatShortDate,
  formatTime,
  zoneLabel,
  type DateOptions,
} from './date';
import { formatDuration } from './duration';
import { formatBytes, formatCompact, formatNumber, formatPercent, formatRate } from './number';

export interface Region {
  timeZone: string;
  calendar: CalendarPref;
}

/** Every formatter bound to the operator's language, zone and calendar. */
export function makeFormat(locale: Locale, region: Region) {
  const opts: DateOptions = {
    timeZone: region.timeZone,
    calendar: region.calendar === 'auto' ? defaultCalendar(locale) : region.calendar,
  };
  return {
    locale,
    timeZone: region.timeZone,
    zone: zoneLabel(region.timeZone, locale),
    number: (n: number, maxFraction?: number) => formatNumber(n, locale, maxFraction),
    compact: (n: number) => formatCompact(n, locale),
    percent: (ratio: number, maxFraction?: number) => formatPercent(ratio, locale, maxFraction),
    bytes: (b: number) => formatBytes(b, locale),
    rate: (bytesPerSecond: number) => formatRate(bytesPerSecond, locale),
    duration: (seconds: number, parts?: number) => formatDuration(seconds, locale, parts),
    dateTime: (d: Date) => formatDateTime(d, locale, opts),
    longDate: (d: Date) => formatLongDate(d, locale, opts),
    shortDate: (d: Date) => formatShortDate(d, locale, opts),
    time: (d: Date) => formatTime(d, locale, opts),
    relative: (d: Date, now: Date) => formatRelative(d, now, locale),
  };
}

export type Format = ReturnType<typeof makeFormat>;
