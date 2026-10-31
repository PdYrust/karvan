import type { Messages } from '../../i18n/locale';

const en = {
  notFound: "This domain doesn't exist anymore.",
  loadFailed: "Couldn't load the domain.",
  certificate: 'Certificate',
  noCertificate: 'No certificate. Rules answer this domain over plain HTTP.',
  problems: 'Problems',
  rules: 'Rules',
  noRules: 'No rule uses this domain.',
  details: 'Certificate details',
  name: 'Name',
  origin: 'Origin',
  origin_acme: "Let's Encrypt",
  origin_file: 'Files on this server',
  origin_pasted: 'Pasted',
  origin_found: 'Found on this server',
  certFile: 'Certificate file',
  keyFile: 'Key file',
  renewal: 'Renewal',
  nextRenewal: 'Next renewal',
  method: 'Method',
  lastError: 'Last error',
  none: 'None',
  panel: 'Karvan panel',
  panelServes: 'The Karvan panel is served with this certificate.',
  panelNot: "The Karvan panel doesn't use this domain.",
};

export type DomainPageKey = keyof typeof en;

const fa: Messages<DomainPageKey>['fa'] = {
  notFound: 'این دامنه دیگر وجود ندارد.',
  loadFailed: 'دامنه بارگذاری نشد.',
  certificate: 'گواهی',
  noCertificate: 'گواهی ندارد. قانون ها با HTTP ساده به این دامنه پاسخ میدهند.',
  problems: 'مشکل ها',
  rules: 'قانون ها',
  noRules: 'هیچ قانونی از این دامنه استفاده نمیکند.',
  details: 'جزئیات گواهی',
  name: 'نام',
  origin: 'منبع',
  origin_acme: "Let's Encrypt",
  origin_file: 'فایل ها روی همین سرور',
  origin_pasted: 'جای گذاری شده',
  origin_found: 'پیدا شده روی همین سرور',
  certFile: 'فایل گواهی',
  keyFile: 'فایل کلید',
  renewal: 'تمدید',
  nextRenewal: 'تمدید بعدی',
  method: 'روش',
  lastError: 'خطای آخر',
  none: 'هیچ',
  panel: 'پنل کاروان',
  panelServes: 'پنل کاروان با همین گواهی باز میشود.',
  panelNot: 'پنل کاروان از این دامنه استفاده نمیکند.',
};

export const domainPageMessages: Messages<DomainPageKey> = { en, fa };
