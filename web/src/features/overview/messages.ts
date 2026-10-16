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
  twoStep: 'Two-step sign-in',
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
  loadFailed: 'نمای کلی لود نشد.',
  panelsTitle: 'پنل ها',
  panelsEmptyTitle: 'هنوز پنلی پیدا نشد',
  scanAgain: 'دوباره بگرد',
  safeModeTitle: 'حالت امن',
  on: 'روشن',
  off: 'خاموش',
  twoStep: 'ورود دو مرحله ای',
  previousSignIn: 'ورود قبلی',
  firstSignIn: 'اولین بار',
  accountTitle: 'حساب',
  signedInAs: 'وارد شدی با',
  panelsSupported: 'با این پنل ها کار میکنه',
};

export const overviewMessages: Messages<Key> = { en, fa };
