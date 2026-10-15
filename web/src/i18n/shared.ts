import type { Messages } from './locale';

const en = {
  brand: 'Karvan',
  languageLabel: 'Language',
  themeLabel: 'Theme',
  themeLight: 'Light',
  themeDark: 'Dark',
  themeSystem: 'Match system',
  switchToDark: 'Switch to dark theme',
  switchToLight: 'Switch to light theme',
  soon: 'Soon',
  errorOffline: 'Karvan can’t be reached. Check your connection, then try again.',
  errorServer: 'Something went wrong on the server. Try again in a moment.',
  errorStale: 'This page is out of date. Reload it and try again.',
  reload: 'Reload',
  retry: 'Try again',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  brand: 'کاروان',
  languageLabel: 'زبان',
  themeLabel: 'پوسته',
  themeLight: 'روشن',
  themeDark: 'تیره',
  themeSystem: 'مثل سیستم',
  switchToDark: 'رفتن به پوستهٔ تیره',
  switchToLight: 'رفتن به پوستهٔ روشن',
  soon: 'به‌زودی',
  errorOffline: 'به کاروان دسترسی نداریم. اتصال را بررسی کنید و دوباره امتحان کنید.',
  errorServer: 'در سرور مشکلی پیش آمد. چند لحظهٔ دیگر دوباره امتحان کنید.',
  errorStale: 'این صفحه قدیمی شده است. آن را دوباره بارگذاری کنید.',
  reload: 'بارگذاری دوباره',
  retry: 'دوباره امتحان کنید',
};

export const sharedMessages: Messages<Key> = { en, fa };

/** Language names are written in their own language, so either can be found. */
export const languageNames = { fa: 'فارسی', en: 'English' } as const;
