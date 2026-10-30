import type { Messages } from '../../../i18n/locale';

const en = {
  addressTitle: 'Panel address',
  pathTitle: 'Hidden path',
  pathHint:
    'The secret part of the address. Without it, the server answers 404. A new one ends old bookmarks.',
  newPath: 'New hidden path',
  portTitle: 'Port',
  portHint: 'Karvan answers on port {port}.',
  changePort: 'Change port',
  pathDialogTitle: 'Make a new hidden path?',
  pathDialogBody:
    'Karvan serves the new address next to this one for 10 minutes. Nothing changes until you confirm it from there.',
  makePath: 'Make a new path',
  portDialogTitle: 'Change the port?',
  portDialogBody:
    'Karvan serves the new port next to this one for 10 minutes. Nothing changes until you confirm it from there.',
  portChoice: 'New port',
  portRandom: 'A random free port from 20000 to 60000',
  portTyped: 'A port I choose',
  portNumber: 'Port number',
  portInvalid: 'Use a number from 1 to 65535 other than {port}.',
  firewall:
    'Open the new port in the server firewall, and in any proxy or CDN in front of Karvan, before you confirm. Otherwise the panel cannot be reached on it.',
  openPort: 'Open the new port',
  waitingTitle: 'New address waiting',
  waitingBody:
    'Karvan serves it next to this one until {time}. Open it in a new tab and confirm there. If you do not, nothing changes.',
  timeLeft: '{time} left',
  openNew: 'Open the new address',
  newAddress: 'New address',
  signInThere: 'Sign in there first: this session does not carry over to the new address.',
  pathShownOnce:
    'The new path was shown once, on the page that started the change. Start again to get a new one.',
  confirmTitle: 'Confirm the new address',
  confirmBody:
    'You opened Karvan at its new address. Confirm it to keep it, and the old one stops.',
  confirm: 'Confirm this address',
  confirmed: 'The new address is saved. Bookmark it, since the old one no longer opens Karvan.',
  loginTitle: 'Operator login',
  usernameTitle: 'Username',
  usernameHint: 'You sign in as {name}.',
  newLogin: 'New username and password',
  loginDialogTitle: 'Make a new username and password?',
  loginDialogBody:
    'Karvan makes both at random and shows them once. Every other browser is signed out; this one stays signed in.',
  makeLogin: 'Make new ones',
  loginShownTitle: 'Your new username and password',
  loginShownNote: 'Other browsers were signed out. Use these the next time you sign in.',
  username: 'Username',
  password: 'Password',
  loadFailed: "Couldn't load the panel address.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  addressTitle: 'آدرس پنل',
  pathTitle: 'مسیر مخفی',
  pathHint:
    'بخش مخفی آدرس. بدون آن، سرور خطای 404 میدهد. مسیر جدید نشانک های قبلی را از کار میاندازد.',
  newPath: 'مسیر مخفی جدید',
  portTitle: 'پورت',
  portHint: 'کاروان روی پورت {port} جواب میدهد.',
  changePort: 'تغییر پورت',
  pathDialogTitle: 'مسیر مخفی جدید ساخته شود؟',
  pathDialogBody:
    'کاروان آدرس جدید را ۱۰ دقیقه کنار همین آدرس باز نگه میدارد. تا وقتی از آنجا تایید نکنید چیزی عوض نمیشود.',
  makePath: 'ساخت مسیر جدید',
  portDialogTitle: 'پورت عوض شود؟',
  portDialogBody:
    'کاروان پورت جدید را ۱۰ دقیقه کنار همین پورت باز نگه میدارد. تا وقتی از آنجا تایید نکنید چیزی عوض نمیشود.',
  portChoice: 'پورت جدید',
  portRandom: 'یک پورت آزاد تصادفی از ۲۰۰۰۰ تا ۶۰۰۰۰',
  portTyped: 'پورتی که خودم انتخاب میکنم',
  portNumber: 'شماره پورت',
  portInvalid: 'عددی از ۱ تا ۶۵۵۳۵ به جز {port} وارد کنید.',
  firewall:
    'پیش از تایید، پورت جدید را در فایروال سرور و در هر پراکسی یا CDN جلوی کاروان باز کنید. وگرنه پنل روی آن در دسترس نیست.',
  openPort: 'باز کردن پورت جدید',
  waitingTitle: 'آدرس جدید در انتظار تایید',
  waitingBody:
    'کاروان تا ساعت {time} آن را کنار همین آدرس باز نگه میدارد. آن را در زبانه جدید باز کنید و همانجا تایید کنید. اگر تایید نکنید چیزی عوض نمیشود.',
  timeLeft: '{time} مانده',
  openNew: 'باز کردن آدرس جدید',
  newAddress: 'آدرس جدید',
  signInThere: 'اول آنجا وارد شوید: این نشست به آدرس جدید منتقل نمیشود.',
  pathShownOnce:
    'مسیر جدید فقط یک بار، در صفحه ای که تغییر را شروع کرد نشان داده شد. برای گرفتن مسیر تازه دوباره شروع کنید.',
  confirmTitle: 'تایید آدرس جدید',
  confirmBody:
    'کاروان را از آدرس جدیدش باز کرده اید. برای نگه داشتن آن تایید کنید تا آدرس قبلی بسته شود.',
  confirm: 'تایید این آدرس',
  confirmed: 'آدرس جدید ذخیره شد. آن را نشانک کنید، چون آدرس قبلی دیگر کاروان را باز نمیکند.',
  loginTitle: 'ورود اپراتور',
  usernameTitle: 'نام کاربری',
  usernameHint: 'شما با نام {name} وارد میشوید.',
  newLogin: 'نام کاربری و رمز جدید',
  loginDialogTitle: 'نام کاربری و رمز جدید ساخته شود؟',
  loginDialogBody:
    'کاروان هر دو را تصادفی میسازد و فقط یک بار نشان میدهد. همه مرورگرهای دیگر خارج میشوند و این مرورگر وارد میماند.',
  makeLogin: 'ساخت نام کاربری و رمز',
  loginShownTitle: 'نام کاربری و رمز جدید شما',
  loginShownNote: 'مرورگرهای دیگر خارج شدند. دفعه بعد با اینها وارد شوید.',
  username: 'نام کاربری',
  password: 'رمز',
  loadFailed: 'آدرس پنل بارگذاری نشد.',
};

export const accessMessages: Messages<Key> = { en, fa };
