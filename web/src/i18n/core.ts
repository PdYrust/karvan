export type Locale = 'fa' | 'en';

export const locales: readonly Locale[] = ['fa', 'en'];

/** A message with plural forms, chosen with Intl.PluralRules. */
export interface Plural {
  one: string;
  other: string;
}

export type Message = string | Plural;

export type Params = Record<string, string | number>;

export type Translate<K extends string> = (key: K, params?: Params) => string;

export function directionOf(locale: Locale): 'rtl' | 'ltr' {
  return locale === 'fa' ? 'rtl' : 'ltr';
}

/** Intl locale tag: Persian uses the Persian calendar and Persian digits. */
export function intlLocale(locale: Locale): string {
  return locale === 'fa' ? 'fa-IR' : 'en-US';
}

const pluralRules = new Map<Locale, Intl.PluralRules>();

function pluralForm(locale: Locale, n: number): 'one' | 'other' {
  let rules = pluralRules.get(locale);
  if (!rules) {
    rules = new Intl.PluralRules(intlLocale(locale));
    pluralRules.set(locale, rules);
  }
  return rules.select(n) === 'one' ? 'one' : 'other';
}

/**
 * Builds t() for one dictionary. `{name}` placeholders are filled from params;
 * numbers are formatted for the locale, so Persian gets Persian digits.
 * A plural message picks its form from params.count.
 */
export function createTranslate<K extends string>(
  dict: Readonly<Record<K, Message>>,
  locale: Locale,
): Translate<K> {
  const numbers = new Intl.NumberFormat(intlLocale(locale));
  return (key, params) => {
    const msg = dict[key];
    const text =
      typeof msg === 'string' ? msg : msg[pluralForm(locale, Number(params?.count ?? 0))];
    if (!params) return text;
    return text.replace(/\{(\w+)\}/g, (whole, name: string) => {
      const value = params[name];
      if (value === undefined) return whole;
      return typeof value === 'number' ? numbers.format(value) : value;
    });
  };
}
