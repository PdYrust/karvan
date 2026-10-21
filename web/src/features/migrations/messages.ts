import type { Messages } from '../../i18n/locale';

const en = {
  stateDraft: 'Draft',
  stateRunning: 'Running',
  stateDone: 'Done',
  stateFailed: 'Failed',
  stateRolledBack: 'Rolled back',
  stateInterrupted: 'Interrupted',
  move: '{from} to {to}',
  moved: '{count} moved',
  movedSkipped: '{moved} moved, {skipped} skipped',
};

export type MigrationsKey = keyof typeof en;

const fa: Messages<MigrationsKey>['fa'] = {
  stateDraft: 'پیش نویس',
  stateRunning: 'در حال اجرا',
  stateDone: 'انجام شده',
  stateFailed: 'ناموفق',
  stateRolledBack: 'برگردانده شده',
  stateInterrupted: 'نیمه تمام',
  move: 'از {from} به {to}',
  moved: '{count} کاربر منتقل شد',
  movedSkipped: '{moved} کاربر منتقل شد، {skipped} رد شد',
};

export const migrationsMessages: Messages<MigrationsKey> = { en, fa };
