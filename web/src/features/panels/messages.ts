import type { Messages } from '../../i18n/locale';

const en = {
  scanned: 'Checked again. Nothing yet.',
  scanFailed: "Couldn't finish the scan. Check the connection and try again.",
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  scanned: 'دوباره بررسی شد. پنلی پیدا نشد.',
  scanFailed: 'جستجو کامل نشد. اتصال را بررسی کنید و دوباره امتحان کنید.',
};

export const panelsMessages: Messages<Key> = { en, fa };
