import { intlLocale, type Locale } from '../../i18n/core';

const cache = new Map<string, Intl.NumberFormat>();

function nf(locale: Locale, opts: Intl.NumberFormatOptions = {}): Intl.NumberFormat {
  const key = `${locale}|${JSON.stringify(opts)}`;
  let f = cache.get(key);
  if (!f) {
    f = new Intl.NumberFormat(intlLocale(locale), opts);
    cache.set(key, f);
  }
  return f;
}

/** "12,500" / "۱۲٬۵۰۰". */
export function formatNumber(n: number, locale: Locale, maxFraction = 0): string {
  return nf(locale, { maximumFractionDigits: maxFraction }).format(n);
}

/** "5.8K" / "۵٫۸ هزار": for tiles and axis labels. */
export function formatCompact(n: number, locale: Locale): string {
  return nf(locale, { notation: 'compact', maximumFractionDigits: 1 }).format(n);
}

/** A ratio from 0 to 1 as "56.5%" / "۵۶٫۵٪". */
export function formatPercent(ratio: number, locale: Locale, maxFraction = 1): string {
  return nf(locale, { style: 'percent', maximumFractionDigits: maxFraction }).format(ratio);
}

const byteUnits = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'] as const;

/**
 * Splits a byte count into a number and a unit, 1024 to a step, as both
 * panels count traffic ("GB" there is 1024³ bytes).
 */
export function splitBytes(bytes: number): { value: number; unit: (typeof byteUnits)[number] } {
  let value = Math.abs(bytes);
  let i = 0;
  while (value >= 1024 && i < byteUnits.length - 1) {
    value /= 1024;
    i++;
  }
  return { value: Math.sign(bytes) * value, unit: byteUnits[i] ?? 'B' };
}

/**
 * A byte count as a number and a unit apart ("12.5" and "GB"), for a big
 * number with a small unit beside it. Under 10 of a unit keeps one decimal.
 */
export function bytesParts(bytes: number, locale: Locale): { value: string; unit: string } {
  const { value, unit } = splitBytes(bytes);
  const digits = unit === 'B' || Math.abs(value) >= 10 ? 0 : 1;
  return { value: formatNumber(value, locale, digits), unit };
}

/** "12.5 GB" / "۱۲٫۵ GB". */
export function formatBytes(bytes: number, locale: Locale): string {
  const { value, unit } = bytesParts(bytes, locale);
  return `${value} ${unit}`;
}

/** "1.2 MB/s". */
export function formatRate(bytesPerSecond: number, locale: Locale): string {
  return `${formatBytes(bytesPerSecond, locale)}/s`;
}
const nativeDigits: Record<string, string> = Object.fromEntries([
  ...[...'۰۱۲۳۴۵۶۷۸۹'].map((d, i) => [d, String(i)]),
  ...[...'٠١٢٣٤٥٦٧٨٩'].map((d, i) => [d, String(i)]),
  ['٫', '.'],
]);
/** Latin digits and point for the Persian and Arabic ones a phone keyboard types, trimmed. */
export function latinDigits(text: string): string {
  return [...text.trim()].map((c) => nativeDigits[c] ?? c).join('');
}
