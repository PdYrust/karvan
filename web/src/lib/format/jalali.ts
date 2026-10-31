/**
 * Jalali (Solar Hijri) and Gregorian dates as plain numbers. The
 * arithmetic is the 33-year rule ICU uses, so the browser's own Intl
 * formatting of a date and this module always agree; the tests check every
 * day of 1300 to 1500 SH against Intl.
 */
export interface CivilDate {
  year: number;
  /** 1 to 12. */
  month: number;
  day: number;
}

/** 1 Farvardin 1 as days since 1 January 1970. */
const epoch = -492268;
/** Days in the year before each month: six months of 31, five of 30, then Esfand. */
const before = [0, 31, 62, 93, 124, 155, 186, 216, 246, 276, 306, 336];

const unixDay = (g: CivilDate) => Math.floor(Date.UTC(g.year, g.month - 1, g.day) / 86_400_000);

/** Days from 1 Farvardin 1 to 1 Farvardin of year. */
const yearStart = (year: number) => 365 * (year - 1) + Math.floor((8 * year + 21) / 33);

export function isJalaliLeap(year: number): boolean {
  return (((25 * year + 11) % 33) + 33) % 33 < 8;
}

export function jalaliMonthLength(year: number, month: number): number {
  if (month <= 6) return 31;
  if (month <= 11) return 30;
  return isJalaliLeap(year) ? 30 : 29;
}

export function gregorianMonthLength(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

export function toJalali(g: CivilDate): CivilDate {
  const days = unixDay(g) - epoch;
  const year = 1 + Math.floor((33 * days + 3) / 12053);
  const inYear = days - yearStart(year);
  const month = inYear < 216 ? Math.floor(inYear / 31) : Math.floor((inYear - 6) / 30);
  return { year, month: month + 1, day: inYear - (before[month] ?? 0) + 1 };
}

export function fromJalali(j: CivilDate): CivilDate {
  const days = epoch + yearStart(j.year) + (before[j.month - 1] ?? 0) + j.day - 1;
  const d = new Date(days * 86_400_000);
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate() };
}
