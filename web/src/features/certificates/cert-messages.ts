import type { Messages } from '../../i18n/locale';
/** The words of the certificate fields a domain and a stored certificate share. */
const en = {
  mode_files: 'Files on this server',
  mode_files_hint: 'A renewed certificate is picked up without a restart.',
  mode_upload: 'Upload files',
  mode_upload_hint: 'Read in this browser and kept by Karvan.',
  mode_paste: 'Paste',
  mode_paste_hint: 'The certificate and its private key as text.',
  mode_none: 'None',
  mode_none_hint: 'Plain HTTP.',
  noneWarning:
    'Old links then go over plain HTTP, and anyone on the way can read them. Use a certificate if the old links were HTTPS.',
  certFileLabel: 'Certificate file',
  certFileHint: 'Full path, such as `/etc/letsencrypt/live/example.com/fullchain.pem`',
  keyFileLabel: 'Private key file',
  certPemLabel: 'Certificate',
  keyPemLabel: 'Private key',
  keyKeep: 'Leave empty to keep the saved key.',
  keyHidden: 'Karvan never shows a saved key again.',
  chooseCert: 'Choose the certificate',
  chooseKey: 'Choose the private key',
  noFile: 'No file chosen',
  readFailed: "Couldn't read this file.",
};
export type CertFieldKey = keyof typeof en;
const fa: Messages<CertFieldKey>['fa'] = {
  mode_files: 'فایل ها روی همین سرور',
  mode_files_hint: 'گواهی تمدید شده بدون راه اندازی دوباره خوانده میشود.',
  mode_upload: 'بارگذاری فایل ها',
  mode_upload_hint: 'در همین مرورگر خوانده میشود و کاروان نگهش میدارد.',
  mode_paste: 'چسباندن',
  mode_paste_hint: 'متن گواهی و کلید خصوصی آن.',
  mode_none: 'هیچ کدام',
  mode_none_hint: 'HTTP ساده.',
  noneWarning:
    'در این حالت لینک های قدیمی با HTTP ساده میروند و هر کسی در مسیر میتواند آنها را بخواند. اگر لینک های قدیمی HTTPS بودند، از گواهی استفاده کنید.',
  certFileLabel: 'فایل گواهی',
  certFileHint: 'مسیر کامل، مثلا `/etc/letsencrypt/live/example.com/fullchain.pem`',
  keyFileLabel: 'فایل کلید خصوصی',
  certPemLabel: 'گواهی',
  keyPemLabel: 'کلید خصوصی',
  keyKeep: 'برای نگه داشتن کلید ذخیره شده خالی بگذارید.',
  keyHidden: 'کاروان کلید ذخیره شده را دیگر نشان نمیدهد.',
  chooseCert: 'انتخاب گواهی',
  chooseKey: 'انتخاب کلید خصوصی',
  noFile: 'فایلی انتخاب نشده',
  readFailed: 'این فایل خوانده نشد.',
};
export const certFieldMessages: Messages<CertFieldKey> = { en, fa };
