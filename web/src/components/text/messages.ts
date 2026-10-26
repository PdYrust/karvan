import type { Messages } from '../../i18n/locale';

const en = {
  copy: 'Copy',
  copied: 'Copied',
  copyBlocked: 'The browser blocked copying. The text is selected for you to copy.',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  copy: 'کپی',
  copied: 'کپی شد',
  copyBlocked: 'مرورگر اجازه کپی نداد. متن انتخاب شده تا خودتان کپی کنید.',
};

export const textMessages: Messages<Key> = { en, fa };
