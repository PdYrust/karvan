import { Input } from '@cloudflare/kumo/components/input';
import { Select } from '@cloudflare/kumo/components/select';
import { useState } from 'react';
import { intlLocale } from '../../i18n/core';
import { useT, type Messages } from '../../i18n/locale';
import type { Calendar, Day } from '../../lib/format/date';
import {
  fromJalali,
  gregorianMonthLength,
  jalaliMonthLength,
  toJalali,
} from '../../lib/format/jalali';
import { latinDigits } from '../../lib/format/number';
import { useFormat } from '../../lib/preferences/region';

const messages: Messages<'day' | 'month' | 'year'> = {
  en: { day: 'Day', month: 'Month', year: 'Year' },
  fa: { day: 'روز', month: 'ماه', year: 'سال' },
};

interface Parts {
  year: string;
  month: number;
  day: number;
}

const monthLength = (calendar: Calendar, year: number, month: number) =>
  calendar === 'persian' ? jalaliMonthLength(year, month) : gregorianMonthLength(year, month);

/** The Gregorian day the parts name, or null until they name one. */
function dayOf(calendar: Calendar, p: Parts): Day | null {
  const year = Number(latinDigits(p.year));
  if (!/^\d{4}$/.test(latinDigits(p.year)) || !p.month || !p.day) return null;
  const day = Math.min(p.day, monthLength(calendar, year, p.month));
  return calendar === 'persian'
    ? fromJalali({ year, month: p.month, day })
    : { year, month: p.month, day };
}

function partsOf(calendar: Calendar, value: Day | null): Parts {
  if (!value) return { year: '', month: 0, day: 0 };
  const d = calendar === 'persian' ? toJalali(value) : value;
  return { year: String(d.year), month: d.month, day: d.day };
}

/**
 * A date as day, month and year in the operator's calendar: Jalali in
 * Persian by default (Kumo's date picker has no Persian calendar),
 * Gregorian otherwise. It reports the Gregorian day, or null while a part
 * is missing.
 */
export function JalaliDateField({
  id,
  label,
  value,
  onChange,
}: {
  id: string;
  label: string;
  value: Day | null;
  onChange: (day: Day | null) => void;
}) {
  const t = useT(messages);
  const fmt = useFormat();
  const calendar = fmt.calendar;
  const [parts, setParts] = useState(() => partsOf(calendar, value));
  const update = (patch: Partial<Parts>) => {
    const next = { ...parts, ...patch };
    setParts(next);
    onChange(dayOf(calendar, next));
  };
  const months = new Intl.DateTimeFormat(`${intlLocale(fmt.locale)}-u-ca-${calendar}`, {
    month: 'long',
    timeZone: 'UTC',
  });
  // A day in the middle of each month of a common year names it.
  const monthName = (m: number) => {
    const g =
      calendar === 'persian'
        ? fromJalali({ year: 1404, month: m, day: 15 })
        : { year: 2026, month: m, day: 15 };
    return months.format(new Date(Date.UTC(g.year, g.month - 1, g.day)));
  };
  const range = (n: number) => Array.from({ length: n }, (_, i) => i + 1);
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 text-sm font-medium text-kumo-strong">{label}</legend>
      <div className="flex flex-wrap items-end gap-2">
        <div className="flex flex-col gap-1">
          <span id={`${id}-day`} className="text-sm text-kumo-subtle">
            {t('day')}
          </span>
          <Select
            aria-labelledby={`${id}-day`}
            placeholder={t('day')}
            className="w-20 pointer-coarse:h-9"
            value={parts.day ? String(parts.day) : null}
            items={Object.fromEntries(range(31).map((d) => [String(d), fmt.number(d)]))}
            onValueChange={(d) => update({ day: Number(d) })}
          />
        </div>
        <div className="flex flex-col gap-1">
          <span id={`${id}-month`} className="text-sm text-kumo-subtle">
            {t('month')}
          </span>
          <Select
            aria-labelledby={`${id}-month`}
            placeholder={t('month')}
            className="w-36 pointer-coarse:h-9"
            value={parts.month ? String(parts.month) : null}
            items={Object.fromEntries(range(12).map((m) => [String(m), monthName(m)]))}
            onValueChange={(m) => update({ month: Number(m) })}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label
            id={`${id}-year-label`}
            htmlFor={`${id}-year`}
            className="text-sm text-kumo-subtle"
          >
            {t('year')}
          </label>
          <Input
            id={`${id}-year`}
            aria-labelledby={`${id}-year-label`}
            inputMode="numeric"
            autoComplete="off"
            dir="ltr"
            maxLength={4}
            value={parts.year}
            onChange={(e) => update({ year: e.target.value })}
            className="w-20 tabular pointer-coarse:h-9"
          />
        </div>
      </div>
    </fieldset>
  );
}
