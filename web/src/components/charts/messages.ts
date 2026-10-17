import type { Messages } from '../../i18n/locale';

const en = {
  noValue: 'No data',
  time: 'Time',
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  noValue: 'بدون داده',
  time: 'زمان',
};

export const chartMessages: Messages<Key> = { en, fa };
