import { intlLocale, type Locale } from '../../i18n/core';

/** Karvan's default time zone; schedules and logs are read in Tehran time. */
export const defaultTimeZone = 'Asia/Tehran';

export type Calendar = 'persian' | 'gregory';

export interface DateOptions {
  calendar?: Calendar;
  timeZone?: string;
}

/** Persian defaults to the Jalali calendar, English to Gregorian. */
export function defaultCalendar(locale: Locale): Calendar {
  return locale === 'fa' ? 'persian' : 'gregory';
}

function tag(locale: Locale, calendar: Calendar): string {
  return `${intlLocale(locale)}-u-ca-${calendar}`;
}

function formatter(
  locale: Locale,
  opts: DateOptions,
  fields: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(tag(locale, opts.calendar ?? defaultCalendar(locale)), {
    timeZone: opts.timeZone ?? defaultTimeZone,
    ...fields,
  });
}

/** "۲۳ مهر ۱۴۰۵، ۱۸:۴۰" / "Oct 15, 2026, 18:40" */
export function formatDateTime(date: Date, locale: Locale, opts: DateOptions = {}): string {
  return formatter(locale, opts, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).format(date);
}

/** "پنجشنبه ۲۳ مهر ۱۴۰۵" / "Thursday, October 15, 2026" */
export function formatLongDate(date: Date, locale: Locale, opts: DateOptions = {}): string {
  const f = formatter(locale, opts, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
  if (locale !== 'fa') return f.format(date);
  // ICU's Persian pattern puts the year first and uses a Latin comma
  // ("۱۴۰۵ مهر ۲۴, جمعه"); Persian readers expect weekday, day, month, year.
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    f.formatToParts(date).find((p) => p.type === type)?.value ?? '';
  return `${part('weekday')} ${part('day')} ${part('month')} ${part('year')}`;
}

/** "۱۸:۴۰" / "18:40" */
export function formatTime(date: Date, locale: Locale, opts: DateOptions = {}): string {
  return formatter(locale, opts, { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(
    date,
  );
}

const relativeSteps: [Intl.RelativeTimeFormatUnit, number][] = [
  ['second', 60],
  ['minute', 60],
  ['hour', 24],
  ['day', 7],
  ['week', 4.35],
  ['month', 12],
  ['year', Infinity],
];

/** "۵ دقیقه پیش" / "5 minutes ago". */
export function formatRelative(date: Date, now: Date, locale: Locale): string {
  const rtf = new Intl.RelativeTimeFormat(intlLocale(locale), { numeric: 'auto' });
  let value = (date.getTime() - now.getTime()) / 1000;
  for (const [unit, size] of relativeSteps) {
    if (Math.abs(value) < size) return rtf.format(Math.round(value), unit);
    value /= size;
  }
  return rtf.format(Math.round(value), 'year');
}

/** A countdown such as "4:05", with Persian digits in Persian. */
export function formatCountdown(totalSeconds: number, locale: Locale): string {
  const s = Math.max(0, Math.ceil(totalSeconds));
  const nf = new Intl.NumberFormat(intlLocale(locale), { minimumIntegerDigits: 2 });
  const minutes = new Intl.NumberFormat(intlLocale(locale)).format(Math.floor(s / 60));
  return `${minutes}:${nf.format(s % 60)}`;
}
