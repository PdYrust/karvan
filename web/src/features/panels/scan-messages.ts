import type { Translate } from '../../i18n/core';
import type { Messages } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';

// Toasts for scanning and checking panels. Kept apart from the pages' words,
// which load with the pages, because the switcher and Ctrl+K scan too.
const en = {
  scanFound: {
    one: 'Found {count} panel on this server.',
    other: 'Found {count} panels on this server.',
  },
  scanNone: 'No Marzban or PasarGuard found on this server.',
  scanNoneUsable: 'No panel on this server that Karvan can use yet.',
  scanUnusable: {
    one: "{count} folder can't be used yet. The Panels page says why.",
    other: "{count} folders can't be used yet. The Panels page says why.",
  },
  scanFailed: "Couldn't scan this server. Try again.",
  checkFailed: "Couldn't check the panels again. Try again.",
};

type Key = keyof typeof en;

const fa: Messages<Key>['fa'] = {
  scanFound: {
    one: '{count} پنل روی این سرور پیدا شد.',
    other: '{count} پنل روی این سرور پیدا شد.',
  },
  scanNone: 'مرزبان یا پاسارگاردی روی این سرور پیدا نشد.',
  scanNoneUsable: 'هنوز پنلی روی این سرور نیست که کاروان بتواند از آن استفاده کند.',
  scanUnusable: {
    one: '{count} پوشه هنوز قابل استفاده نیست. دلیلش در صفحه پنل ها آمده است.',
    other: '{count} پوشه هنوز قابل استفاده نیستند. دلیلش در صفحه پنل ها آمده است.',
  },
  scanFailed: 'جستجوی این سرور انجام نشد. دوباره امتحان کنید.',
  checkFailed: 'بررسی دوباره پنل ها انجام نشد. دوباره امتحان کنید.',
};

export const scanMessages: Messages<Key> = { en, fa };

/**
 * The toast a finished scan shows: the panels it found, and a second line
 * for folders it saw but could not use (id 0).
 */
export function scanToast(
  found: readonly Pick<Panel, 'id'>[],
  t: Translate<Key>,
): { title: string; description?: string } {
  const usable = found.filter((p) => p.id > 0).length;
  const unusable = found.length - usable;
  return {
    title:
      usable > 0
        ? t('scanFound', { count: usable })
        : t(unusable > 0 ? 'scanNoneUsable' : 'scanNone'),
    description: unusable > 0 ? t('scanUnusable', { count: unusable }) : undefined,
  };
}
