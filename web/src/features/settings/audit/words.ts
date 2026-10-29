import type { Messages } from '../../../i18n/locale';

/** What each audited action did, in words (internal/audit/audit.go). */
const en = {
  login: 'Signed in',
  logout: 'Signed out',
  lockout: 'Sign-in locked',
  'totp.enable': 'Turned on two-step sign-in',
  'totp.disable': 'Turned off two-step sign-in',
  'cli.init': 'Set up Karvan on the server',
  'cli.reset': 'Reset the login on the server',
  'panels.scan': 'Looked for panels',
  'migration.run': 'Started a migration',
  'migration.rollback': 'Rolled back a migration',
  'migration.cancel': 'Stopped a migration',
  'migration.delete': 'Deleted a draft',
  'migration.passwords_download': 'Downloaded new passwords',
  'safe_mode.on': 'Turned on Safe Mode',
  'safe_mode.off': 'Turned off Safe Mode',
  'safe_mode.rollback': 'Changed what Safe Mode does after a failure',
  'safe_mode.backup_delete': 'Deleted a backup',
  'safe_mode.backup_restore': 'Put a backup back',
  'cli.safe_mode_restore': 'Put a backup back on the server',
  'legacy_site.save': 'Saved a rule',
  'legacy_site.delete': 'Deleted a rule',
  'legacy_site.tokens': 'Uploaded old links',
  'batch.rename': 'Renamed a batch',
  'batch.delete': 'Deleted a batch',
  'batch.reactivate': 'Made a batch active again',
  'batch.show_link': "Showed a user's new link",
  'domain.add': 'Added a domain',
  'domain.delete': 'Deleted a domain',
  'domain.certificate': "Changed a domain's certificate",
  'certificate.add': 'Added a certificate',
  'certificate.save': 'Changed a certificate',
  'certificate.delete': 'Deleted a certificate',
  'certificate.scan': 'Looked for certificates',
  'redirect.save': 'Saved a redirect rule',
  'redirect.delete': 'Deleted a redirect rule',
  'sessions.end': 'Signed out sessions',
  'account.credentials': 'Made a new username and password',
  'settings.address_start': 'Started an address change',
  'settings.address': 'Changed the panel address',
  'settings.https': 'Started an HTTPS change',
};

export type AuditAction = keyof typeof en;

const fa: Messages<AuditAction>['fa'] = {
  login: 'ورود',
  logout: 'خروج',
  lockout: 'قفل شدن ورود',
  'totp.enable': 'روشن کردن ورود دومرحله ای',
  'totp.disable': 'خاموش کردن ورود دومرحله ای',
  'cli.init': 'راه اندازی کاروان روی سرور',
  'cli.reset': 'بازنشانی ورود روی سرور',
  'panels.scan': 'جستجوی پنل ها',
  'migration.run': 'شروع انتقال',
  'migration.rollback': 'برگرداندن انتقال',
  'migration.cancel': 'توقف انتقال',
  'migration.delete': 'حذف پیش نویس',
  'migration.passwords_download': 'دانلود رمزهای جدید',
  'safe_mode.on': 'روشن کردن حالت امن',
  'safe_mode.off': 'خاموش کردن حالت امن',
  'safe_mode.rollback': 'تغییر کار حالت امن بعد از شکست',
  'safe_mode.backup_delete': 'حذف پشتیبان',
  'safe_mode.backup_restore': 'برگرداندن پشتیبان',
  'cli.safe_mode_restore': 'برگرداندن پشتیبان روی سرور',
  'legacy_site.save': 'ذخیره قانون',
  'legacy_site.delete': 'حذف قانون',
  'legacy_site.tokens': 'بارگذاری لینک های قدیمی',
  'batch.rename': 'تغییر نام دسته',
  'batch.delete': 'حذف دسته',
  'batch.reactivate': 'فعال کردن دوباره دسته',
  'batch.show_link': 'نمایش لینک جدید کاربر',
  'domain.add': 'افزودن دامنه',
  'domain.delete': 'حذف دامنه',
  'domain.certificate': 'تغییر گواهی دامنه',
  'certificate.add': 'افزودن گواهی',
  'certificate.save': 'تغییر گواهی',
  'certificate.delete': 'حذف گواهی',
  'certificate.scan': 'جستجوی گواهی ها',
  'redirect.save': 'ذخیره قانون تغییر مسیر',
  'redirect.delete': 'حذف قانون تغییر مسیر',
  'sessions.end': 'بستن نشست ها',
  'account.credentials': 'ساخت نام کاربری و رمز جدید',
  'settings.address_start': 'شروع تغییر آدرس',
  'settings.address': 'تغییر آدرس پنل',
  'settings.https': 'شروع تغییر HTTPS',
};

export const actionMessages: Messages<AuditAction> = { en, fa };

export const auditActions = Object.keys(en) as AuditAction[];

export const isAuditAction = (a: string): a is AuditAction => Object.hasOwn(en, a);

/** Why a failed action failed, for the reasons the server records. */
const reasonsEn = {
  bad_password: 'Wrong password',
  bad_credentials: 'Wrong username or password',
  bad_totp: 'Wrong code',
  port_unavailable: "The port couldn't be opened",
  not_new_address: 'Not confirmed from the new address',
};

export type AuditReason = keyof typeof reasonsEn;

export const reasonMessages: Messages<AuditReason> = {
  en: reasonsEn,
  fa: {
    bad_password: 'رمز اشتباه',
    bad_credentials: 'نام کاربری یا رمز اشتباه',
    bad_totp: 'کد اشتباه',
    port_unavailable: 'پورت باز نشد',
    not_new_address: 'تایید از آدرس جدید نبود',
  },
};

export const isAuditReason = (r: string | undefined): r is AuditReason =>
  r !== undefined && Object.hasOwn(reasonsEn, r);
