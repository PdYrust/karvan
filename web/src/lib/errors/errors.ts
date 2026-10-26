import type { Translate } from '../../i18n/core';
import { useT, type Messages } from '../../i18n/locale';
import { ApiError } from '../api/client';

/**
 * One sentence for every error code the API sends (D4), and for the codes a
 * migration's event stream logs when a compare fails. Each says what
 * happened and what to do; no server text reaches the screen.
 */
const en = {
  unknown: "Karvan couldn't do that. Try again in a bit.",
  offline: "Can't reach Karvan. Check your connection and try again.",
  internal: 'Something went wrong on the server. Try again in a bit.',
  bad_request: "Karvan couldn't use what was sent. Check the fields and try again.",
  unauthorized: 'Your session ended. Sign in again.',
  csrf: 'This page is out of date. Reload it.',
  not_found: "Karvan can't find this. It may have been removed.",
  busy: 'Karvan is still working on this. Wait for it to finish, then try again.',
  locked: 'Too many tries. Wait a few minutes and try again.',
  blocked: 'Some problems still need a fix before this can start.',
  not_checked: "Karvan hasn't read the old panel for this migration yet.",
  login_gone: 'Karvan no longer holds the old panel login. Start a new migration to sign in again.',
  stale_plan: 'This migration changed in another tab. Reload the page.',
  plan_invalid: "The plan has a value Karvan can't use. Check the last change you made.",
  panel_not_ready: "The destination panel isn't ready. Open it on the Panels page to see why.",
  panel_control_unavailable:
    "Rolling back from a backup needs Karvan to stop and start the panel, and it can't here. Choose undo in Safe Mode settings.",
  panel_changed: 'The panel changed after this backup was made.',
  template_name: 'A template with this name already exists.',
  name_taken: 'This name is already taken.',
  confirm_required: 'Type the name to confirm.',
  gone: 'This file was already downloaded. Karvan keeps it only until then.',
  cancelled: 'This was stopped before it finished.',
  not_rollbackable: 'Only a failed or interrupted migration can be rolled back, and only once.',
  not_deletable: 'This migration has run, so its record stays. Only a draft can be deleted.',
  destination_locked:
    'Another migration or rollback is writing to this panel. Wait for it to finish, then try again.',
  weak_password: "This admin password doesn't meet the panel's rules.",
  weak_password_short: 'The admin password needs at least 12 characters.',
  weak_password_long: 'The admin password is too long. Use a shorter one.',
  weak_password_weak:
    'The admin password needs at least 2 digits, 2 capital letters, 2 small letters and 1 symbol.',
  weak_password_has_name: "The admin password can't contain the username.",
  weak_password_has_quote: 'The admin password can\'t contain a double quote (").',
  invalid_credentials: "The username or password isn't right.",
  totp_required: 'Enter the code from your authenticator app.',
  totp_invalid: "That code isn't right. Enter the current one.",
  totp_not_started: 'Start the two-step setup again.',
  totp_already_on: 'Two-step sign-in is already on.',
  totp_not_on: 'Two-step sign-in is already off.',
  port_unavailable:
    "Karvan can't open this port. It may be in use, or need rights Karvan doesn't have. Pick another one.",
  no_address_change:
    'No new address is waiting. It may have expired after 10 minutes. Start the change again.',
  not_new_address: 'Confirm only works from the new address. Open it and confirm there.',
  site_invalid: "A field of this site has a value Karvan can't use. Check the fields.",
  site_overlap: 'Another site already serves this domain and port.',
  plain_http_off: 'Plain HTTP is off for old links. Turn it on or use HTTPS.',
  port_mixed:
    'This port already has sites the other way. All sites on one port use HTTPS, or none do.',
  too_large: 'The file is larger than 16 MB. Split it and upload the parts one by one.',
  source_unknown: "Karvan couldn't read the old panel. Try again in a bit.",
  source_unreachable:
    "Karvan can't reach the old panel. Check the address and that the panel is running.",
  source_timeout: 'The old panel took too long to answer. Try again in a bit.',
  source_forbidden_address:
    "Karvan doesn't connect to this address. Use the panel's own domain or IP address.",
  source_tls_failed:
    "Karvan couldn't open a secure connection to the old panel. Check the address and its port.",
  source_tls_untrusted: "Karvan doesn't trust the old panel's certificate.",
  source_redirect:
    'The old panel address sends Karvan somewhere else. Enter the address it moves to.',
  source_unauthorized:
    "The old panel didn't accept this sign-in. Check the username and password, or the API key.",
  source_forbidden:
    'The old panel refused this admin. Sign in with an admin that can see the users.',
  source_not_found: 'Karvan found no panel at this address. Check the address, including its path.',
  source_rate_limited: 'The old panel is limiting requests. Wait a minute and try again.',
  source_server_error: 'The old panel had an error. Check that it works, then try again.',
  source_bad_response:
    "The answer from this address doesn't look like Marzban or PasarGuard. Check the address and the kind.",
  source_too_large: 'The old panel sent more data in one answer than Karvan accepts.',
};

