import type { Message } from '../../i18n/core';
import type { Messages } from '../../i18n/locale';

const en = {
  pageTitle: 'Overview · Karvan',
  title: 'Overview',
  today: '{date}. It is {time} in Tehran.',
  loading: 'Loading the overview',
  loadFailed: 'The overview didn’t load.',
  panelsTitle: 'Panels',
  panelsEmptyTitle: 'No panels on this server yet',
  panelsEmptyBody:
    'Finding Marzban and PasarGuard on this server arrives in the next version. Karvan only reads a panel’s settings; it never changes them.',
  scanAgain: 'Scan again',
  safeModeTitle: 'Safe Mode',
  on: 'On',
  off: 'Off',
  safeModeBody:
    'Every migration starts with a backup of the destination, and a failed run is rolled back.',
  twoStep: 'Two-step sign-in',
  twoStepOffHint:
    'Ask for a code from your phone at every sign-in. You can turn it on from Security, coming next.',
  twoStepOnHint: 'Every sign-in asks for a code from your phone.',
  previousSignIn: 'Previous sign-in',
  firstSignIn: 'This is the first sign-in.',
  accountTitle: 'Account',
  signedInAs: 'Signed in as',
  panelsSupported: 'Supported panels',
};

type Key = keyof typeof en;

const fa: Record<Key, Message> = {
  pageTitle: 'نمای کلی · کاروان',
  title: 'نمای کلی',
  today: '{date}؛ ساعت {time} به وقت تهران.',
  loading: 'در حال بارگذاری نمای کلی',
  loadFailed: 'نمای کلی بارگذاری نشد.',
  panelsTitle: 'پنل‌ها',
  panelsEmptyTitle: 'هنوز پنلی روی این سرور نیست',
  panelsEmptyBody:
    'پیدا کردن مرزبان و پاسارگارد روی این سرور در نسخهٔ بعدی می‌آید. کاروان تنظیمات پنل را فقط می‌خواند و هرگز تغییرشان نمی‌دهد.',
  scanAgain: 'جست‌وجوی دوباره',
  safeModeTitle: 'حالت امن',
  on: 'روشن',
  off: 'خاموش',
  safeModeBody:
    'هر انتقال با گرفتن پشتیبان از مقصد شروع می‌شود و اگر کار نیمه‌کاره بماند، همه‌چیز برمی‌گردد.',
  twoStep: 'ورود دومرحله‌ای',
  twoStepOffHint: 'در هر ورود، کدی از گوشی‌تان خواسته می‌شود. روشن کردنش از بخش امنیت، به‌زودی.',
  twoStepOnHint: 'هر ورود به کدی از گوشی‌تان نیاز دارد.',
  previousSignIn: 'ورود قبلی',
  firstSignIn: 'این اولین ورود است.',
  accountTitle: 'حساب',
  signedInAs: 'وارد شده با',
  panelsSupported: 'پنل‌های پشتیبانی‌شده',
};

export const overviewMessages: Messages<Key> = { en, fa };
export type OverviewKey = Key;
