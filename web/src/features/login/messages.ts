import type { Messages } from '../../i18n/locale';

const en = {
  pageTitle: 'Sign in · Karvan',
  title: 'Sign in to Karvan',
  username: 'Username',
  password: 'Password',
  showPassword: 'Show password',
  hidePassword: 'Hide password',
  submit: 'Sign in',
  submitting: 'Signing in…',
  totpTitle: 'Two-step code',
  totpHint: 'Open your authenticator app and enter the 6-digit code for Karvan.',
  totpCode: 'Code',
  totpSubmit: 'Verify and sign in',
  totpBack: 'Use a different account',
  errorRequired: 'Enter your username and password.',
  errorInvalid: 'The username or password is wrong.',
  errorTotpInvalid:
    'That code didn’t work. Codes change every 30 seconds, so enter the one on screen now.',
  errorLocked: 'Too many failed tries. You can try again in {time}.',
  lostAccess: 'Lost your password? On the server, run',
  sceneMove: 'Move',
  sceneMerge: 'Merge',
  moveFromMarzban: 'Every user goes from Marzban to PasarGuard, and Marzban can be removed.',
  moveFromPasarguard: 'Every user goes from PasarGuard to Marzban, and PasarGuard can be removed.',
  mergeFromMarzban: 'Marzban users are added to PasarGuard. Both panels stay.',
  mergeFromPasarguard: 'PasarGuard users are added to Marzban. Both panels stay.',
  tagline: 'Move your users to a new panel. Their configs keep working.',
  taglineDetail:
    'Marzban or PasarGuard, in either direction. UUIDs and passwords stay the same, so nobody sets up their app again.',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  pageTitle: 'ورود · کاروان',
  title: 'ورود به کاروان',
  username: 'نام کاربری',
  password: 'رمز عبور',
  showPassword: 'نمایش رمز',
  hidePassword: 'پنهان کردن رمز',
  submit: 'ورود',
  submitting: 'در حال ورود…',
  totpTitle: 'کد ورود دومرحله‌ای',
  totpHint: 'برنامهٔ احراز هویت را باز کنید و کد ۶ رقمی کاروان را وارد کنید.',
  totpCode: 'کد',
  totpSubmit: 'تأیید و ورود',
  totpBack: 'ورود با حساب دیگر',
  errorRequired: 'نام کاربری و رمز عبور را وارد کنید.',
  errorInvalid: 'نام کاربری یا رمز عبور درست نیست.',
  errorTotpInvalid:
    'این کد پذیرفته نشد. کدها هر ۳۰ ثانیه عوض می‌شوند؛ کدی را که الان می‌بینید وارد کنید.',
  errorLocked: 'تلاش ناموفق زیاد بود. {time} دیگر دوباره امتحان کنید.',
  lostAccess: 'رمز را فراموش کرده‌اید؟ روی سرور اجرا کنید',
  sceneMove: 'انتقال',
  sceneMerge: 'ادغام',
  moveFromMarzban: 'همهٔ کاربران از مرزبان به پاسارگارد می‌روند و مرزبان را می‌شود برداشت.',
  moveFromPasarguard: 'همهٔ کاربران از پاسارگارد به مرزبان می‌روند و پاسارگارد را می‌شود برداشت.',
  mergeFromMarzban: 'کاربران مرزبان به پاسارگارد اضافه می‌شوند و هر دو پنل می‌مانند.',
  mergeFromPasarguard: 'کاربران پاسارگارد به مرزبان اضافه می‌شوند و هر دو پنل می‌مانند.',
  tagline: 'کاربران را به پنل تازه ببرید. کانفیگ‌هایشان همان‌طور کار می‌کند.',
  taglineDetail:
    'از مرزبان یا پاسارگارد، به هر سمت. UUID و رمزها عوض نمی‌شوند، پس کسی لازم نیست برنامه‌اش را از نو تنظیم کند.',
};

export const loginMessages: Messages<Key> = { en, fa };
