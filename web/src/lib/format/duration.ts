import type { Locale } from '../../i18n/core';
import { formatNumber } from './number';

type Unit = 'day' | 'hour' | 'minute' | 'second';

const units: [Unit, number][] = [
  ['day', 86400],
  ['hour', 3600],
  ['minute', 60],
  ['second', 1],
];

const names: Record<Locale, Record<Unit, [string, string]>> = {
  en: {
    day: ['day', 'days'],
    hour: ['hour', 'hours'],
    minute: ['minute', 'minutes'],
    second: ['second', 'seconds'],
  },
  fa: {
    day: ['روز', 'روز'],
    hour: ['ساعت', 'ساعت'],
    minute: ['دقیقه', 'دقیقه'],
    second: ['ثانیه', 'ثانیه'],
  },
};

const joiner: Record<Locale, string> = { en: ' ', fa: ' و ' };

/**
 * The two largest units of a duration: "3 days 4 hours" / "۳ روز و ۴ ساعت",
 * "45 seconds". Rounds down, so a countdown never promises less than it has.
 */
export function formatDuration(totalSeconds: number, locale: Locale, parts = 2): string {
  let rest = Math.max(0, Math.floor(totalSeconds));
  const out: string[] = [];
  for (const [unit, size] of units) {
    const n = Math.floor(rest / size);
    rest -= n * size;
    // "3 days", never "3 days 0 hours".
    if (out.length > 0 && n === 0) break;
    if (n > 0 || unit === 'second') {
      const [one, many] = names[locale][unit];
      out.push(`${formatNumber(n, locale)} ${n === 1 ? one : many}`);
    }
    if (out.length === parts) break;
  }
  return out.join(joiner[locale]);
}
