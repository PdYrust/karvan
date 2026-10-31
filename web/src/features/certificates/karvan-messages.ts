import type { Messages } from '../../i18n/locale';

/** Words of the Karvan domain: the menu item, its toasts, the list badge and the settings choice. */
const en = {
  useKarvan: 'Use for Karvan',
  stopKarvan: 'Stop using for Karvan',
  karvanOn: 'Karvan uses {name} now',
  karvanOn_cert: "Its certificate serves this name. The server's IP keeps Karvan's own.",
  karvanOn_own: "Until it has a certificate, this name gets Karvan's own.",
  karvanOn_plain: 'Until it has a certificate, this name answers plain HTTP.',
  karvanOff: 'Karvan stopped using {name}',
  karvanOffHint: "This name and the server's IP get Karvan's own certificate.",
  karvanFailed: "Couldn't change the Karvan domain.",
  karvanBadge: 'Karvan',
  noCertTitle: 'For the Karvan domain without a certificate',
  noCertHint: "The server's IP and other names always get Karvan's own certificate.",
  noCertOwn: "Karvan's own certificate",
  noCertOwnHint: 'HTTPS. The browser warns about it once, and the password stays encrypted.',
  noCertPlain: 'Plain HTTP',
  noCertPlainHint:
    'Only for this name, and the password crosses the network unencrypted. HTTPS on the same port keeps working.',
};

export type KarvanKey = keyof typeof en;

const fa: Messages<KarvanKey>['fa'] = {
  useKarvan: 'استفاده برای کاروان',
  stopKarvan: 'توقف استفاده برای کاروان',
  karvanOn: 'کاروان حالا از {name} استفاده میکند',
  karvanOn_cert: 'گواهی آن برای همین نام استفاده میشود. IP سرور گواهی خود کاروان را نگه میدارد.',
  karvanOn_own: 'تا وقتی گواهی ندارد، این نام گواهی خود کاروان را میگیرد.',
  karvanOn_plain: 'تا وقتی گواهی ندارد، این نام با HTTP ساده جواب میدهد.',
  karvanOff: 'کاروان دیگر از {name} استفاده نمیکند',
  karvanOffHint: 'این نام و IP سرور گواهی خود کاروان را میگیرند.',
  karvanFailed: 'دامنه کاروان تغییر نکرد.',
  karvanBadge: 'کاروان',
  noCertTitle: 'برای دامنه کاروان بدون گواهی',
  noCertHint: 'IP سرور و نام های دیگر همیشه گواهی خود کاروان را میگیرند.',
  noCertOwn: 'گواهی خود کاروان',
  noCertOwnHint: 'HTTPS. مرورگر یک بار درباره آن هشدار میدهد و رمز رمزنگاری شده میماند.',
  noCertPlain: 'HTTP ساده',
  noCertPlainHint:
    'فقط برای همین نام، و رمز بدون رمزنگاری از شبکه میگذرد. HTTPS روی همان پورت کار میکند.',
};

export const karvanMessages: Messages<KarvanKey> = { en, fa };
