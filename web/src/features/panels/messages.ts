import type { Messages } from '../../i18n/locale';

const en = {
  scanned: 'Checked again. Nothing yet.',
  scanFailed: "Couldn't finish the scan. Check the connection and try again.",
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  scanned: 'دوباره گشتم. هنوز پنلی نیست.',
  scanFailed: 'گشتن تموم نشد. اتصال رو چک کن و دوباره امتحان کن.',
};

export const panelsMessages: Messages<Key> = { en, fa };
