import type { Messages } from '../../../i18n/locale';

const en = {
  password: 'Current password',
  code: 'Code from your authenticator app',
  showPassword: 'Show password',
  hidePassword: 'Hide password',
  cancel: 'Cancel',
  close: 'Close',
  wrongPassword: "That password isn't right.",
  copy: 'Copy',
  copied: 'Copied',
  savedThem: 'I saved them',
  shownOnce: 'Karvan shows these only once. Save them somewhere safe before you close this.',
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  password: 'رمز فعلی',
  code: 'کد برنامه احراز هویت',
  showPassword: 'نمایش رمز',
  hidePassword: 'پنهان کردن رمز',
  cancel: 'انصراف',
  close: 'بستن',
  wrongPassword: 'این رمز درست نیست.',
  copy: 'کپی',
  copied: 'کپی شد',
  savedThem: 'ذخیره کردم',
  shownOnce: 'کاروان اینها را فقط یک بار نشان میدهد. پیش از بستن، آنها را جای امنی ذخیره کنید.',
};

export const securityMessages: Messages<Key> = { en, fa };
