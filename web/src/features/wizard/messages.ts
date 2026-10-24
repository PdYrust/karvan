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
  stepSoon: 'This step opens here soon.',
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
  stepSoon: 'این مرحله به زودی همینجا باز میشود.',
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
};

export const wizardMessages: Messages<WizardKey> = { en, fa };