export type ErrorKey = keyof typeof en;

const fa: Messages<ErrorKey>['fa'] = {
  unknown: 'کاروان نتوانست این کار را انجام دهد. چند لحظه دیگر دوباره امتحان کنید.',
  offline: 'اتصال به کاروان برقرار نشد. اتصال را بررسی کنید و دوباره امتحان کنید.',
  internal: 'مشکلی در سرور پیش آمد. چند لحظه دیگر دوباره امتحان کنید.',
  bad_request:
    'کاروان نتوانست از مقدارهای ارسال شده استفاده کند. فیلدها را بررسی کنید و دوباره امتحان کنید.',
  unauthorized: 'نشست شما تمام شده است. دوباره وارد شوید.',
  csrf: 'این صفحه قدیمی است. صفحه را دوباره بارگذاری کنید.',
  not_found: 'کاروان این مورد را پیدا نمیکند. شاید حذف شده باشد.',
  busy: 'کاروان هنوز روی این کار است. صبر کنید تمام شود و بعد دوباره امتحان کنید.',
  locked: 'تلاش ها زیاد بود. چند دقیقه صبر کنید و دوباره امتحان کنید.',
  blocked: 'هنوز چند مشکل مانده که پیش از شروع باید رفع شوند.',
  not_checked: 'کاروان هنوز پنل قدیمی این انتقال را نخوانده است.',
  login_gone: 'کاروان دیگر ورود پنل قدیمی را ندارد. برای ورود دوباره، انتقال جدیدی شروع کنید.',
  stale_plan: 'این انتقال در زبانه دیگری تغییر کرده است. صفحه را دوباره بارگذاری کنید.',
  plan_invalid:
    'در برنامه مقداری هست که کاروان نمیتواند از آن استفاده کند. آخرین تغییر را بررسی کنید.',
  panel_not_ready: 'پنل مقصد آماده نیست. برای دیدن دلیل، آن را در صفحه پنل ها باز کنید.',
  panel_control_unavailable:
    'برگرداندن از نسخه پشتیبان نیاز دارد کاروان پنل را متوقف و روشن کند و اینجا نمیتواند. در تنظیمات حالت امن، بازگردانی تغییرات را انتخاب کنید.',
  panel_changed: 'پنل پس از گرفتن این نسخه پشتیبان تغییر کرده است.',
  template_name: 'الگویی با این نام از قبل وجود دارد.',
  name_taken: 'این نام قبلا گرفته شده است.',
  confirm_required: 'برای تایید، نام را بنویسید.',
  gone: 'این فایل قبلا دانلود شده است. کاروان آن را فقط تا همان موقع نگه میدارد.',
  cancelled: 'این کار پیش از تمام شدن متوقف شد.',
  not_rollbackable: 'فقط انتقال ناموفق یا نیمه تمام را میشود برگرداند، آن هم یک بار.',
  not_deletable: 'این انتقال اجرا شده است و سابقه اش میماند. فقط پیش نویس حذف میشود.',
  destination_locked:
    'انتقال یا برگرداندن دیگری روی این پنل مینویسد. صبر کنید تمام شود و بعد دوباره امتحان کنید.',
  weak_password: 'این رمز ادمین با قوانین پنل جور نیست.',
  weak_password_short: 'رمز ادمین باید دست کم ۱۲ نویسه باشد.',
  weak_password_long: 'رمز ادمین خیلی بلند است. رمز کوتاهتری بگذارید.',
  weak_password_weak: 'رمز ادمین باید دست کم ۲ رقم، ۲ حرف بزرگ، ۲ حرف کوچک و ۱ نماد داشته باشد.',
  weak_password_has_name: 'رمز ادمین نباید نام کاربری را داشته باشد.',
  weak_password_has_quote: 'رمز ادمین نباید علامت نقل قول (") داشته باشد.',
  invalid_credentials: 'نام کاربری یا رمز درست نیست.',
  totp_required: 'کد برنامه احراز هویت را وارد کنید.',
  totp_invalid: 'این کد درست نیست. کد فعلی را وارد کنید.',
  totp_not_started: 'راه اندازی ورود دومرحله ای را دوباره شروع کنید.',
  totp_already_on: 'ورود دومرحله ای از قبل روشن است.',
  totp_not_on: 'ورود دومرحله ای از قبل خاموش است.',
  port_unavailable:
    'کاروان نمیتواند این پورت را باز کند. شاید در حال استفاده باشد یا دسترسی لازم را نداشته باشد. پورت دیگری انتخاب کنید.',
  no_address_change:
    'آدرس جدیدی در انتظار نیست. شاید بعد از ۱۰ دقیقه منقضی شده باشد. تغییر را دوباره شروع کنید.',
  not_new_address: 'تایید فقط از آدرس جدید انجام میشود. آن را باز کنید و همانجا تایید کنید.',
  site_invalid:
    'یکی از فیلدهای این سایت مقداری دارد که کاروان نمیتواند استفاده کند. فیلدها را بررسی کنید.',
  site_overlap: 'سایت دیگری همین دامنه و پورت را دارد.',
  plain_http_off:
    'HTTP ساده برای لینک های قدیمی خاموش است. آن را روشن کنید یا از HTTPS استفاده کنید.',
  port_mixed:
    'این پورت سایت هایی به روش دیگر دارد. یا همه سایت های یک پورت HTTPS هستند یا هیچ کدام.',
  too_large:
    'فایل بزرگتر از ۱۶ مگابایت است. آن را چند قسمت کنید و قسمت ها را یکی یکی بارگذاری کنید.',
  source_unknown: 'کاروان نتوانست پنل قدیمی را بخواند. چند لحظه دیگر دوباره امتحان کنید.',
  source_unreachable:
    'کاروان به پنل قدیمی دسترسی ندارد. آدرس را بررسی کنید و مطمئن شوید پنل روشن است.',
  source_timeout: 'پنل قدیمی دیر جواب داد. چند لحظه دیگر دوباره امتحان کنید.',
  source_forbidden_address: 'کاروان به این آدرس وصل نمیشود. دامنه یا IP خود پنل را وارد کنید.',
  source_tls_failed:
    'کاروان نتوانست اتصال امن با پنل قدیمی برقرار کند. آدرس و پورت آن را بررسی کنید.',
  source_tls_untrusted: 'کاروان به گواهی پنل قدیمی اعتماد ندارد.',
  source_redirect: 'آدرس پنل قدیمی کاروان را به جای دیگری میفرستد. آدرس مقصد آن را وارد کنید.',
  source_unauthorized:
    'پنل قدیمی این ورود را نپذیرفت. نام کاربری و رمز، یا کلید API را بررسی کنید.',
  source_forbidden: 'پنل قدیمی به این ادمین اجازه نداد. با ادمینی وارد شوید که کاربرها را میبیند.',
  source_not_found: 'کاروان در این آدرس پنلی پیدا نکرد. آدرس را همراه مسیر آن بررسی کنید.',
  source_rate_limited:
    'پنل قدیمی تعداد درخواست ها را محدود کرده است. یک دقیقه صبر کنید و دوباره امتحان کنید.',
  source_server_error: 'پنل قدیمی خطا داد. بررسی کنید درست کار کند و بعد دوباره امتحان کنید.',
  source_bad_response: 'جواب این آدرس شبیه مرزبان یا پاسارگارد نیست. آدرس و نوع پنل را بررسی کنید.',
  source_too_large: 'پنل قدیمی در یک جواب بیشتر از حدی که کاروان میپذیرد داده فرستاد.',
};

