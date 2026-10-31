import type { Translate } from '../../i18n/core';
import type { Messages } from '../../i18n/locale';
import { errorOf } from './errors';

/**
 * Sentences for the codes only the HTTPS settings tab meets. They
 * load with that tab; errors.ts keeps the codes every page can meet.
 */
const en = {
  https_invalid: "Karvan can't use these HTTPS settings. Check the fields.",
  https_invalid_unknown: 'Pick one of the registered domains.',
  https_invalid_no_certificate:
    'This domain has no certificate for its name. Give it one on the Domains page first.',
  port_busy: 'Another program uses this port. Pick another one.',
  https_port_mixed:
    "A shared port must be the port of an HTTPS rule for this domain. Pick that port, or use the panel's own port.",
};

export type HttpsErrorKey = keyof typeof en;

const fa: Messages<HttpsErrorKey>['fa'] = {
  https_invalid: 'کاروان نمیتواند از این تنظیمات HTTPS استفاده کند. فیلدها را بررسی کنید.',
  https_invalid_unknown: 'یکی از دامنه های ثبت شده را انتخاب کنید.',
  https_invalid_no_certificate:
    'این دامنه برای نام خود گواهی ندارد. اول در صفحه دامنه ها برای آن گواهی بگیرید.',
  port_busy: 'برنامه دیگری از این پورت استفاده میکند. پورت دیگری انتخاب کنید.',
  https_port_mixed:
    'پورت مشترک باید پورت یک قانون HTTPS برای همین دامنه باشد. همان پورت را انتخاب کنید یا از پورت خود پنل استفاده کنید.',
};

export const httpsErrorMessages: Messages<HttpsErrorKey> = { en, fa };

/** Every code the server sends that this list words, for the words test. */
export const httpsCodes: readonly string[] = ['https_invalid', 'port_busy', 'port_mixed'];

const reasonKeys: Readonly<Record<string, HttpsErrorKey>> = {
  unknown: 'https_invalid_unknown',
  no_certificate: 'https_invalid_no_certificate',
};

/** The sentence a code and its extras get here, or undefined for another code. */
export function httpsErrorKey(
  code: string,
  detail: Readonly<Record<string, unknown>> = {},
): HttpsErrorKey | undefined {
  switch (code) {
    case 'https_invalid': {
      const { field, reason } = detail;
      if (field === 'domainId' && typeof reason === 'string' && Object.hasOwn(reasonKeys, reason))
        return reasonKeys[reason];
      return 'https_invalid';
    }
    case 'port_busy':
      return 'port_busy';
    case 'port_mixed':
      return 'https_port_mixed';
  }
  return undefined;
}

/** The sentence for a failed HTTPS request: this list's own codes, else the shared sentence. */
export function httpsErrorText(
  t: Translate<HttpsErrorKey>,
  shared: (err: unknown) => string,
  err: unknown,
): string {
  const { code, detail } = errorOf(err);
  const key = httpsErrorKey(code, detail);
  return key ? t(key) : shared(err);
}
