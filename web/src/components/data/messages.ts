import type { Messages } from '../../i18n/locale';
const en = {
  filter: 'Filter',
  filtersOn: { one: '{count} filter on', other: '{count} filters on' },
  clearFilters: 'Clear filters',
  sortBy: 'Sort by',
  showAll: 'Show all',
  sortNewest: 'Newest first',
  sortName: 'Name',
  sortLargest: 'Largest first',
};
export type DataKey = keyof typeof en;
const fa: Messages<DataKey>['fa'] = {
  filter: 'فیلتر',
  filtersOn: { one: '{count} فیلتر فعال', other: '{count} فیلتر فعال' },
  clearFilters: 'پاک کردن فیلترها',
  sortBy: 'مرتب سازی',
  showAll: 'نمایش همه',
  sortNewest: 'جدیدترین',
  sortName: 'نام',
  sortLargest: 'بزرگترین',
};
/** The list toolbar's own words, shared by every list. */
export const dataMessages: Messages<DataKey> = { en, fa };
