import { describe, expect, it } from 'vitest';
import {
  fromJalali,
  gregorianMonthLength,
  isJalaliLeap,
  jalaliMonthLength,
  toJalali,
  type CivilDate,
} from './jalali';

const persian = new Intl.DateTimeFormat('en-u-ca-persian-nu-latn', {
  timeZone: 'UTC',
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
});

/** The Jalali date Intl gives for a Gregorian day. */
function intlJalali(g: CivilDate): CivilDate {
  const parts = persian.formatToParts(new Date(Date.UTC(g.year, g.month - 1, g.day)));
  const part = (type: string) => Number(parts.find((p) => p.type === type)?.value);
  return { year: part('year'), month: part('month'), day: part('day') };
}

function nextDay(g: CivilDate): CivilDate {
  const d = new Date(Date.UTC(g.year, g.month - 1, g.day + 1));
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate() };
}

describe('jalali', () => {
  it('matches Intl on every day of 1300 to 1500 SH, both ways', () => {
    // 1 Farvardin 1300 is 21 March 1921; the loop stops after 29 or 30 Esfand 1500.
    let g: CivilDate = { year: 1921, month: 3, day: 21 };
    let days = 0;
    const wrong: string[] = [];
    for (;;) {
      const want = intlJalali(g);
      if (want.year > 1500) break;
      const got = toJalali(g);
      const back = fromJalali(want);
      if (
        got.year !== want.year ||
        got.month !== want.month ||
        got.day !== want.day ||
        back.year !== g.year ||
        back.month !== g.month ||
        back.day !== g.day
      ) {
        wrong.push(`${g.year}-${g.month}-${g.day}`);
      }
      g = nextDay(g);
      days++;
    }
    expect(wrong).toEqual([]);
    // 201 years of 365 or 366 days.
    expect(days).toBeGreaterThan(201 * 365);
    expect(days).toBeLessThan(201 * 366);
  });

  it('gives each month its length, Esfand 30 days in a leap year', () => {
    expect(jalaliMonthLength(1403, 1)).toBe(31);
    expect(jalaliMonthLength(1403, 6)).toBe(31);
    expect(jalaliMonthLength(1403, 7)).toBe(30);
    expect(jalaliMonthLength(1403, 12)).toBe(30);
    expect(jalaliMonthLength(1404, 12)).toBe(29);
    expect(isJalaliLeap(1403)).toBe(true);
    expect(isJalaliLeap(1404)).toBe(false);
  });

  it('knows the length of every Esfand from Intl', () => {
    for (let y = 1300; y <= 1500; y++) {
      const lastDay = fromJalali({ year: y, month: 12, day: jalaliMonthLength(y, 12) });
      expect(intlJalali(nextDay(lastDay))).toEqual({ year: y + 1, month: 1, day: 1 });
    }
  });

  it('gives Gregorian months their length', () => {
    expect(gregorianMonthLength(2024, 2)).toBe(29);
    expect(gregorianMonthLength(2026, 2)).toBe(28);
    expect(gregorianMonthLength(2026, 4)).toBe(30);
    expect(gregorianMonthLength(2026, 12)).toBe(31);
  });
});
