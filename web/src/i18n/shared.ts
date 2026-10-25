import type { Messages } from './locale';

const en = {
  brand: 'Karvan',
  languageLabel: 'Language',
  switchToDark: 'Dark theme',
  switchToLight: 'Light theme',
  errorOffline: "Can't reach Karvan. Check your connection and try again.",
  errorServer: 'Something went wrong on the server. Try again in a bit.',
  errorStale: 'This page is out of date. Reload it.',
  retry: 'Try again',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  brand: 'کاروان',
  languageLabel: 'زبان',
  switchToDark: 'پوسته تیره',
  switchToLight: 'پوسته روشن',
  errorOffline: 'اتصال به کاروان برقرار نشد. اتصال را بررسی کنید و دوباره امتحان کنید.',
  errorServer: 'مشکلی در سرور پیش آمد. چند لحظه دیگر دوباره امتحان کنید.',
  errorStale: 'این صفحه قدیمی است. صفحه را دوباره بارگذاری کنید.',
  retry: 'دوباره امتحان کنید',
};

export const sharedMessages: Messages<Key> = { en, fa };

/** Language names are written in their own language, so either can be found. */
export const languageNames = { fa: 'فارسی', en: 'English' } as const;
