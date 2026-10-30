import type { Messages } from '../../i18n/locale';
import type { Finding } from '../../lib/api/panels';

/**
 * What each panel finding means and how to fix it, for every page that
 * shows one. Any finding keeps Karvan from using the panel.
 */
const en = {
  env_unreadable: "Karvan can't read this panel's `.env` file.",
  env_unreadable_fix: 'Make sure the file exists and Karvan is allowed to read it.',
  db_url_missing: "This panel's `.env` file has no database address.",
  db_url_missing_fix: 'Add `SQLALCHEMY_DATABASE_URL` to it.',
  db_relative_path: "The database path in `.env` is relative, so Karvan can't find the file.",
  db_relative_path_fix: "Use the file's full path, starting with `/`.",
  db_unsupported: "Karvan doesn't support this type of database.",
  db_unsupported_fix: 'Karvan works with SQLite, MySQL, MariaDB and PostgreSQL.',
  db_unreachable: "Karvan can't connect to this database.",
  db_unreachable_fix: 'Check that the database is running and the address in `.env` is right.',
  db_not_writable: "Karvan can't write to this panel's database file.",
  db_not_writable_fix:
    "Let Karvan's user write to the file and its folder. SQLite keeps its `-wal` and `-shm` files there.",
  panel_copy: 'This folder is a copy of another panel and uses the same database.',
  panel_copy_fix:
    'A migration, batch or rule still uses it. Remove this panel once nothing needs it.',
  not_a_panel: "This database doesn't look like a Marzban or PasarGuard database.",
  not_a_panel_fix: "Check that the address in `.env` points to the panel's own database.",
  unknown_revision: "Karvan doesn't support this version of the panel.",
  unknown_revision_fix: 'Karvan works with Marzban 0.8.4 and PasarGuard 5.4.1.',
  schema_mismatch: "This panel's tables don't match its version.",
  schema_mismatch_fix:
    'Karvan writes only to an unmodified panel database. Check whether an add-on or a manual change altered it.',
  unknown_children: "A table Karvan doesn't know points to this panel's users or admins.",
  unknown_children_fix:
    "Karvan can't safely undo a move while it is there. Remove the add-on that created it.",
  xray_config_unreadable: "Karvan can't read this Marzban's Xray config.",
  xray_config_unreadable_fix:
    'Check that `XRAY_JSON` in `.env` points to the config file and that the file is valid JSON.',
  cert_unreadable: "Karvan can't open the panel's TLS certificate or key.",
  cert_unreadable_fix:
    'Check the paths in `UVICORN_SSL_CERTFILE` and `UVICORN_SSL_KEYFILE`, and that Karvan may read the files.',
  path_relative: "A path in this panel's `.env` is relative, so Karvan can't find the file.",
  path_relative_fix: 'Write the full path in `.env`, starting with `/`.',
};

export type FindingKey = keyof typeof en;

const fa: Messages<FindingKey>['fa'] = {
  env_unreadable: 'کاروان نمیتواند فایل `.env` این پنل را بخواند.',
  env_unreadable_fix: 'مطمئن شوید فایل وجود دارد و کاروان اجازه خواندن آن را دارد.',
  db_url_missing: 'فایل `.env` این پنل آدرس دیتابیس ندارد.',
  db_url_missing_fix: 'مقدار `SQLALCHEMY_DATABASE_URL` را به آن اضافه کنید.',
  db_relative_path: 'مسیر دیتابیس در `.env` نسبی است و کاروان فایل را پیدا نمیکند.',
  db_relative_path_fix: 'مسیر کامل فایل را بنویسید که با `/` شروع شود.',
  db_unsupported: 'کاروان از این نوع دیتابیس پشتیبانی نمیکند.',
  db_unsupported_fix: 'کاروان با SQLite، MySQL، MariaDB و PostgreSQL کار میکند.',
  db_unreachable: 'کاروان به این دیتابیس وصل نمیشود.',
  db_unreachable_fix: 'بررسی کنید دیتابیس روشن باشد و آدرس آن در `.env` درست باشد.',
  db_not_writable: 'کاروان نمیتواند در فایل دیتابیس این پنل بنویسد.',
  db_not_writable_fix:
    'به کاربر کاروان اجازه نوشتن در این فایل و پوشه آن را بدهید. SQLite فایل های `-wal` و `-shm` را همانجا نگه میدارد.',
  panel_copy: 'این پوشه کپی پنل دیگری است و از همان دیتابیس استفاده میکند.',
  panel_copy_fix:
    'هنوز یک انتقال، دسته یا قانون از آن استفاده میکند. وقتی دیگر لازم نبود، این پنل را حذف کنید.',
  not_a_panel: 'این دیتابیس، دیتابیس مرزبان یا پاسارگارد نیست.',
  not_a_panel_fix: 'بررسی کنید آدرس دیتابیس در `.env` به دیتابیس خود پنل اشاره کند.',
  unknown_revision: 'کاروان از این نسخه پنل پشتیبانی نمیکند.',
  unknown_revision_fix: 'کاروان با مرزبان 0.8.4 و پاسارگارد 5.4.1 کار میکند.',
  schema_mismatch: 'جدول های دیتابیس این پنل با نسخه آن مطابقت ندارند.',
  schema_mismatch_fix:
    'کاروان فقط در دیتابیس دست نخورده پنل مینویسد. بررسی کنید افزونه ای یا تغییری دستی آن را عوض نکرده باشد.',
  unknown_children: 'جدولی که کاروان نمیشناسد به کاربرها یا ادمین های این پنل اشاره میکند.',
  unknown_children_fix:
    'تا وقتی این جدول هست، کاروان نمیتواند انتقال را با اطمینان برگرداند. افزونه ای را که آن را ساخته حذف کنید.',
  xray_config_unreadable: 'کاروان نمیتواند کانفیگ Xray این مرزبان را بخواند.',
  xray_config_unreadable_fix:
    'بررسی کنید `XRAY_JSON` در `.env` به فایل کانفیگ اشاره کند و فایل JSON درستی باشد.',
  cert_unreadable: 'کاروان نمیتواند گواهی یا کلید TLS پنل را باز کند.',
  cert_unreadable_fix:
    'مسیرهای `UVICORN_SSL_CERTFILE` و `UVICORN_SSL_KEYFILE` را بررسی کنید و مطمئن شوید کاروان اجازه خواندن فایل ها را دارد.',
  path_relative: 'یکی از مسیرهای `.env` این پنل نسبی است و کاروان فایل را پیدا نمیکند.',
  path_relative_fix: 'مسیر کامل را در `.env` بنویسید که با `/` شروع شود.',
};

export const findingMessages: Messages<FindingKey> = { en, fa };

const known = new Set<string>(Object.keys(en).filter((k) => !k.endsWith('_fix')));

/** The sentence and its fix for a finding code; undefined for a code this list does not know. */
export function findingText(code: string): { what: FindingKey; fix: FindingKey } | undefined {
  if (!known.has(code)) return undefined;
  return { what: code as FindingKey, fix: `${code}_fix` as FindingKey };
}

// These details name something (a panel, a revision, a table, a path); the
// others hold raw error text, which never reaches the screen.
const named = new Set(['panel_copy', 'unknown_revision', 'unknown_children', 'path_relative']);

/** The machine value a finding is about, when its detail is one. */
export function findingDetail(f: Finding): string | undefined {
  return named.has(f.code) && f.detail ? f.detail : undefined;
}
