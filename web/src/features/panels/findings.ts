import type { Messages } from '../../i18n/locale';

/**
 * What a panel finding means and how to fix it, for every page that shows
 * one. These six stop Karvan from reading the panel's database.
 */
const en = {
  env_unreadable: "Karvan can't read this panel's .env file.",
  env_unreadable_fix: 'Make sure the file exists and Karvan is allowed to read it.',
  db_url_missing: "This panel's .env file has no database address.",
  db_url_missing_fix: 'Add SQLALCHEMY_DATABASE_URL to it.',
  db_relative_path: 'The database path in .env is relative, so Karvan cannot find the file.',
  db_relative_path_fix: "Use the file's full path, starting with /.",
  db_unsupported: "Karvan doesn't support this type of database.",
  db_unsupported_fix: 'Karvan works with SQLite, MySQL, MariaDB and PostgreSQL.',
  db_unreachable: "Karvan can't connect to this database.",
  db_unreachable_fix: 'Check that the database is running and the address in .env is right.',
  not_a_panel: "This database doesn't look like a Marzban or PasarGuard database.",
  not_a_panel_fix: "Check that the address in .env points to the panel's own database.",
};

export type FindingKey = keyof typeof en;

const fa: Messages<FindingKey>['fa'] = {
  env_unreadable: 'کاروان نمیتواند فایل .env این پنل را بخواند.',
  env_unreadable_fix: 'مطمئن شوید فایل وجود دارد و کاروان اجازه خواندن آن را دارد.',
  db_url_missing: 'فایل .env این پنل آدرس دیتابیس ندارد.',
  db_url_missing_fix: 'مقدار SQLALCHEMY_DATABASE_URL را به آن اضافه کنید.',
  db_relative_path: 'مسیر دیتابیس در .env نسبی است و کاروان فایل را پیدا نمیکند.',
  db_relative_path_fix: 'مسیر کامل فایل را بنویسید که با / شروع شود.',
  db_unsupported: 'کاروان از این نوع دیتابیس پشتیبانی نمیکند.',
  db_unsupported_fix: 'کاروان با SQLite، MySQL، MariaDB و PostgreSQL کار میکند.',
  db_unreachable: 'کاروان به این دیتابیس وصل نمیشود.',
  db_unreachable_fix: 'بررسی کنید دیتابیس روشن باشد و آدرس آن در .env درست باشد.',
  not_a_panel: 'این دیتابیس، دیتابیس مرزبان یا پاسارگارد نیست.',
  not_a_panel_fix: 'بررسی کنید آدرس دیتابیس در .env به دیتابیس خود پنل اشاره کند.',
};

export const findingMessages: Messages<FindingKey> = { en, fa };

const known = new Set<string>(Object.keys(en).filter((k) => !k.endsWith('_fix')));

/** The sentence and its fix for a finding code; undefined for a code this list does not know. */
export function findingText(code: string): { what: FindingKey; fix: FindingKey } | undefined {
  if (!known.has(code)) return undefined;
  return { what: code as FindingKey, fix: `${code}_fix` as FindingKey };
}
