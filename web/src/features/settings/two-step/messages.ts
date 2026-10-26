import type { Messages } from '../../../i18n/locale';

const en = {
  title: 'Two-step sign-in',
  hintOff:
    'After the password, Karvan also asks for a 6-digit code from an authenticator app on your phone.',
  hintOn:
    'Karvan asks for a code from your authenticator app at each sign-in and before changes to the login or the address.',
  on: 'On',
  off: 'Off',
  setUp: 'Set up',
  turnOff: 'Turn off',
  lostApp: 'Lost the app? Run `karvan reset -totp` on the server to turn it off.',
  setupTitle: 'Set up two-step sign-in',
  scan: 'Scan this code with the authenticator app on your phone.',
  qrLabel: 'QR code for your authenticator app',
  key: 'Or enter this key',
  enterCode: 'Enter the 6-digit code the app shows.',
  turnOn: 'Turn on',
  turnedOn: 'Two-step sign-in is on.',
  turnOffTitle: 'Turn off two-step sign-in?',
  turnOffBody:
    'Signing in will ask for the password only. Enter your password and a code from the app.',
  turnedOff: 'Two-step sign-in is off.',
  loadFailed: "Couldn't load two-step sign-in.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  title: 'ورود دومرحله ای',
  hintOff: 'کاروان بعد از رمز، یک کد ۶ رقمی هم از برنامه احراز هویت گوشی شما میخواهد.',
  hintOn: 'کاروان در هر ورود و پیش از تغییر ورود یا آدرس، کد برنامه احراز هویت را میخواهد.',
  on: 'روشن',
  off: 'خاموش',
  setUp: 'راه اندازی',
  turnOff: 'خاموش کردن',
  lostApp:
    'برنامه را از دست داده اید؟ برای خاموش کردن، روی سرور `karvan reset -totp` را اجرا کنید.',
  setupTitle: 'راه اندازی ورود دومرحله ای',
  scan: 'این کد را با برنامه احراز هویت گوشی خود اسکن کنید.',
  qrLabel: 'کد QR برای برنامه احراز هویت',
  key: 'یا این کلید را وارد کنید',
  enterCode: 'کد ۶ رقمی را که برنامه نشان میدهد وارد کنید.',
  turnOn: 'روشن کردن',
  turnedOn: 'ورود دومرحله ای روشن شد.',
  turnOffTitle: 'ورود دومرحله ای خاموش شود؟',
  turnOffBody: 'ورود فقط رمز را میخواهد. رمز خود و یک کد از برنامه را وارد کنید.',
  turnedOff: 'ورود دومرحله ای خاموش شد.',
  loadFailed: 'وضعیت ورود دومرحله ای بارگذاری نشد.',
};

export const twoStepMessages: Messages<Key> = { en, fa };
