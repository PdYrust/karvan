import { describe, expect, it } from 'vitest';
import {
  dayIn,
  defaultCalendar,
  formatCountdown,
  formatDateTime,
  formatLongDate,
  formatRelative,
  formatSeconds,
  formatTime,
  startOfDay,
} from './date';

// 2026-10-15 15:10 UTC is 18:40 in Tehran (UTC+3:30, no DST since 2022).
const moment = new Date(Date.UTC(2026, 9, 15, 15, 10));

describe('dates in Tehran time', () => {
  it('uses the Jalali calendar and Persian digits for Persian', () => {
    expect(formatLongDate(moment, 'fa')).toBe('پنجشنبه ۲۳ مهر ۱۴۰۵');
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

  it('gives log lines their seconds', () => {
    const later = new Date(moment.getTime() + 5000);
    expect(formatSeconds(later, 'en')).toBe('18:40:05');
    expect(formatSeconds(later, 'fa')).toBe('۱۸:۴۰:۰۵');
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
describe('days in a time zone', () => {
  it('reads the day it is in the zone', () => {
    // 21:00 UTC is already the next day in Tehran.
    expect(dayIn(new Date(Date.UTC(2026, 2, 20, 21)), 'Asia/Tehran')).toEqual({
      year: 2026,
      month: 3,
      day: 21,
    });
    expect(dayIn(new Date(Date.UTC(2026, 2, 20, 21)), 'UTC')).toEqual({
      year: 2026,
      month: 3,
      day: 20,
    });
  });
  it('finds when a day starts in the zone', () => {
    expect(startOfDay({ year: 2026, month: 3, day: 21 }, 'Asia/Tehran').toISOString()).toBe(
      '2026-03-20T20:30:00.000Z',
    );
    expect(startOfDay({ year: 2026, month: 7, day: 1 }, 'Europe/Berlin').toISOString()).toBe(
      '2026-06-30T22:00:00.000Z',
    );
    expect(startOfDay({ year: 2026, month: 1, day: 1 }, 'Europe/Berlin').toISOString()).toBe(
      '2025-12-31T23:00:00.000Z',
    );
  });
  it('starts a day a clock change falls on at its real midnight', () => {
    // Clocks went forward at 2:00 on 29 March 2026 in Berlin; midnight was still +1.
    expect(startOfDay({ year: 2026, month: 3, day: 29 }, 'Europe/Berlin').toISOString()).toBe(
      '2026-03-28T23:00:00.000Z',
    );
    expect(startOfDay({ year: 2026, month: 3, day: 30 }, 'Europe/Berlin').toISOString()).toBe(
      '2026-03-29T22:00:00.000Z',
    );
  });
});
