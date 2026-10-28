import type { Translate } from '../../i18n/core';
import { useT, type Messages } from '../../i18n/locale';
import { errorOf, useErrorText } from './errors';

/**
 * Sentences for the codes only the Rules and Domains and certificates
 * pages meet (D58), the Let's Encrypt ones included. They load with those pages; errors.ts
 * keeps the codes every page can meet.
 */
const en = {
  site_invalid: "A field of this rule has a value Karvan can't use. Check the fields.",
  site_overlap: 'Another rule already answers this domain and path on this port.',
  port_taken: "Karvan's own panel uses this port. Pick another port.",
  plain_http_off: 'Plain HTTP is off for old links. Turn it on or use HTTPS.',
  port_mixed:
    'This port already has rules the other way. All rules on one port use HTTPS, or none do.',
  domain_invalid:
    'Enter a domain like sub.example.com, a wildcard like *.example.com, or an IP address.',
  domain_invalid_certificate: 'Pick how this domain gets its certificate.',
  domain_invalid_plainOk: 'Confirm plain HTTP for this domain, or give it a certificate.',
  domain_in_use: 'A rule still uses this domain. Remove it from the rule first.',
  certificate_invalid:
    "A field of this certificate has a value Karvan can't use. Check the fields.",
  certificate_invalid_unreadable:
    "Karvan can't read this file. Check the path and that the file is there.",
  certificate_invalid_no_cert:
    'Karvan found no certificate here. Use a PEM certificate file, or paste the certificate.',
  certificate_invalid_key_mismatch:
    "This key doesn't belong to the certificate. Use the key that was made with it.",
  certificate_invalid_not_covered:
    "This certificate doesn't cover the domain. Use one made for this name.",
  certificate_in_use:
    'A domain or rule still uses this certificate. Give it another certificate first.',
  certificate_managed:
    "Karvan renews this Let's Encrypt certificate itself, so it can't be changed by hand.",
  certificate_not_adopted:
    "Give this rule's domains their certificates in Domains and certificates, then save it again.",
  acme_invalid:
    "Karvan can't start this Let's Encrypt order. Check the domain and the way you picked.",
  site_invalid_unknown: 'A batch of this rule is gone. Pick the batches again.',
  site_invalid_inactive: 'A batch of this rule is not active. Pick only active batches.',
  site_invalid_mixed:
    'These batches come from different old panels or destinations. Pick batches of one old panel and one destination.',
  site_invalid_not_registered:
    'A domain of this rule is not in Domains and certificates. Add it there first.',
  site_invalid_pending:
    "A domain of this rule is still waiting for its Let's Encrypt certificate. Save the rule as plain HTTP, or wait.",
  site_invalid_no_certificate:
    "HTTPS needs a certificate. Give this rule's domains certificates, or turn HTTPS off.",
  redirect_invalid: "A field of this redirect has a value Karvan can't use. Check the fields.",
  redirect_invalid_loop:
    'This address leads back to a redirect rule of Karvan. Enter an address outside Karvan.',
};

export type SubscriptionErrorKey = keyof typeof en;

