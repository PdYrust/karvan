import type { Messages } from '../../i18n/locale';

/**
 * Who the Karvan domain lets in, on its row and in the overview's banner. Apart from
 * karvan-messages because the overview loads with the app.
 */
const en = {
  waiting: 'Waits for a first sign-in through it',
  fallback_expired:
    "The Karvan domain's certificate has ended, so the server's IP opens the panel again.",
  fallback_missing:
    "Karvan can't read the Karvan domain's certificate, so the server's IP opens the panel again.",
  endsSoon: "The Karvan domain's certificate ends on {date} and does not renew by itself.",
  openDomain: 'Open the domain',
};

export type AccessKey = keyof typeof en;

const fa: Messages<AccessKey>['fa'] = {
  waiting: 'منتظر اولین ورود از راه همین دامنه',
  fallback_expired: 'اعتبار گواهی دامنه کاروان تمام شده است، پس IP سرور دوباره پنل را باز میکند.',
  fallback_missing:
    'کاروان نمیتواند گواهی دامنه کاروان را بخواند، پس IP سرور دوباره پنل را باز میکند.',
  endsSoon: 'اعتبار گواهی دامنه کاروان در {date} تمام میشود و خودکار تمدید نمیشود.',
  openDomain: 'باز کردن دامنه',
};

export const accessMessages: Messages<AccessKey> = { en, fa };
