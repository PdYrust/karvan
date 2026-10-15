import { describe, expect, it } from 'vitest';
import {
  defaultCalendar,
  formatCountdown,
  formatDateTime,
  formatLongDate,
  formatRelative,
  formatTime,
} from './date';

// 2026-10-15 15:10 UTC is 18:40 in Tehran (UTC+3:30, no DST since 2022).
const moment = new Date(Date.UTC(2026, 9, 15, 15, 10));

describe('dates in Tehran time', () => {
  it('uses the Jalali calendar and Persian digits for Persian', () => {
    expect(formatLongDate(moment, 'fa')).toContain('مهر');
    expect(formatLongDate(moment, 'fa')).toContain('۱۴۰۵');
    expect(formatTime(moment, 'fa')).toBe('۱۸:۴۰');
  });

  it('uses the Gregorian calendar for English', () => {
    expect(formatLongDate(moment, 'en')).toBe('Thursday, October 15, 2026');
    expect(formatDateTime(moment, 'en')).toBe('Oct 15, 2026, 18:40');
  });

  it('can switch calendar and zone', () => {
    expect(formatLongDate(moment, 'en', { calendar: 'persian' })).toContain('1405');
    expect(formatTime(moment, 'en', { timeZone: 'UTC' })).toBe('15:10');
  });

  it('picks the calendar from the language', () => {
    expect(defaultCalendar('fa')).toBe('persian');
    expect(defaultCalendar('en')).toBe('gregory');
  });
});

describe('formatRelative', () => {
  const now = new Date(Date.UTC(2026, 9, 15, 12, 0));
  it.each([
    [-30, 'en', '30 seconds ago'],
    [-5 * 60, 'en', '5 minutes ago'],
    [-3 * 3600, 'en', '3 hours ago'],
    [-86400, 'en', 'yesterday'],
    [-3 * 7 * 86400, 'en', '3 weeks ago'],
    [-400 * 86400, 'en', 'last year'],
    [-5 * 60, 'fa', '۵ دقیقه پیش'],
  ] as const)('%d seconds in %s', (offset, locale, want) => {
    expect(formatRelative(new Date(now.getTime() + offset * 1000), now, locale)).toBe(want);
  });

  it('handles the far future', () => {
    expect(formatRelative(new Date(now.getTime() + 5 * 366 * 86400 * 1000), now, 'en')).toBe(
      'in 5 years',
    );
  });
});

describe('formatCountdown', () => {
  it('pads seconds and rounds up', () => {
    expect(formatCountdown(299.2, 'en')).toBe('5:00');
    expect(formatCountdown(65, 'en')).toBe('1:05');
    expect(formatCountdown(-3, 'en')).toBe('0:00');
    expect(formatCountdown(65, 'fa')).toBe('۱:۰۵');
  });
});