const fa: Messages<SubscriptionErrorKey>['fa'] = {
  site_invalid:
    'یکی از فیلدهای این قانون مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  site_overlap: 'قانون دیگری روی این پورت به همین دامنه و مسیر جواب میدهد.',
  port_taken: 'پنل خود کاروان از این پورت استفاده میکند. پورت دیگری انتخاب کنید.',
  plain_http_off:
    'HTTP ساده برای لینک های قدیمی خاموش است. آن را روشن کنید یا از HTTPS استفاده کنید.',
  port_mixed:
    'این پورت قانون هایی به روش دیگر دارد. یا همه قانون های یک پورت HTTPS هستند یا هیچ کدام.',
  domain_invalid:
    'دامنه ای مثل sub.example.com، دامنه ستاره دار مثل *.example.com یا یک IP وارد کنید.',
  domain_invalid_certificate: 'انتخاب کنید این دامنه گواهی اش را از کجا بگیرد.',
  domain_invalid_plainOk: 'HTTP ساده را برای این دامنه تایید کنید یا به آن گواهی بدهید.',
  domain_in_use: 'یک قانون هنوز از این دامنه استفاده میکند. اول آن را از قانون بردارید.',
  certificate_invalid:
    'یکی از فیلدهای این گواهی مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  certificate_invalid_unreadable:
    'کاروان نمیتواند این فایل را بخواند. مسیر را بررسی کنید و مطمئن شوید فایل آنجاست.',
  certificate_invalid_no_cert:
    'کاروان اینجا گواهی پیدا نکرد. فایل گواهی PEM بدهید یا متن گواهی را بچسبانید.',
  certificate_invalid_key_mismatch:
    'این کلید برای این گواهی نیست. کلیدی را بدهید که همراه همین گواهی ساخته شده است.',
  certificate_invalid_not_covered:
    'این گواهی این دامنه را پوشش نمیدهد. گواهی ای بدهید که برای همین نام ساخته شده است.',
  certificate_in_use:
    'یک دامنه یا قانون هنوز از این گواهی استفاده میکند. اول گواهی دیگری به آن بدهید.',
  certificate_managed:
    "کاروان این گواهی Let's Encrypt را خودش تمدید میکند و نمیشود آن را دستی تغییر داد.",
  certificate_not_adopted:
    'در دامنه ها و گواهی ها به دامنه های این قانون گواهی بدهید و بعد قانون را دوباره ذخیره کنید.',
  acme_invalid:
    "کاروان نمیتواند این سفارش Let's Encrypt را شروع کند. دامنه و روشی را که انتخاب کرده اید بررسی کنید.",
  site_invalid_unknown: 'یکی از دسته های این قانون دیگر نیست. دسته ها را دوباره انتخاب کنید.',
  site_invalid_inactive: 'یکی از دسته های این قانون فعال نیست. فقط دسته های فعال را انتخاب کنید.',
  site_invalid_mixed:
    'این دسته ها از پنل های قدیمی یا مقصدهای مختلف هستند. دسته های یک پنل قدیمی و یک مقصد را انتخاب کنید.',
  site_invalid_not_registered:
    'یکی از دامنه های این قانون در دامنه ها و گواهی ها نیست. اول آن را آنجا اضافه کنید.',
  site_invalid_pending:
    "یکی از دامنه های این قانون هنوز منتظر گواهی Let's Encrypt است. قانون را با HTTP ساده ذخیره کنید یا صبر کنید.",
  site_invalid_no_certificate:
    'HTTPS گواهی لازم دارد. به دامنه های این قانون گواهی بدهید یا HTTPS را خاموش کنید.',
  redirect_invalid:
    'یکی از فیلدهای این ریدایرکت مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  redirect_invalid_loop:
    'این آدرس دوباره به یک قانون ریدایرکت کاروان میرسد. آدرسی بیرون از کاروان وارد کنید.',
};

export const subscriptionErrorMessages: Messages<SubscriptionErrorKey> = { en, fa };

/** Why a rule's domains or HTTPS were refused, for panel and redirect rules alike (D48, D63). */
const domainReasons = {
  not_registered: 'site_invalid_not_registered',
  pending: 'site_invalid_pending',
  no_certificate: 'site_invalid_no_certificate',
} as const;

/** The sentences for why a field was refused, by code and reason. */
const reasonKeys: Readonly<Record<string, Readonly<Record<string, SubscriptionErrorKey>>>> = {
  certificate_invalid: {
    unreadable: 'certificate_invalid_unreadable',
    no_cert: 'certificate_invalid_no_cert',
    key_mismatch: 'certificate_invalid_key_mismatch',
    not_covered: 'certificate_invalid_not_covered',
  },
  site_invalid: {
    unknown: 'site_invalid_unknown',
    inactive: 'site_invalid_inactive',
    mixed: 'site_invalid_mixed',
    ...domainReasons,
  },
  redirect_invalid: { loop: 'redirect_invalid_loop', ...domainReasons },
};

/** The sentences for a refused field that has no reason, by code and field. */
const fieldKeys: Readonly<Record<string, Readonly<Record<string, SubscriptionErrorKey>>>> = {
  domain_invalid: { certificate: 'domain_invalid_certificate', plainOk: 'domain_invalid_plainOk' },
};

const isReason = (k: string) =>
  k.startsWith('certificate_invalid_') ||
  k.startsWith('site_invalid_') ||
  k.startsWith('redirect_invalid_') ||
  k.startsWith('domain_invalid_');

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
  const fields = Object.hasOwn(fieldKeys, code) ? fieldKeys[code] : undefined;
  if (fields && typeof detail.field === 'string' && Object.hasOwn(fields, detail.field)) {
    return fields[detail.field];
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
