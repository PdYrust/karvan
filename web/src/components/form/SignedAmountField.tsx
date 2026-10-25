import { Input } from '@cloudflare/kumo/components/input';
import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';
import { useT, type Messages } from '../../i18n/locale';

const messages: Messages<'add' | 'takeOff'> = {
  en: { add: 'Adds. Switch to take off.', takeOff: 'Takes off. Switch to add.' },
  fa: { add: 'اضافه میکند. برای کم کردن بزنید.', takeOff: 'کم میکند. برای اضافه کردن بزنید.' },
};

/** "-5" for "5" and "5" for "-5" or "+5": the sign button's work. */
function flip(value: string): string {
  const t = value.trim();
  if (t.startsWith('-')) return t.slice(1);
  return `-${t.startsWith('+') ? t.slice(1) : t}`;
}

/**
 * A number that adds or takes off, such as +10 GB or -5 days. The operator
 * types the sign or uses the sign button, since a phone's number keyboard
 * often has no minus key. The parent reads the value with signedAmount.
 */
export function SignedAmountField({
  id,
  label,
  unit,
  value,
  onChange,
  hint,
  className,
}: {
  id: string;
  label: string;
  /** Shown after the number: "GB", "days". */
  unit: string;
  value: string;
  onChange: (value: string) => void;
  /** What the value does in words, under the field. */
  hint?: ReactNode;
  className?: string;
}) {
  const t = useT(messages);
  const negative = value.trim().startsWith('-');
  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <label id={`${id}-label`} htmlFor={id} className="text-sm font-medium text-kumo-strong">
        {label}
      </label>
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-label={negative ? t('takeOff') : t('add')}
          title={negative ? t('takeOff') : t('add')}
          onClick={() => onChange(flip(value))}
          className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-kumo-base text-lg font-semibold text-kumo-strong ring ring-kumo-line hover:bg-kumo-tint pointer-coarse:size-11"
        >
          <span aria-hidden="true">{negative ? '-' : '+'}</span>
        </button>
        <Input
          id={id}
          aria-labelledby={`${id}-label`}
          aria-describedby={hint ? `${id}-hint` : undefined}
          dir="ltr"
          inputMode="decimal"
          autoComplete="off"
          spellCheck={false}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-28 tabular pointer-coarse:h-11"
        />
        <span className="text-kumo-subtle">{unit}</span>
      </div>
      {hint ? (
        <p id={`${id}-hint`} aria-live="polite" className="text-sm text-kumo-subtle">
          {hint}
        </p>
      ) : null}
    </div>
  );
}
