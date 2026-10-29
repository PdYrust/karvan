import type { Messages } from '../../i18n/locale';

const en = {
  newTitle: 'Add a rule',
  editTitle: 'Edit {name}',
  description: 'Users who still have an old link reach the new panel through this rule.',
  nameLabel: 'Name',
  nameHint: 'Only for you, such as the old domain.',
  oldTitle: 'Old panel',
  oldHint: 'The panel the users moved from.',
  kindLabel: 'Kind',
  urlLabel: 'Address',
  urlHint: 'The address the migration used, such as `https://old.example.com:8443`',
  ipLabel: 'Its IP address',
  ipHint:
    'Once the old domain points here, Karvan reaches the old panel by this address. Lazy checks need it.',
  pinLabel: 'Certificate fingerprint',
  pinHint: 'Optional: the `SHA-256` of its certificate. Karvan then trusts only that certificate.',
  answerTitle: 'What this rule answers',
  pathLabel: 'Old subscription path',
  pathHint: 'The part before the token, such as `sub` in `/sub/token`.',
  domainsLabel: 'Domains',
  domainsHint: 'One per line. `*.example.com` covers one level. Empty: every domain on this port.',
  portLabel: 'Port',
  lazyLabel: 'Lazy checks',
  lazyHint: 'An old link Karvan has not seen is checked with the old panel once.',
  plainLabel: 'The old panel is plain HTTP',
  plainHint:
    'Anyone on the way could answer for the old panel. Turn this on only on a network you trust.',
  cancel: 'Cancel',
  save: 'Save',
  addSite: 'Add rule',
  saved: 'Rule saved',
  added: 'Rule added',
  saveFailed: "Couldn't save the rule.",
  missing: "This rule doesn't exist anymore.",
  field_name: 'Give the rule a name of up to 64 characters.',
  field_panelId: 'Choose a destination panel.',
  field_sourceKind: 'Choose the kind of the old panel.',
  field_sourceUrl: 'Enter the old panel address with `http://` or `https://`, without a username.',
  field_sourceIp:
    'Enter an IP address Karvan can reach. Lazy checks need it when the address is a name.',
  field_sourcePin: 'The fingerprint is 64 letters and digits, with or without colons.',
  field_oldPath: 'Use letters, digits, `-` and `_` only, such as `sub`.',
  field_domains:
    'Each domain is a name such as `old.example.com` or `*.example.com`, or an IP address.',
  field_port: 'Use a port from 1 to 65535.',
};

export type SiteFormKey = keyof typeof en;

const fa: Messages<SiteFormKey>['fa'] = {
  newTitle: 'افزودن قانون',
  editTitle: 'ویرایش {name}',
  description: 'کاربرهایی که هنوز لینک قدیمی دارند از راه این قانون به پنل جدید میرسند.',
  nameLabel: 'نام',
  nameHint: 'فقط برای خودتان، مثلا دامنه قدیمی.',
  oldTitle: 'پنل قدیمی',
  oldHint: 'پنلی که کاربرها از آن منتقل شدند.',
  kindLabel: 'نوع',
  urlLabel: 'آدرس',
  urlHint: 'آدرسی که انتقال با آن انجام شد، مثلا `https://old.example.com:8443`',
  ipLabel: 'آدرس IP آن',
  ipHint:
    'وقتی دامنه قدیمی به اینجا اشاره کند، کاروان پنل قدیمی را با این آدرس پیدا میکند. بررسی هنگام درخواست به آن نیاز دارد.',
  pinLabel: 'اثر انگشت گواهی',
  pinHint: 'اختیاری: `SHA-256` گواهی آن. کاروان آن وقت فقط به همان گواهی اعتماد میکند.',
  answerTitle: 'این قانون به چه چیزی پاسخ میدهد',
  pathLabel: 'مسیر اشتراک قدیمی',
  pathHint: 'بخش پیش از توکن، مثلا `sub` در `/sub/token`.',
  domainsLabel: 'دامنه ها',
  domainsHint:
    'هر خط یک دامنه. `*.example.com` یک سطح را پوشش میدهد. خالی: همه دامنه های این پورت.',
  portLabel: 'پورت',
  lazyLabel: 'بررسی هنگام درخواست',
  lazyHint: 'لینک قدیمی ای که کاروان ندیده است یک بار با پنل قدیمی بررسی میشود.',
  plainLabel: 'پنل قدیمی HTTP ساده است',
  plainHint:
    'هر کسی در مسیر میتواند به جای پنل قدیمی جواب بدهد. فقط در شبکه ای که به آن اعتماد دارید روشنش کنید.',
  cancel: 'انصراف',
  save: 'ذخیره',
  addSite: 'افزودن قانون',
  saved: 'قانون ذخیره شد',
  added: 'قانون اضافه شد',
  saveFailed: 'قانون ذخیره نشد.',
  missing: 'این قانون دیگر وجود ندارد.',
  field_name: 'برای قانون نامی تا ۶۴ نویسه بگذارید.',
  field_panelId: 'پنل مقصد را انتخاب کنید.',
  field_sourceKind: 'نوع پنل قدیمی را انتخاب کنید.',
  field_sourceUrl: 'آدرس پنل قدیمی را با `http://` یا `https://` و بدون نام کاربری وارد کنید.',
  field_sourceIp:
    'آدرس IP ای وارد کنید که کاروان به آن دسترسی دارد. وقتی آدرس پنل یک نام است، بررسی هنگام درخواست به آن نیاز دارد.',
  field_sourcePin: 'اثر انگشت ۶۴ حرف و رقم است، با دونقطه یا بدون آن.',
  field_oldPath: 'فقط حرف، رقم، `-` و `_` به کار ببرید، مثلا `sub`.',
  field_domains: 'هر دامنه یک نام مثل `old.example.com` یا `*.example.com`، یا یک آدرس IP است.',
  field_port: 'پورتی از ۱ تا ۶۵۵۳۵ به کار ببرید.',
};

export const siteFormMessages: Messages<SiteFormKey> = { en, fa };
