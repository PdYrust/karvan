import type { Messages } from '../../i18n/locale';

const en = {
  eyebrow: 'Migrations',
  title: 'New migration',
  stepsLabel: 'Migration steps',
  stepOf: 'Step {n} of {total}: {name}',
  stepConnect: 'Connect',
  stepSummary: 'Summary',
  stepCompare: 'Compare',
  stepChanges: 'Changes',
  stepPreview: 'Preview',
  stepStart: 'Start',
  intro:
    'Choose where the users go, then sign in to the old panel. Nothing changes on either panel until you start.',
  jobFailed: "This migration didn't load.",
  back: 'Back',
  continue: 'Continue',
  notChecked:
    "Karvan hasn't read the old panel for this migration yet. Start a new migration to read it.",
  conflictTitle: 'This plan changed somewhere else',
  conflictBody:
    'Another tab changed this migration, or Karvan read the panels again, while you made your last change. It was not saved.',
  conflictReplay: 'Apply my change again',
  conflictKeep: 'Keep the saved plan',
  saveFailed: "Your last change wasn't saved.",
  search: 'Search users',
  usersFailed: "The users didn't load.",
  noMatch: 'No user matches this search.',
  leftOut: 'Left out',
  impactLabel: 'What this plan does',
  moving: 'Move',
  skipped: 'Left out',
  renamed: 'Renamed',
  configChanged: 'Configs change',
  saving: 'Saving',
  blockers: {
    one: '{count} problem needs a fix before you start.',
    other: '{count} problems need a fix before you start.',
  },
  noBlockers: 'Nothing stands in the way.',
};

export type WizardKey = keyof typeof en;

const fa: Messages<WizardKey>['fa'] = {
  eyebrow: 'انتقال ها',
  title: 'انتقال جدید',
  stepsLabel: 'مراحل انتقال',
  stepOf: 'مرحله {n} از {total}: {name}',
  stepConnect: 'اتصال',
  stepSummary: 'خلاصه',
  stepCompare: 'مقایسه',
  stepChanges: 'تغییرات',
  stepPreview: 'پیش نمایش',
  stepStart: 'شروع',
  intro:
    'پنل مقصد را انتخاب کنید و وارد پنل قدیمی شوید. تا وقتی شروع نکنید، در هیچ کدام از دو پنل چیزی تغییر نمیکند.',
  jobFailed: 'این انتقال بارگذاری نشد.',
  back: 'قبلی',
  continue: 'ادامه',
  notChecked:
    'کاروان هنوز پنل قدیمی این انتقال را نخوانده است. برای خواندن آن، انتقال جدیدی شروع کنید.',
  conflictTitle: 'این برنامه جای دیگری تغییر کرد',
  conflictBody:
    'وقتی آخرین تغییر را میدادید، زبانه دیگری این انتقال را تغییر داد یا کاروان پنل ها را دوباره خواند. تغییر شما ذخیره نشد.',
  conflictReplay: 'تغییر من دوباره اعمال شود',
  conflictKeep: 'برنامه ذخیره شده بماند',
  saveFailed: 'آخرین تغییر شما ذخیره نشد.',
  search: 'جستجوی کاربرها',
  usersFailed: 'کاربرها بارگذاری نشدند.',
  noMatch: 'هیچ کاربری با این جستجو پیدا نشد.',
  leftOut: 'منتقل نمیشود',
  impactLabel: 'نتیجه این برنامه',
  moving: 'منتقل میشوند',
  skipped: 'منتقل نمیشوند',
  renamed: 'تغییر نام',
  configChanged: 'تغییر کانفیگ',
  saving: 'در حال ذخیره',
  blockers: {
    one: '{count} مشکل پیش از شروع باید رفع شود.',
    other: '{count} مشکل پیش از شروع باید رفع شوند.',
  },
  noBlockers: 'مانعی برای شروع نیست.',
};

export const wizardMessages: Messages<WizardKey> = { en, fa };
