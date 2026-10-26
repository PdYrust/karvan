import type { Messages } from '../../i18n/locale';

const en = {
  pageTitle: 'Sign in | Karvan',
  title: 'Sign in to Karvan',
  username: 'Username',
  password: 'Password',
  showPassword: 'Show password',
  hidePassword: 'Hide password',
  submit: 'Sign in',
  submitting: 'Signing in...',
  totpTitle: 'Two-step code',
  totpHint: 'Enter the 6-digit code from your authenticator app.',
  totpCode: 'Code',
  totpSubmit: 'Sign in',
  totpBack: 'Use another account',
  errorRequired: 'Enter your username and password.',
  errorInvalid: 'Wrong username or password.',
  errorTotpInvalid: "That code didn't work. Try the new one.",
  errorLocked: 'Too many tries. Try again in {time}.',
  lostAccess: 'Forgot your password? Run this on the server:',
  tagline: 'Take your users to another panel. Their configs stay the same.',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  pageTitle: 'ورود | کاروان',
  title: 'ورود به کاروان',
  username: 'نام کاربری',
  password: 'رمز عبور',
  showPassword: 'نمایش رمز',
  hidePassword: 'پنهان کردن رمز',
  submit: 'ورود',
  submitting: 'در حال ورود...',
  totpTitle: 'کد ورود دومرحله ای',
  totpHint: 'کد ۶ رقمی برنامه احراز هویت را وارد کنید.',
  totpCode: 'کد',
  totpSubmit: 'ورود',
  totpBack: 'ورود با حساب دیگر',
  errorRequired: 'نام کاربری و رمز عبور را وارد کنید.',
  errorInvalid: 'نام کاربری یا رمز عبور درست نیست.',
  errorTotpInvalid: 'کد درست نیست. کد جدید را وارد کنید.',
  errorLocked: 'تعداد تلاش ها زیاد بود. {time} دیگر دوباره امتحان کنید.',
  lostAccess: 'رمز را فراموش کرده اید؟ این دستور را روی سرور اجرا کنید:',
  tagline: 'کاربران را به پنل دیگری ببرید. کانفیگ ها تغییر نمیکنند.',
};

export const loginMessages: Messages<Key> = { en, fa };
