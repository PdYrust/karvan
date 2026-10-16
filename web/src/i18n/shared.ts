import type { Messages } from './locale';

const en = {
  brand: 'Karvan',
  languageLabel: 'Language',
  switchToDark: 'Dark theme',
  switchToLight: 'Light theme',
  soon: 'Soon',
  errorOffline: "Can't reach Karvan. Check your connection and try again.",
  errorServer: 'Something went wrong on the server. Try again in a bit.',
  errorStale: 'This page is out of date. Reload it.',
  retry: 'Try again',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  brand: 'کاروان',
  languageLabel: 'زبان',
  switchToDark: 'تم تیره',
  switchToLight: 'تم روشن',
  soon: 'به زودی',
  errorOffline: 'به کاروان وصل نشد. اینترنت رو چک کن و دوباره امتحان کن.',
  errorServer: 'یه مشکلی تو سرور پیش اومد. چند لحظه دیگه دوباره امتحان کن.',
  errorStale: 'این صفحه قدیمی شده. یه بار رفرشش کن.',
  retry: 'دوباره امتحان کن',
};

export const sharedMessages: Messages<Key> = { en, fa };

/** Language names are written in their own language, so either can be found. */
export const languageNames = { fa: 'فارسی', en: 'English' } as const;
