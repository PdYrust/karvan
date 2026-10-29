import type { Translate } from '../../i18n/core';
import { useT, type Messages } from '../../i18n/locale';
import { errorOf } from './errors';
import { useSubscriptionErrorText } from './subscriptions';

/**
 * Sentences for the Let's Encrypt codes (D58, D71): the answers' codes and
 * the codes of an order's error. Only the Domains and certificates page
 * meets them, so they load with it alone.
 */
const en = {
  acme_invalid:
    "Karvan can't start this Let's Encrypt order. Check the domain and the way you picked.",
  acme_invalid_agreeTos:
    "Accept the Let's Encrypt subscriber agreement to get the first certificate.",
  acme_invalid_email: 'Enter one email address, like ops@example.com, or leave it empty.',
  acme_invalid_ip: "Let's Encrypt gives no certificates for IP addresses here. Pick another way.",
  acme_invalid_wildcard_needs_dns: "A wildcard domain needs Let's Encrypt by DNS.",
  acme_busy: "A Let's Encrypt order for this domain is already running. Wait for it, or cancel it.",
  acme_state: "This order isn't waiting for its DNS record anymore. Reload the page.",
  acme_dns_not_found: "The TXT record isn't visible yet. DNS changes can take a few minutes.",
  acme_port_denied:
    "Karvan can't use port 80 because it doesn't run as root. Use Let's Encrypt by DNS.",
  acme_port_busy: "Port 80 is used by another program. Use Let's Encrypt by DNS.",
  acme_port_tls:
    "An HTTPS rule uses port 80. Use Let's Encrypt by DNS, or move that rule to another port.",
  acme_rate_limited: "Let's Encrypt is limiting requests for this domain. Try again after {time}.",
  acme_challenge_failed:
    "Let's Encrypt couldn't confirm this domain. Check the domain's DNS and try again.",
  acme_caa: "A CAA record of this domain doesn't allow Let's Encrypt. Change it and try again.",
  acme_rejected: "Let's Encrypt gives no certificate for this name.",
  acme_unreachable:
    "Karvan couldn't reach Let's Encrypt. Check this server's internet access and try again.",
  acme_expired: 'The order expired before it finished. Start it again.',
  acme_account: "Karvan's Let's Encrypt account had a problem. Try again.",
  acme_failed: "Let's Encrypt couldn't give the certificate. Try again later.",
};

export type AcmeErrorKey = keyof typeof en;

const fa: Messages<AcmeErrorKey>['fa'] = {
  acme_invalid:
    "کاروان نمیتواند این سفارش Let's Encrypt را شروع کند. دامنه و روشی را که انتخاب کرده اید بررسی کنید.",
  acme_invalid_agreeTos: "برای گرفتن اولین گواهی، توافق نامه مشترکان Let's Encrypt را بپذیرید.",
  acme_invalid_email: 'یک ایمیل مثل ops@example.com وارد کنید یا آن را خالی بگذارید.',
  acme_invalid_ip: "Let's Encrypt اینجا برای IP گواهی نمیدهد. روش دیگری انتخاب کنید.",
  acme_invalid_wildcard_needs_dns: "برای دامنه ستاره دار فقط Let's Encrypt با DNS ممکن است.",
  acme_busy: "یک سفارش Let's Encrypt برای این دامنه در حال انجام است. صبر کنید یا آن را لغو کنید.",
  acme_state: 'این سفارش دیگر منتظر رکورد DNS نیست. صفحه را دوباره باز کنید.',
  acme_dns_not_found: 'رکورد TXT هنوز دیده نمیشود. اعمال تغییرات DNS ممکن است چند دقیقه طول بکشد.',
  acme_port_denied:
    "کاروان نمیتواند از پورت 80 استفاده کند، چون با دسترسی root اجرا نمیشود. از Let's Encrypt با DNS استفاده کنید.",
  acme_port_busy: "پورت 80 در اختیار برنامه دیگری است. از Let's Encrypt با DNS استفاده کنید.",
  acme_port_tls:
    "یک قانون HTTPS از پورت 80 استفاده میکند. از Let's Encrypt با DNS استفاده کنید یا آن قانون را به پورت دیگری ببرید.",
  acme_rate_limited:
    "Let's Encrypt فعلا درخواست های این دامنه را محدود کرده است. بعد از {time} دوباره امتحان کنید.",
  acme_challenge_failed:
    "Let's Encrypt نتوانست این دامنه را تایید کند. DNS دامنه را بررسی کنید و دوباره امتحان کنید.",
  acme_caa:
    "رکورد CAA این دامنه اجازه Let's Encrypt را نمیدهد. آن را تغییر دهید و دوباره امتحان کنید.",
  acme_rejected: "Let's Encrypt برای این نام گواهی نمیدهد.",
  acme_unreachable:
    "کاروان به Let's Encrypt دسترسی پیدا نکرد. اینترنت این سرور را بررسی کنید و دوباره امتحان کنید.",
  acme_expired: 'سفارش پیش از تمام شدن منقضی شد. آن را دوباره شروع کنید.',
  acme_account: "حساب Let's Encrypt کاروان مشکل پیدا کرد. دوباره امتحان کنید.",
  acme_failed: "Let's Encrypt نتوانست گواهی را بدهد. کمی بعد دوباره امتحان کنید.",
};

export const acmeErrorMessages: Messages<AcmeErrorKey> = { en, fa };

/** The sentences for a refused acme_invalid, by reason, then by field. */
const reasonKeys: Readonly<Record<string, AcmeErrorKey>> = {
  ip: 'acme_invalid_ip',
  wildcard_needs_dns: 'acme_invalid_wildcard_needs_dns',
};
const fieldKeys: Readonly<Record<string, AcmeErrorKey>> = {
  agreeTos: 'acme_invalid_agreeTos',
  email: 'acme_invalid_email',
};

const isReason = (k: string) => k.startsWith('acme_invalid_');

/** Every code the server sends that this list words, for the words test. */
export const acmeCodes = (Object.keys(en) as AcmeErrorKey[]).filter((k) => !isReason(k));

/** The sentence a code and its extras get here, or undefined for another code. */
export function acmeErrorKey(
  code: string,
  detail: Readonly<Record<string, unknown>> = {},
): AcmeErrorKey | undefined {
  if (code === 'acme_invalid') {
    const { reason, field } = detail;
    if (typeof reason === 'string' && Object.hasOwn(reasonKeys, reason)) return reasonKeys[reason];
    if (typeof field === 'string' && Object.hasOwn(fieldKeys, field)) return fieldKeys[field];
  }
  return Object.hasOwn(en, code) && !isReason(code) ? (code as AcmeErrorKey) : undefined;
}

/** The sentence for a failed request: a Let's Encrypt code here, else the subscriptions one. */
export function acmeErrorText(
  t: Translate<AcmeErrorKey>,
  other: (err: unknown) => string,
  err: unknown,
): string {
  const { code, detail } = errorOf(err);
  const key = acmeErrorKey(code, detail);
  return key ? t(key) : other(err);
}

/** acmeErrorText in the page's language. */
export function useAcmeErrorText(): (err: unknown) => string {
  const t = useT(acmeErrorMessages);
  const other = useSubscriptionErrorText();
  return (err) => acmeErrorText(t, other, err);
}
