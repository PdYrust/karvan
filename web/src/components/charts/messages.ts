import type { Messages } from '../../i18n/locale';

const en = {
  noValue: 'No data',
  time: 'Time',
  moreSeries: '+{count} more',
  changeUp: 'Up {by} in {within}',
  changeDown: 'Down {by} in {within}',
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  noValue: 'بدون داده',
  time: 'زمان',
  moreSeries: '+{count} مورد دیگر',
  changeUp: '{by} افزایش در {within}',
  changeDown: '{by} کاهش در {within}',
};

export const chartMessages: Messages<Key> = { en, fa };
