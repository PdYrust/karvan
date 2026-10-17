import type { Messages } from '../../i18n/locale';

const en = {
  scanFound: {
    one: 'Found {count} panel on this server.',
    other: 'Found {count} panels on this server.',
  },
  scanNone: 'No Marzban or PasarGuard found on this server.',
  scanFailed: "Couldn't scan this server. Try again.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  scanFound: {
    one: '{count} پنل روی این سرور پیدا شد.',
    other: '{count} پنل روی این سرور پیدا شد.',
  },
  scanNone: 'مرزبان یا پاسارگاردی روی این سرور پیدا نشد.',
  scanFailed: 'جستجوی این سرور انجام نشد. دوباره امتحان کنید.',
};

export const panelsMessages: Messages<Key> = { en, fa };
