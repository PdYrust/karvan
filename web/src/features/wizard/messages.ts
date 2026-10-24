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
};

export const wizardMessages: Messages<WizardKey> = { en, fa };
