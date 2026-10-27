import type { Translate } from '../../i18n/core';
import { useT, type Messages } from '../../i18n/locale';
import { errorOf, useErrorText } from './errors';

/**
 * Sentences for the codes only the Old links, Rules and Domains and
 * certificates pages meet (D58). They load with those pages; errors.ts
 * keeps the codes every page can meet.
 */
const en = {
  site_invalid: "A field of this site has a value Karvan can't use. Check the fields.",
  site_overlap: 'Another site already serves this domain and port.',
  plain_http_off: 'Plain HTTP is off for old links. Turn it on or use HTTPS.',
  port_mixed:
    'This port already has sites the other way. All sites on one port use HTTPS, or none do.',
  domain_invalid:
    'Enter a domain like sub.example.com, a wildcard like *.example.com, or an IP address.',
  domain_in_use: 'A rule still uses this domain. Remove it from the rule first.',
  certificate_invalid:
    "A field of this certificate has a value Karvan can't use. Check the fields.",
  certificate_invalid_unreadable:
    "Karvan can't read this file. Check the path and that the file is there.",
  certificate_invalid_no_cert:
    'Karvan found no certificate here. Use a PEM certificate file, or paste the certificate.',
  certificate_invalid_key_mismatch:
    "This key doesn't belong to the certificate. Use the key that was made with it.",
  certificate_in_use: 'A rule still uses this certificate. Pick another one in the rule first.',
};

export type SubscriptionErrorKey = keyof typeof en;

const fa: Messages<SubscriptionErrorKey>['fa'] = {
  site_invalid:
    'یکی از فیلدهای این سایت مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  site_overlap: 'سایت دیگری همین دامنه و پورت را دارد.',
  plain_http_off:
    'HTTP ساده برای لینک های قدیمی خاموش است. آن را روشن کنید یا از HTTPS استفاده کنید.',
  port_mixed:
    'این پورت سایت هایی به روش دیگر دارد. یا همه سایت های یک پورت HTTPS هستند یا هیچ کدام.',
  domain_invalid:
    'دامنه ای مثل sub.example.com، دامنه ستاره دار مثل *.example.com یا یک IP وارد کنید.',
  domain_in_use: 'یک قانون هنوز از این دامنه استفاده میکند. اول آن را از قانون بردارید.',
  certificate_invalid:
    'یکی از فیلدهای این گواهی مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  certificate_invalid_unreadable:
    'کاروان نمیتواند این فایل را بخواند. مسیر را بررسی کنید و مطمئن شوید فایل آنجاست.',
  certificate_invalid_no_cert:
    'کاروان اینجا گواهی پیدا نکرد. فایل گواهی PEM بدهید یا متن گواهی را بچسبانید.',
  certificate_invalid_key_mismatch:
    'این کلید برای این گواهی نیست. کلیدی را بدهید که همراه همین گواهی ساخته شده است.',
  certificate_in_use:
    'یک قانون هنوز از این گواهی استفاده میکند. اول در آن قانون گواهی دیگری انتخاب کنید.',
};

export const subscriptionErrorMessages: Messages<SubscriptionErrorKey> = { en, fa };

const reasonKeys: Readonly<Record<string, SubscriptionErrorKey | undefined>> = {
  unreadable: 'certificate_invalid_unreadable',
  no_cert: 'certificate_invalid_no_cert',
  key_mismatch: 'certificate_invalid_key_mismatch',
};

/** Every code the server sends that this list words, for the words test. */
export const subscriptionCodes = (Object.keys(en) as SubscriptionErrorKey[]).filter(
  (k) => !k.startsWith('certificate_invalid_'),
);

const isCode = (code: string): code is SubscriptionErrorKey =>
  Object.hasOwn(en, code) && !code.startsWith('certificate_invalid_');

/** The sentence a code and its extras get here, or undefined for a shared code. */
export function subscriptionErrorKey(
  code: string,
  detail: Readonly<Record<string, unknown>> = {},
): SubscriptionErrorKey | undefined {
  if (code === 'certificate_invalid' && typeof detail.reason === 'string') {
    return (Object.hasOwn(reasonKeys, detail.reason) && reasonKeys[detail.reason]) || code;
  }
  return isCode(code) ? code : undefined;
}

/** The sentence for a failed request: this list's own codes, else the shared sentence. */
export function subscriptionErrorText(
  t: Translate<SubscriptionErrorKey>,
  shared: (err: unknown) => string,
  err: unknown,
): string {
  const { code, detail } = errorOf(err);
  const key = subscriptionErrorKey(code, detail);
  return key ? t(key) : shared(err);
}

/** subscriptionErrorText in the page's language. */
export function useSubscriptionErrorText(): (err: unknown) => string {
  const t = useT(subscriptionErrorMessages);
  const shared = useErrorText();
  return (err) => subscriptionErrorText(t, shared, err);
}
