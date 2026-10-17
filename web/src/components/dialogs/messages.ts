import type { Messages } from '../../i18n/locale';

const en = {
  close: 'Close',
  cancel: 'Cancel',
  typeToConfirm: 'Type this to confirm',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  close: 'بستن',
  cancel: 'انصراف',
  typeToConfirm: 'برای تایید این را بنویسید',
};

export const dialogMessages: Messages<Key> = { en, fa };
