import type { Messages } from '../../i18n/locale';

const en = {
  title: 'Backups',
  description:
    'Safe Mode backs up the new panel before a migration writes to it. Karvan keeps every backup until you delete it.',
  list: 'Safe Mode backups',
  refresh: 'Refresh',
  loadFailed: "Karvan couldn't load the backups.",
  colMigration: 'Migration',
  colPanel: 'Panel',
  colDatabase: 'Database',
  colSize: 'Size',
  colTaken: 'Taken',
  colState: 'State',
  colActions: 'Actions',
  moreFor: 'More for backup {id}',
  restore: 'Put back',
  delete: 'Delete',
  running: 'Its migration is running',
  gone: "The panel isn't registered",
  controlNotice: "Karvan can't stop and start {name}, so its backups can't be put back from here.",
  emptyTitle: 'No backups yet',
  emptyBody: 'Safe Mode takes one before each migration writes to the new panel.',
  safeModeSettings: 'Safe Mode settings',
  restoreTitle: 'Put this backup back?',
  restoreBody:
    'Karvan stops {name}, puts its database back as it was at {time} and starts it again. Everything written to it since then is lost, by this migration and by anyone else.',
  restoreFailed: "Karvan couldn't start putting it back.",
  deleteTitle: 'Delete this backup?',
  deleteBody: "The backup file is removed and can't come back.",
  deleteFailed: "Karvan couldn't delete it.",
  deleted: 'Backup deleted',
  shown: { one: 'Showing one backup', other: 'Showing {count} backups' },
};

export type BackupsKey = keyof typeof en;

const fa: Messages<BackupsKey>['fa'] = {
  title: 'پشتیبان ها',
  description:
    'حالت امن پیش از اینکه انتقال در پنل جدید بنویسد از آن پشتیبان میگیرد. کاروان هر پشتیبان را تا وقتی حذفش نکنید نگه میدارد.',
  list: 'پشتیبان های حالت امن',
  refresh: 'تازه کردن',
  loadFailed: 'کاروان نتوانست پشتیبان ها را بخواند.',
  colMigration: 'انتقال',
  colPanel: 'پنل',
  colDatabase: 'پایگاه داده',
  colSize: 'حجم',
  colTaken: 'زمان',
  colState: 'وضعیت',
  colActions: 'کارها',
  moreFor: 'گزینه های بیشتر برای پشتیبان {id}',
  restore: 'برگرداندن',
  delete: 'حذف',
  running: 'انتقال آن در حال اجراست',
  gone: 'این پنل ثبت نشده است',
  controlNotice:
    'کاروان نمیتواند {name} را متوقف و دوباره روشن کند، پس پشتیبان های آن از اینجا برنمیگردند.',
  emptyTitle: 'هنوز پشتیبانی نیست',
  emptyBody: 'حالت امن پیش از اینکه هر انتقال در پنل جدید بنویسد یک پشتیبان میگیرد.',
  safeModeSettings: 'تنظیمات حالت امن',
  restoreTitle: 'این پشتیبان برگردانده شود؟',
  restoreBody:
    'کاروان {name} را متوقف میکند، پایگاه داده آن را به وضعیت {time} برمیگرداند و دوباره روشنش میکند. هر چه از آن موقع در این پنل نوشته شده از بین میرود، چه با این انتقال و چه با هر کس دیگر.',
  restoreFailed: 'کاروان نتوانست برگرداندن را شروع کند.',
  deleteTitle: 'این پشتیبان حذف شود؟',
  deleteBody: 'فایل پشتیبان پاک میشود و دیگر برنمیگردد.',
  deleteFailed: 'کاروان نتوانست آن را حذف کند.',
  deleted: 'پشتیبان حذف شد',
  shown: { one: 'نمایش یک پشتیبان', other: 'نمایش {count} پشتیبان' },
};

export const backupsMessages: Messages<BackupsKey> = { en, fa };
