import type { Messages } from '../../i18n/locale';

const en = {
  title: 'Settings',
  description: 'How Karvan looks, keeps you safe during a move, and who can sign in.',
  tabs: 'Settings sections',
  tabGeneral: 'General',
  tabSafeMode: 'Safe Mode',
  tabAccess: 'Access',
  tabTwoStep: 'Two-step sign-in',
  tabSessions: 'Sessions',
  tabAudit: 'Audit log',
  lookTitle: 'Look and language',
  regionTitle: 'Time and calendar',
  language: 'Language',
  languageHint: 'The panel switches at once; this browser remembers it.',
  appearance: 'Appearance',
  appearanceHint: 'Same as system follows the light or dark setting of this device.',
  themeLight: 'Light',
  themeDark: 'Dark',
  themeSystem: 'Same as system',
  timeZone: 'Time zone',
  timeZoneHint: 'Times on every page are shown in this zone. It is {time} in {zone}.',
  calendar: 'Calendar',
  calendarHint: 'Dates in lists, reports and charts.',
  calendarAuto: 'Follow the language',
  calendarPersian: 'Solar Hijri (Jalali)',
  calendarGregory: 'Gregorian',
  safeModeTitle: 'Safe Mode',
  safeModeHint:
    'Before a migration writes anything, Karvan backs up the new panel. If the move fails, it is undone.',
  safeModeOn: 'On',
  safeModeOff: 'Off',
  safeModeOffWarning:
    'Safe Mode is off. A failed migration leaves what it already wrote on the new panel, and no backup is taken.',
  turnOffTitle: 'Turn off Safe Mode?',
  turnOffBody:
    'Migrations will run without a backup, and a failed one will not be undone. Turn it off only if you have your own backup of the new panel.',
  turnOffAction: 'Turn off Safe Mode',
  turnedOff: 'Safe Mode is off.',
  turnedOn: 'Safe Mode is on.',
  rollbackTitle: 'When a migration fails',
  rollbackHint: 'What Safe Mode does with the new panel after a failed move.',
  rollbackUndo: 'Remove what the migration wrote',
  rollbackUndoHint:
    'Only the moved users and the admin come out. Changes other admins made meanwhile stay. The backup goes back only if something is left.',
  rollbackBackup: 'Put the whole backup back',
  rollbackBackupHint:
    'The new panel goes back to how it was before the move, including changes others made meanwhile. Karvan stops the panel for a few seconds.',
  rollbackSaved: 'Saved.',
  controlMissing: {
    one: 'Karvan cannot stop and start {names}, so it cannot put a whole backup back there.',
    other:
      'Karvan cannot stop and start these panels, so it cannot put a whole backup back there: {names}.',
  },
  saveFailed: "Couldn't save. Try again.",
  loadFailed: "Couldn't load the Safe Mode settings.",
  retry: 'Try again',
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  title: 'تنظیمات',
  description: 'ظاهر کاروان، محافظت در هنگام انتقال و اینکه چه کسی میتواند وارد شود.',
  tabs: 'بخش های تنظیمات',
  tabGeneral: 'عمومی',
  tabSafeMode: 'حالت امن',
  tabAccess: 'دسترسی',
  tabTwoStep: 'ورود دومرحله ای',
  tabSessions: 'نشست ها',
  tabAudit: 'گزارش رویدادها',
  lookTitle: 'ظاهر و زبان',
  regionTitle: 'زمان و تقویم',
  language: 'زبان',
  languageHint: 'پنل همان لحظه عوض میشود و این مرورگر آن را به خاطر میسپارد.',
  appearance: 'ظاهر',
  appearanceHint: 'گزینه مطابق سیستم از تنظیم روشن یا تیره همین دستگاه پیروی میکند.',
  themeLight: 'روشن',
  themeDark: 'تیره',
  themeSystem: 'مطابق سیستم',
  timeZone: 'منطقه زمانی',
  timeZoneHint:
    'زمان ها در همه صفحه ها به این منطقه نشان داده میشوند. اکنون ساعت {time} به وقت {zone} است.',
  calendar: 'تقویم',
  calendarHint: 'تاریخ ها در فهرست ها، گزارش ها و نمودارها.',
  calendarAuto: 'مطابق زبان',
  calendarPersian: 'خورشیدی',
  calendarGregory: 'میلادی',
  safeModeTitle: 'حالت امن',
  safeModeHint:
    'پیش از اینکه انتقال چیزی بنویسد، کاروان از پنل جدید پشتیبان میگیرد. اگر انتقال شکست بخورد، برگردانده میشود.',
  safeModeOn: 'روشن',
  safeModeOff: 'خاموش',
  safeModeOffWarning:
    'حالت امن خاموش است. انتقالی که شکست بخورد هر چه نوشته باشد روی پنل جدید میماند و پشتیبانی هم گرفته نمیشود.',
  turnOffTitle: 'حالت امن خاموش شود؟',
  turnOffBody:
    'انتقال ها بدون پشتیبان انجام میشوند و انتقالی که شکست بخورد برگردانده نمیشود. فقط وقتی خاموشش کنید که خودتان از پنل جدید پشتیبان دارید.',
  turnOffAction: 'خاموش کردن حالت امن',
  turnedOff: 'حالت امن خاموش شد.',
  turnedOn: 'حالت امن روشن شد.',
  rollbackTitle: 'وقتی انتقال شکست بخورد',
  rollbackHint: 'کاری که حالت امن بعد از شکست انتقال با پنل جدید میکند.',
  rollbackUndo: 'حذف چیزهایی که انتقال نوشته',
  rollbackUndoHint:
    'فقط کاربرهای منتقل شده و ادمین حذف میشوند. تغییراتی که ادمین های دیگر در این مدت داده اند میمانند. پشتیبان فقط وقتی برمیگردد که چیزی باقی مانده باشد.',
  rollbackBackup: 'برگرداندن کامل پشتیبان',
  rollbackBackupHint:
    'پنل جدید به حالت پیش از انتقال برمیگردد و تغییرات دیگران در این مدت هم برمیگردند. کاروان پنل را چند ثانیه متوقف میکند.',
  rollbackSaved: 'ذخیره شد.',
  controlMissing: {
    one: 'کاروان نمیتواند {names} را متوقف و دوباره روشن کند، پس نمیتواند پشتیبان کامل را آنجا برگرداند.',
    other:
      'کاروان نمیتواند این پنل ها را متوقف و دوباره روشن کند، پس نمیتواند پشتیبان کامل را آنجا برگرداند: {names}.',
  },
  saveFailed: 'ذخیره نشد. دوباره امتحان کنید.',
  loadFailed: 'تنظیمات حالت امن بارگذاری نشد.',
  retry: 'دوباره امتحان کنید',
};

export const settingsMessages: Messages<Key> = { en, fa };
