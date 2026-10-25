import type { Messages } from '../../../i18n/locale';

const en = {
  title: 'Audit log',
  description: 'Who did what in Karvan, and from where. Newest first.',
  caption: 'Audit log',
  action: 'Action',
  outcome: 'Outcome',
  allActions: 'All actions',
  allOutcomes: 'All outcomes',
  success: 'Succeeded',
  failure: 'Failed',
  when: 'When',
  what: 'What',
  who: 'Who',
  ip: 'IP',
  loadMore: 'Load more',
  shown: { one: 'Showing {count} event', other: 'Showing {count} events' },
  emptyTitle: 'No events yet',
  emptyBody: 'Sign-ins, migrations and changes to settings show up here.',
  noMatchTitle: 'No events match',
  noMatchBody: 'Try another action or outcome.',
  clearFilters: 'Clear filters',
  loadFailed: "Couldn't load the audit log.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  title: 'گزارش رویدادها',
  description: 'چه کسی در کاروان چه کاری کرده و از کجا. جدیدترین ها اول.',
  caption: 'گزارش رویدادها',
  action: 'کار',
  outcome: 'نتیجه',
  allActions: 'همه کارها',
  allOutcomes: 'همه نتیجه ها',
  success: 'موفق',
  failure: 'ناموفق',
  when: 'زمان',
  what: 'کار',
  who: 'چه کسی',
  ip: 'IP',
  loadMore: 'نمایش بیشتر',
  shown: { one: 'نمایش {count} رویداد', other: 'نمایش {count} رویداد' },
  emptyTitle: 'هنوز رویدادی نیست',
  emptyBody: 'ورودها، انتقال ها و تغییر تنظیمات اینجا نشان داده میشوند.',
  noMatchTitle: 'رویدادی پیدا نشد',
  noMatchBody: 'کار یا نتیجه دیگری را امتحان کنید.',
  clearFilters: 'پاک کردن فیلترها',
  loadFailed: 'گزارش رویدادها بارگذاری نشد.',
};

export const auditMessages: Messages<Key> = { en, fa };
