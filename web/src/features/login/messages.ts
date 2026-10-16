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
  password: 'رمز',
  showPassword: 'نشون دادن رمز',
  hidePassword: 'مخفی کردن رمز',
  submit: 'ورود',
  submitting: 'در حال ورود...',
  totpTitle: 'کد دو مرحله ای',
  totpHint: 'کد ۶ رقمی اپ احراز هویت رو بزن.',
  totpCode: 'کد',
  totpSubmit: 'ورود',
  totpBack: 'ورود با یه حساب دیگه',
  errorRequired: 'نام کاربری و رمز رو بنویس.',
  errorInvalid: 'نام کاربری یا رمز اشتباهه.',
  errorTotpInvalid: 'این کد نشد. کد جدید رو بزن.',
  errorLocked: 'زیادی اشتباه زدی. {time} دیگه دوباره امتحان کن.',
  lostAccess: 'رمز یادت رفته؟ اینو رو سرور بزن:',
  tagline: 'کاربرهات رو ببر رو یه پنل دیگه. کانفیگشون عوض نمیشه.',
};

export const loginMessages: Messages<Key> = { en, fa };