export const errorMessages: Messages<ErrorKey> = { en, fa };

/** Every code the API or a compare's log can carry, for the tests. */
export const knownCodes: readonly string[] = Object.keys(en).filter(
  (k) => k !== 'unknown' && k !== 'source_unknown' && !k.startsWith('weak_password_'),
);

const passwordRules: Readonly<Record<string, ErrorKey | undefined>> = {
  password_short: 'weak_password_short',
  password_long: 'weak_password_long',
  password_weak: 'weak_password_weak',
  password_has_name: 'weak_password_has_name',
  password_has_quote: 'weak_password_has_quote',
};

const isKey = (code: string): code is ErrorKey => Object.hasOwn(en, code);

/** The sentence a code and its extras get. */
export function errorKey(code: string, detail: Readonly<Record<string, unknown>> = {}): ErrorKey {
  if (code === 'weak_password' && typeof detail.rule === 'string') {
    return (
      (Object.hasOwn(passwordRules, detail.rule) && passwordRules[detail.rule]) || 'weak_password'
    );
  }
  if (isKey(code) && !code.startsWith('weak_password_')) return code;
  return code.startsWith('source_') ? 'source_unknown' : 'unknown';
}

/** A failure as a code and its extras; anything that is not an API error is a server failure. */
export interface ErrorInfo {
  code: string;
  detail: Readonly<Record<string, unknown>>;
}

export function errorOf(err: unknown): ErrorInfo {
  if (err instanceof ApiError) return { code: err.code, detail: err.detail };
  return { code: 'internal', detail: {} };
}

/** The sentence for a failed request, or for a code logged on a migration's stream. */
export function errorText(t: Translate<ErrorKey>, err: unknown): string {
  const info =
    typeof err === 'object' && err !== null && !(err instanceof Error) && 'code' in err
      ? (err as ErrorInfo)
      : errorOf(err);
  return t(errorKey(info.code, info.detail));
}

/** errorText in the page's language. */
export function useErrorText(): (err: unknown) => string {
  const t = useT(errorMessages);
  return (err) => errorText(t, err);
}
