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

// Intl constructors are slow next to format(), and charts and lists format
// many dates per render, so each formatter is made once.
const formatters = new Map<string, Intl.DateTimeFormat>();
function formatter(
  locale: Locale,
  opts: DateOptions,
  fields: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  const calendar = opts.calendar ?? defaultCalendar(locale);
  const timeZone = opts.timeZone ?? defaultTimeZone;
  const key = `${locale}|${calendar}|${timeZone}|${JSON.stringify(fields)}`;
  let f = formatters.get(key);
  if (!f) {
    f = new Intl.DateTimeFormat(tag(locale, calendar), { timeZone, ...fields });
    formatters.set(key, f);
  }
  return f;
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

/** "۱۸:۴۰:۰۵" / "18:40:05": a log line, where seconds tell events apart. */
export function formatSeconds(date: Date, locale: Locale, opts: DateOptions = {}): string {
  return formatter(locale, opts, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  }).format(date);
}

/** "۲۷ اسفند ۱۴۰۸" / "Mar 17, 2030": a day that may be in another year, such as an expiry. */
export function formatDate(date: Date, locale: Locale, opts: DateOptions = {}): string {
  return formatter(locale, opts, { year: 'numeric', month: 'short', day: 'numeric' }).format(date);
}
/** "۲۳ مهر" / "Oct 15": axis labels and short lists. */
export function formatShortDate(date: Date, locale: Locale, opts: DateOptions = {}): string {
  return formatter(locale, opts, { month: 'short', day: 'numeric' }).format(date);
}

/** "تهران" / "Tehran" for Karvan's default zone, else the zone's own name ("Central European Time"). */
export function zoneLabel(timeZone: string, locale: Locale): string {
  if (timeZone === defaultTimeZone) return locale === 'fa' ? 'تهران' : 'Tehran';
  const part = new Intl.DateTimeFormat(intlLocale(locale), {
    timeZone,
    timeZoneName: 'longGeneric',
  })
    .formatToParts(new Date())
    .find((p) => p.type === 'timeZoneName');
  return part?.value ?? timeZone;
}

/**
 * The zone's name as it reads after "in" or "به وقت". Persian names from Intl
 * already start with وقت ("وقت مرکز اروپا"), so the sentence would say it twice.
 */
export function zoneInSentence(timeZone: string, locale: Locale): string {
  const name = zoneLabel(timeZone, locale);
  return locale === 'fa' ? name.replace(/^وقت /, '') : name;
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
/** A day on the Gregorian calendar, as numbers. */
export interface Day {
  year: number;
  month: number;
  day: number;
}
const civilFormat = (timeZone: string) =>
  new Intl.DateTimeFormat('en-u-ca-gregory-nu-latn', {
    timeZone,
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    second: 'numeric',
    hourCycle: 'h23',
  });
/** The wall clock of timeZone at date, read as if it were UTC. */
function wallClock(date: Date, timeZone: string): number {
  const parts = civilFormat(timeZone).formatToParts(date);
  const n = (type: Intl.DateTimeFormatPartTypes) =>
    Number(parts.find((p) => p.type === type)?.value ?? 0);
  return Date.UTC(n('year'), n('month') - 1, n('day'), n('hour'), n('minute'), n('second'));
}
/** The moment day starts in timeZone. */
export function startOfDay(day: Day, timeZone: string): Date {
  const local = Date.UTC(day.year, day.month - 1, day.day);
  // The zone's offset at a first guess, then once more at the result, for days a clock change starts.
  let at = local - (wallClock(new Date(local), timeZone) - local);
  at = local - (wallClock(new Date(at), timeZone) - at);
  return new Date(at);
}
