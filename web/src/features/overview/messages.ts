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
  activityTitle: 'Recent activity',
  activityEmpty: 'Nothing has happened yet.',
  activityShowing: { one: 'Showing the latest event', other: 'Showing the latest {count} events' },
  colEvent: 'Event',
  colFrom: 'From',
  colWhen: 'When',
  accountTitle: 'Account',
  signedInAs: 'Signed in as',
  panelsSupported: 'Supported panels',
  'act.login.success': 'Signed in',
  'act.login.failure': 'A sign-in failed',
  'act.logout.success': 'Signed out',
  'act.lockout.success': 'Sign-in paused after too many failed tries',
  'act.totp.enable.success': 'Turned two-step sign-in on',
  'act.totp.disable.success': 'Turned two-step sign-in off',
  'act.totp.disable.failure': 'A try to turn two-step sign-in off failed',
  'act.cli.init.success': 'Karvan was set up on this server',
  'act.cli.reset.success': 'Sign-in details were reset from the command line',
  'act.unknown': 'Something changed ({action})',
  fromCommandLine: 'Command line',
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
  activityTitle: 'رویدادهای اخیر',
  activityEmpty: 'هنوز رویدادی ثبت نشده.',
  activityShowing: { one: 'آخرین رویداد', other: '{count} رویداد آخر' },
  colEvent: 'رویداد',
  colFrom: 'از',
  colWhen: 'زمان',
  accountTitle: 'حساب',
  signedInAs: 'وارد شده با',
  panelsSupported: 'پنل‌های پشتیبانی‌شده',
  'act.login.success': 'ورود انجام شد',
  'act.login.failure': 'یک ورود ناموفق بود',
  'act.logout.success': 'خروج انجام شد',
  'act.lockout.success': 'ورود پس از تلاش‌های ناموفق زیاد موقتاً بسته شد',
  'act.totp.enable.success': 'ورود دومرحله‌ای روشن شد',
  'act.totp.disable.success': 'ورود دومرحله‌ای خاموش شد',
  'act.totp.disable.failure': 'تلاش برای خاموش کردن ورود دومرحله‌ای ناموفق بود',
  'act.cli.init.success': 'کاروان روی این سرور نصب شد',
  'act.cli.reset.success': 'اطلاعات ورود از خط فرمان بازنشانی شد',
  'act.unknown': 'تغییری ثبت شد ({action})',
  fromCommandLine: 'خط فرمان',
};

export const overviewMessages: Messages<Key> = { en, fa };
export type OverviewKey = Key;
