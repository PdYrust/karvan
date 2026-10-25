import type { Messages } from '../../../i18n/locale';

const en = {
  title: 'Signed-in browsers',
  description:
    'Each browser signed in to Karvan now. Sign out any you do not know, then change the password.',
  caption: 'Sessions',
  browser: 'Browser',
  ip: 'Sign-in IP',
  started: 'Signed in',
  lastUsed: 'Last used',
  actions: 'Actions',
  thisBrowser: 'This browser',
  on: '{browser} on {system}',
  unknownBrowser: 'Unknown browser',
  signOut: 'Sign out',
  signOutNamed: 'Sign out {name}, signed in {time}',
  signOutOthers: 'Sign out other sessions',
  signedOutOne: 'Session signed out.',
  signedOutOthers: { one: '{count} session signed out.', other: '{count} sessions signed out.' },
  noOthers: 'No other sessions.',
  loadFailed: "Couldn't load the sessions.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  title: 'مرورگرهای وارد شده',
  description:
    'هر مرورگری که الان وارد کاروان شده است. هر کدام را که نمیشناسید خارج کنید و بعد رمز را عوض کنید.',
  caption: 'نشست ها',
  browser: 'مرورگر',
  ip: 'IP ورود',
  started: 'زمان ورود',
  lastUsed: 'آخرین استفاده',
  actions: 'کارها',
  thisBrowser: 'همین مرورگر',
  on: '{browser} روی {system}',
  unknownBrowser: 'مرورگر ناشناخته',
  signOut: 'خروج',
  signOutNamed: 'خروج {name}، ورود {time}',
  signOutOthers: 'خروج از نشست های دیگر',
  signedOutOne: 'نشست بسته شد.',
  signedOutOthers: { one: '{count} نشست بسته شد.', other: '{count} نشست بسته شد.' },
  noOthers: 'نشست دیگری نیست.',
  loadFailed: 'نشست ها بارگذاری نشدند.',
};

export const sessionsMessages: Messages<Key> = { en, fa };
