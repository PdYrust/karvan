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
};

export const wizardMessages: Messages<WizardKey> = { en, fa };
