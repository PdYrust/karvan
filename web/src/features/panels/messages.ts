import type { Messages } from '../../i18n/locale';

const en = {
  scanned: 'Checked again. No panels yet.',
  scanFailed: 'The scan didn’t finish. Check the connection and try again.',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  scanned: 'دوباره بررسی شد. هنوز پنلی نیست.',
  scanFailed: 'جست‌وجو تمام نشد. اتصال را بررسی کنید و دوباره امتحان کنید.',
};

export const panelsMessages: Messages<Key> = { en, fa };
