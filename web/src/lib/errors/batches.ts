import type { Translate } from '../../i18n/core';
import type { Messages } from '../../i18n/locale';
import { errorOf } from './errors';

/**
 * Sentences for the codes only the Batches page meets (D58). They load
 * with that page; errors.ts keeps the codes every page can meet.
 */
const en = {
  batch_invalid: 'Use 1 to 64 characters on one line for the batch name.',
  batch_active: "An active batch can't be deleted. Only an inactive one can.",
  batch_in_use:
    'This batch is the only batch of a rule. Add another batch to the rule, or delete the rule first.',
  not_reactivatable: 'Only a batch whose backup restore failed can be made active again.',
};

export type BatchErrorKey = keyof typeof en;

const fa: Messages<BatchErrorKey>['fa'] = {
  batch_invalid: 'نام دسته باید ۱ تا ۶۴ نویسه و در یک خط باشد.',
  batch_active: 'دسته فعال حذف نمیشود. فقط دسته غیرفعال را میشود حذف کرد.',
  batch_in_use:
    'این دسته تنها دسته یک قانون است. اول دسته دیگری به آن قانون اضافه کنید یا قانون را حذف کنید.',
  not_reactivatable: 'فقط دسته ای که برگرداندن پشتیبانش ناموفق بوده دوباره فعال میشود.',
};

export const batchErrorMessages: Messages<BatchErrorKey> = { en, fa };

/** Every code of this list, for the words test. */
export const batchCodes = Object.keys(en) as readonly BatchErrorKey[];

const isBatchCode = (code: string): code is BatchErrorKey => Object.hasOwn(en, code);

/** The sentence for a failed batch request: this list's own codes, else the shared sentence. */
export function batchErrorText(
  t: Translate<BatchErrorKey>,
  shared: (err: unknown) => string,
  err: unknown,
): string {
  const { code } = errorOf(err);
  return isBatchCode(code) ? t(code) : shared(err);
}
