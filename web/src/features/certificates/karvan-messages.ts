import type { Messages } from '../../i18n/locale';

/** Words of the Karvan domain: the menu item, its toasts, the list badge and its settings. */
const en = {
  useKarvan: 'Use for Karvan',
  stopKarvan: 'Stop using for Karvan',
  karvanOn: 'Karvan uses {name} now',
  karvanOn_cert: 'Its certificate serves this name.',
  karvanOn_own: "Until it has a certificate, this name gets Karvan's own.",
  karvanOn_plain: 'Until it has a certificate, this name answers plain HTTP.',
  karvanOnIp: "The server's IP keeps Karvan's own certificate.",
  karvanOnOnly: 'Once you sign in through it, the IP no longer opens the panel.',
  karvanOff: 'Karvan stopped using {name}',
  karvanOffHint: "This name and the server's IP get Karvan's own certificate.",
  karvanFailed: "Couldn't change the Karvan domain.",
  karvanBadge: 'Karvan',
  plainTitle: 'Plain HTTP for a Karvan domain without a certificate',
  plainHint:
    "Off: Karvan's own certificate until it has one. On: plain HTTP for that name only, unencrypted.",
  onlyTitle: 'Only open Karvan on its domain',
  onlyHint:
    "After your first sign-in through it, the server's IP and other names get no panel. `karvan access open-ip` on the server opens the IP again.",
};

export type KarvanKey = keyof typeof en;

const fa: Messages<KarvanKey>['fa'] = {
  useKarvan: 'استفاده برای کاروان',
  stopKarvan: 'توقف استفاده برای کاروان',
  karvanOn: 'کاروان حالا از {name} استفاده میکند',
  karvanOn_cert: 'گواهی آن برای همین نام استفاده میشود.',
  karvanOn_own: 'تا وقتی گواهی ندارد، این نام گواهی خود کاروان را میگیرد.',
  karvanOn_plain: 'تا وقتی گواهی ندارد، این نام با HTTP ساده جواب میدهد.',
  karvanOnIp: 'IP سرور گواهی خود کاروان را نگه میدارد.',
  karvanOnOnly: 'بعد از اولین ورود از راه آن، پنل دیگر با IP سرور باز نمیشود.',
  karvanOff: 'کاروان دیگر از {name} استفاده نمیکند',
  karvanOffHint: 'این نام و IP سرور گواهی خود کاروان را میگیرند.',
  karvanFailed: 'دامنه کاروان تغییر نکرد.',
  karvanBadge: 'کاروان',
  plainTitle: 'HTTP ساده برای دامنه کاروان بدون گواهی',
  plainHint:
    'خاموش: تا وقتی گواهی ندارد، گواهی خود کاروان. روشن: فقط برای همین نام HTTP ساده و بدون رمزنگاری.',
  onlyTitle: 'کاروان فقط با دامنه اش باز شود',
  onlyHint:
    'بعد از اولین ورود از راه دامنه، پنل با IP سرور و نام های دیگر باز نمیشود. `karvan access open-ip` روی سرور دوباره IP را باز میکند.',
};

export const karvanMessages: Messages<KarvanKey> = { en, fa };
