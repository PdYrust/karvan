import type { Messages } from '../../../i18n/locale';

const en = {
  password: 'Current password',
  code: 'Code from your authenticator app',
  showPassword: 'Show password',
  hidePassword: 'Hide password',
  wrongPassword: "That password isn't right.",
  savedThem: 'I saved them',
  shownOnce: 'Karvan shows these only once. Save them somewhere safe before you close this.',
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  password: 'رمز فعلی',
  code: 'کد برنامه احراز هویت',
  showPassword: 'نمایش رمز',
  hidePassword: 'پنهان کردن رمز',
  wrongPassword: 'این رمز درست نیست.',
  savedThem: 'ذخیره کردم',
  shownOnce: 'کاروان اینها را فقط یک بار نشان میدهد. پیش از بستن، آنها را جای امنی ذخیره کنید.',
};

export const securityMessages: Messages<Key> = { en, fa };
