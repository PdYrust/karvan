import type { Messages } from '../../i18n/locale';

const en = {
  title: 'HTTPS',
  loadFailed: "Couldn't load the HTTPS settings.",
  plainWarning:
    'The panel is served over plain HTTP, so your password and session cross the network unencrypted. Turn on HTTPS below.',
  modeTitle: 'Connection',
  modeOff: 'Plain HTTP on port {port}.',
  modeSelf: 'HTTPS with a self-signed certificate on port {port}.',
  modeDomain: 'HTTPS with the certificate of {domain} on port {port}.',
  modeShared: 'HTTPS with the certificate of {domain} on port {port}, shared with the rules there.',
  change: 'Change',
  fingerprintTitle: 'Certificate fingerprint',
  fingerprintHint:
    'Browsers warn about a self-signed certificate. Go on only when the SHA-256 fingerprint the browser shows is this one.',
  fingerprint: 'SHA-256 fingerprint',
  dialogTitle: 'Change how the panel is served?',
  dialogBody:
    'Karvan serves the new address next to this one for 10 minutes. Nothing changes until you confirm it from there.',
  modeChoice: 'Connection',
  off: 'Plain HTTP',
  offHint: 'Not encrypted. Use it only behind a proxy of your own that adds HTTPS.',
  self: 'HTTPS with a self-signed certificate',
  selfHint: 'Encrypted, for an address with no domain. Browsers show a warning first.',
  domain: 'HTTPS with a domain',
  domainHint: 'Uses the certificate of a registered domain.',
  domainLabel: 'Domain',
  pickDomain: 'Pick a domain',
  noDomains:
    'No domain has a certificate for its name yet. Give one a certificate on the Domains page.',
  whereChoice: 'Port',
  whereKeep: 'Keep port {port}',
  whereTyped: 'A port I choose',
  whereShared: 'The port of an HTTPS rule for this domain',
  portNumber: 'Port number',
  portInvalid: 'Use a number from 1 to 65535.',
  rulePort: 'Rule port',
  action: 'Serve the new address',
  signInThere: 'Sign in there first: this session does not carry over to the new address.',
};

export type HttpsKey = keyof typeof en;

const fa: Messages<HttpsKey>['fa'] = {
  title: 'HTTPS',
  loadFailed: 'تنظیمات HTTPS بارگذاری نشد.',
  plainWarning:
    'پنل روی HTTP ساده باز میشود و رمز و نشست شما بدون رمزنگاری از شبکه میگذرد. HTTPS را در همین صفحه روشن کنید.',
  modeTitle: 'اتصال',
  modeOff: 'HTTP ساده روی پورت {port}.',
  modeSelf: 'HTTPS با گواهی خودامضا روی پورت {port}.',
  modeDomain: 'HTTPS با گواهی {domain} روی پورت {port}.',
  modeShared: 'HTTPS با گواهی {domain} روی پورت {port}، مشترک با قانون های همان پورت.',
  change: 'تغییر',
  fingerprintTitle: 'اثر انگشت گواهی',
  fingerprintHint:
    'مرورگرها برای گواهی خودامضا هشدار میدهند. فقط وقتی ادامه دهید که اثر انگشت SHA-256 در مرورگر همین باشد.',
  fingerprint: 'اثر انگشت SHA-256',
  dialogTitle: 'روش باز شدن پنل عوض شود؟',
  dialogBody:
    'کاروان آدرس جدید را ۱۰ دقیقه کنار همین آدرس باز نگه میدارد. تا وقتی از آنجا تایید نکنید چیزی عوض نمیشود.',
  modeChoice: 'اتصال',
  off: 'HTTP ساده',
  offHint: 'رمزنگاری ندارد. فقط پشت پراکسی خودتان که HTTPS اضافه میکند از آن استفاده کنید.',
  self: 'HTTPS با گواهی خودامضا',
  selfHint: 'رمزنگاری دارد و برای آدرس بدون دامنه است. مرورگرها اول هشدار نشان میدهند.',
  domain: 'HTTPS با دامنه',
  domainHint: 'از گواهی یک دامنه ثبت شده استفاده میکند.',
  domainLabel: 'دامنه',
  pickDomain: 'انتخاب دامنه',
  noDomains: 'هنوز هیچ دامنه ای برای نام خود گواهی ندارد. در صفحه دامنه ها به یکی گواهی بدهید.',
  whereChoice: 'پورت',
  whereKeep: 'همان پورت {port}',
  whereTyped: 'پورتی که خودم انتخاب میکنم',
  whereShared: 'پورت یک قانون HTTPS برای همین دامنه',
  portNumber: 'شماره پورت',
  portInvalid: 'عددی از ۱ تا ۶۵۵۳۵ وارد کنید.',
  rulePort: 'پورت قانون',
  action: 'باز کردن آدرس جدید',
  signInThere: 'اول آنجا وارد شوید: این نشست به آدرس جدید منتقل نمیشود.',
};

export const httpsMessages: Messages<HttpsKey> = { en, fa };
