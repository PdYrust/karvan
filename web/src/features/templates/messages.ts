import type { Messages } from '../../i18n/locale';

const en = {
  title: 'Templates',
  description: 'Saved choices to use again in a migration. Apply one in the wizard, at Changes.',
  list: 'Saved templates',
  saved: 'Saved {time}',
  doesNothing: 'This template changes nothing.',
  problem: 'Users with "{problem}"',
  fix: '{problem}: {fix}',
  reset_estimate: 'Next traffic reset: estimated from the old panel',
  reset_restart: 'Next traffic reset: counted again from the move',
  delete: 'Delete',
  deleteNamed: 'Delete {name}',
  deleteTitle: 'Delete this template',
  deleteBody: 'Migrations that used it keep their plans. The template itself is gone for good.',
  deleteFailed: "Couldn't delete the template.",
  deleted: 'Template deleted',
  emptyTitle: 'No templates yet',
  emptyBody: 'Save the choices of a migration as a template at its Changes step, or when it ends.',
  newMigration: 'New migration',
  loadFailed: "Couldn't load the templates.",
};

export type TemplatesKey = keyof typeof en;

const fa: Messages<TemplatesKey>['fa'] = {
  title: 'الگوها',
  description:
    'انتخاب های ذخیره شده برای استفاده دوباره در انتقال. الگو را در مرحله تغییرها به کار ببرید.',
  list: 'الگوهای ذخیره شده',
  saved: 'ذخیره شده {time}',
  doesNothing: 'این الگو چیزی را تغییر نمیدهد.',
  problem: 'کاربرهای با مشکل "{problem}"',
  fix: '{problem}: {fix}',
  reset_estimate: 'ریست بعدی حجم: تخمین از پنل قدیمی',
  reset_restart: 'ریست بعدی حجم: شمارش دوباره از زمان انتقال',
  delete: 'حذف',
  deleteNamed: 'حذف {name}',
  deleteTitle: 'حذف این الگو',
  deleteBody:
    'انتقال هایی که از آن استفاده کردند برنامه خود را نگه میدارند. خود الگو برای همیشه حذف میشود.',
  deleteFailed: 'الگو حذف نشد.',
  deleted: 'الگو حذف شد',
  emptyTitle: 'هنوز الگویی نیست',
  emptyBody: 'انتخاب های یک انتقال را در مرحله تغییرها یا وقتی تمام شد به عنوان الگو ذخیره کنید.',
  newMigration: 'انتقال جدید',
  loadFailed: 'الگوها خوانده نشدند.',
};

export const templatesMessages: Messages<TemplatesKey> = { en, fa };
