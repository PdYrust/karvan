import { createContext, use, useMemo, useState, type ReactNode } from 'react';
import { readLocale, saveLocale } from '../lib/preferences/preferences';
import { createTranslate, directionOf, type Locale, type Message, type Translate } from './core';

interface LocaleContextValue {
  locale: Locale;
  dir: 'rtl' | 'ltr';
  setLocale: (l: Locale) => void;
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(readLocale);
  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      dir: directionOf(locale),
      setLocale: (l) => {
        saveLocale(l);
        setLocaleState(l);
      },
    }),
    [locale],
  );
  return <LocaleContext value={value}>{children}</LocaleContext>;
}

export function useLocale(): LocaleContextValue {
  const ctx = use(LocaleContext);
  if (!ctx) throw new Error('useLocale needs a LocaleProvider');
  return ctx;
}

/** Messages in both languages; Persian must have every English key. */
export type Messages<K extends string> = Readonly<Record<Locale, Readonly<Record<K, Message>>>>;

export function useT<K extends string>(messages: Messages<K>): Translate<K> {
  const { locale } = useLocale();
  return useMemo(() => createTranslate(messages[locale], locale), [messages, locale]);
}
