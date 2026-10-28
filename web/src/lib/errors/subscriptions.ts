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
  site_overlap: 'Another rule already answers this domain and path on this port.',
  cert_conflict: 'Rules for one domain on one port must use the same certificate.',
  port_taken: "Karvan's own panel uses this port. Pick another port.",
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
  certificate_not_adopted:
    "Add this rule's certificate in Domains and certificates, then pick it here.",
  site_invalid_unknown: 'A batch of this rule is gone. Pick the batches again.',
  site_invalid_inactive: 'A batch of this rule is not active. Pick only active batches.',
  site_invalid_mixed:
    'These batches come from different old panels or destinations. Pick batches of one old panel and one destination.',
};

export type SubscriptionErrorKey = keyof typeof en;

const fa: Messages<SubscriptionErrorKey>['fa'] = {
  site_invalid:
    'یکی از فیلدهای این سایت مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  site_overlap: 'قانون دیگری روی این پورت به همین دامنه و مسیر جواب میدهد.',
  cert_conflict: 'قانون های یک دامنه روی یک پورت باید از یک گواهی استفاده کنند.',
  port_taken: 'پنل خود کاروان از این پورت استفاده میکند. پورت دیگری انتخاب کنید.',
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
  certificate_not_adopted:
    'گواهی این قانون را در دامنه ها و گواهی ها اضافه کنید و بعد اینجا انتخاب کنید.',
  site_invalid_unknown: 'یکی از دسته های این قانون دیگر نیست. دسته ها را دوباره انتخاب کنید.',
  site_invalid_inactive: 'یکی از دسته های این قانون فعال نیست. فقط دسته های فعال را انتخاب کنید.',
  site_invalid_mixed:
    'این دسته ها از پنل های قدیمی یا مقصدهای مختلف هستند. دسته های یک پنل قدیمی و یک مقصد را انتخاب کنید.',
};

export const subscriptionErrorMessages: Messages<SubscriptionErrorKey> = { en, fa };

/** The sentences for why a field was refused, by code and reason. */
const reasonKeys: Readonly<Record<string, Readonly<Record<string, SubscriptionErrorKey>>>> = {
  certificate_invalid: {
    unreadable: 'certificate_invalid_unreadable',
    no_cert: 'certificate_invalid_no_cert',
    key_mismatch: 'certificate_invalid_key_mismatch',
  },
  site_invalid: {
    unknown: 'site_invalid_unknown',
    inactive: 'site_invalid_inactive',
    mixed: 'site_invalid_mixed',
  },
};

const isReason = (k: string) =>
  k.startsWith('certificate_invalid_') || k.startsWith('site_invalid_');

/** Every code the server sends that this list words, for the words test. */
export const subscriptionCodes = (Object.keys(en) as SubscriptionErrorKey[]).filter(
  (k) => !isReason(k),
);

const isCode = (code: string): code is SubscriptionErrorKey =>
  Object.hasOwn(en, code) && !isReason(code);

/** The sentence a code and its extras get here, or undefined for a shared code. */
export function subscriptionErrorKey(
  code: string,
  detail: Readonly<Record<string, unknown>> = {},
): SubscriptionErrorKey | undefined {
  const reasons = Object.hasOwn(reasonKeys, code) ? reasonKeys[code] : undefined;
  if (reasons && typeof detail.reason === 'string' && Object.hasOwn(reasons, detail.reason)) {
    return reasons[detail.reason];
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
