import type { Messages } from '../../i18n/locale';
import type { PanelControl } from '../../lib/api/panels';

/**
 * Whether Karvan can stop and start a panel (to put a whole Safe Mode
 * backup back), and if not, why and how to allow it. Each reason has a
 * short label for tables and a sentence with its fix for detail pages.
 */
const en = {
  available: 'Available',
  availableHint: 'Karvan can stop and start this panel, so Safe Mode can put a whole backup back.',
  unavailable: 'Not available',
  unavailableHint: "Karvan can't stop and start this panel.",
  undoOnly: 'Until then, Safe Mode can only remove what a migration wrote.',
  turned_off: 'Turned off',
  turned_off_hint: "Panel control is turned off in Karvan's config file.",
  turned_off_fix: 'Set `panelControl` to `auto` in that file and restart Karvan.',
  no_compose_file: 'No compose file',
  no_compose_file_hint: "There's no compose file next to this panel's `.env`.",
  no_compose_file_fix:
    "Karvan stops and starts a panel through its `docker-compose.yml`. The panel's official installer puts that file next to `.env`.",
  no_panel_service: 'No panel service',
  no_panel_service_hint: 'The compose file next to `.env` runs no Marzban or PasarGuard image.',
  no_panel_service_fix:
    "Karvan stops only services that run `gozargah/marzban` or `pasarguard/panel`. Use the panel's official image in that file.",
  docker_denied: 'No Docker access',
  docker_denied_hint: "Karvan isn't allowed to run `docker compose` for this panel.",
  docker_denied_fix: "Give Karvan's user a `sudo` rule for this panel's `docker compose` commands.",
};

export type ControlKey = keyof typeof en;

const fa: Messages<ControlKey>['fa'] = {
  available: 'در دسترس',
  availableHint:
    'کاروان میتواند این پنل را متوقف و دوباره روشن کند، پس حالت امن میتواند پشتیبان کامل را برگرداند.',
  unavailable: 'در دسترس نیست',
  unavailableHint: 'کاروان نمیتواند این پنل را متوقف و دوباره روشن کند.',
  undoOnly: 'تا آن موقع حالت امن فقط میتواند چیزهایی را که انتقال نوشته حذف کند.',
  turned_off: 'خاموش شده',
  turned_off_hint: 'کنترل پنل در فایل تنظیمات کاروان خاموش است.',
  turned_off_fix:
    'در همان فایل مقدار `panelControl` را `auto` بگذارید و کاروان را دوباره راه اندازی کنید.',
  no_compose_file: 'بدون فایل compose',
  no_compose_file_hint: 'کنار فایل `.env` این پنل فایل compose نیست.',
  no_compose_file_fix:
    'کاروان پنل را با فایل `docker-compose.yml` آن متوقف و روشن میکند. نصب کننده رسمی پنل این فایل را کنار `.env` میگذارد.',
  no_panel_service: 'بدون سرویس پنل',
  no_panel_service_hint: 'فایل compose کنار `.env` هیچ سرویسی با ایمیج مرزبان یا پاسارگارد ندارد.',
  no_panel_service_fix:
    'کاروان فقط سرویس هایی را متوقف میکند که `gozargah/marzban` یا `pasarguard/panel` را اجرا میکنند. در همان فایل از ایمیج رسمی پنل استفاده کنید.',
  docker_denied: 'بدون دسترسی به Docker',
  docker_denied_hint: 'کاروان اجازه اجرای `docker compose` را برای این پنل ندارد.',
  docker_denied_fix:
    'برای کاربر کاروان یک قانون `sudo` برای دستورهای `docker compose` این پنل اضافه کنید.',
};

export const controlMessages: Messages<ControlKey> = { en, fa };

const reasons = new Set(['turned_off', 'no_compose_file', 'no_panel_service', 'docker_denied']);

export interface ControlText {
  label: ControlKey;
  hint: ControlKey;
  fix?: ControlKey;
}

/** The words for a panel's control state; a reason this list does not know gets the general ones. */
export function controlText(c: PanelControl): ControlText {
  if (c.available) return { label: 'available', hint: 'availableHint' };
  if (!c.reason || !reasons.has(c.reason)) return { label: 'unavailable', hint: 'unavailableHint' };
  const r = c.reason as ControlKey;
  return { label: r, hint: `${r}_hint` as ControlKey, fix: `${r}_fix` as ControlKey };
}
