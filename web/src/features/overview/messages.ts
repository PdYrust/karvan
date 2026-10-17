import type { Messages } from '../../i18n/locale';

const en = {
  pageTitle: 'Overview | Karvan',
  title: 'Overview',
  today: '{date}, {time} Tehran time',
  loading: 'Loading',
  loadFailed: "The overview didn't load.",
  panelsTitle: 'Panels',
  panelsEmptyTitle: 'No panels found yet',
  scanAgain: 'Scan again',
  safeModeTitle: 'Safe Mode',
  on: 'On',
  off: 'Off',
  previousSignIn: 'Last sign-in',
  firstSignIn: 'First time',
  accountTitle: 'Account',
  signedInAs: 'Signed in as',
  panelsSupported: 'Works with',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  pageTitle: 'نمای کلی | کاروان',
  title: 'نمای کلی',
  today: '{date}، ساعت {time} تهران',
  loading: 'در حال بارگذاری',
  loadFailed: 'نمای کلی بارگذاری نشد.',
  panelsTitle: 'پنل ها',
  panelsEmptyTitle: 'هنوز پنلی پیدا نشد',
  scanAgain: 'جستجوی دوباره',
  safeModeTitle: 'حالت امن',
  on: 'روشن',
  off: 'خاموش',
  previousSignIn: 'ورود قبلی',
  firstSignIn: 'اولین ورود',
  accountTitle: 'حساب',
  signedInAs: 'وارد شده با',
  panelsSupported: 'پنل های پشتیبانی شده',
};

export const overviewMessages: Messages<Key> = { en, fa };
