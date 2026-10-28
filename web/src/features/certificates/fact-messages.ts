import type { Messages } from '../../i18n/locale';

/** What Karvan read from a certificate and its problems; the rule form shows them too. */
const en = {
  certCovers: 'Covers',
  certIssuer: 'Issued by',
  certEnds: 'Ends',
  certFine: 'No problems found.',
  warn_unreadable: "Karvan can't read this certificate.",
  warn_no_key: 'Karvan has no private key it can read for it.',
  warn_key_mismatch: "The private key doesn't belong to this certificate.",
  warn_domain_mismatch: "It doesn't cover every domain that uses it.",
  warn_expired: 'It has ended. Renew it.',
  warn_not_yet_valid: "It isn't valid yet.",
  warn_expires_soon: 'It ends within 14 days. Renew it soon.',
};

export type CertFactKey = keyof typeof en;

const fa: Messages<CertFactKey>['fa'] = {
  certCovers: 'دامنه های گواهی',
  certIssuer: 'صادرکننده',
  certEnds: 'پایان اعتبار',
  certFine: 'مشکلی پیدا نشد.',
  warn_unreadable: 'کاروان نمیتواند این گواهی را بخواند.',
  warn_no_key: 'کاروان کلید خصوصی خوانایی برای آن ندارد.',
  warn_key_mismatch: 'کلید خصوصی مال این گواهی نیست.',
  warn_domain_mismatch: 'همه دامنه هایی را که از آن استفاده میکنند پوشش نمیدهد.',
  warn_expired: 'اعتبارش تمام شده است. آن را تمدید کنید.',
  warn_not_yet_valid: 'هنوز معتبر نشده است.',
  warn_expires_soon: 'تا ۱۴ روز دیگر اعتبارش تمام میشود. زودتر تمدیدش کنید.',
};

export const certFactMessages: Messages<CertFactKey> = { en, fa };
