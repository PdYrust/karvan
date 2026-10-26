import { Input } from '@cloudflare/kumo/components/input';
import { cn } from '@cloudflare/kumo/utils';
import { FormField } from '../../../components/form/FormField';
import { codeDigits } from './code';

const cells = [0, 1, 2, 3, 4, 5] as const;

/**
 * The 6-digit code from an authenticator app, drawn as six boxes. One real
 * input lies over them, so typing, pasting, the phone's code suggestion and
 * screen readers all see a single field; Persian digits work too.
 */
export function CodeField({
  id,
  label,
  hint,
  hideLabel,
  value,
  onChange,
  disabled,
}: {
  id: string;
  label: string;
  hint?: string;
  hideLabel?: boolean;
  value: string;
  onChange: (code: string) => void;
  disabled?: boolean;
}) {
  const at = Math.min(value.length, cells.length - 1);
  return (
    <FormField id={id} label={label} hint={hint} hideLabel={hideLabel}>
      <div dir="ltr" className={cn('group relative flex w-fit gap-2', disabled && 'opacity-50')}>
        {cells.map((i) => (
          <span
            key={i}
            aria-hidden="true"
            className={cn(
              'grid h-10 w-9 place-items-center rounded-lg bg-kumo-control font-mono text-lg text-kumo-strong ring ring-kumo-line pointer-coarse:h-11 pointer-coarse:w-10',
              i === at && 'group-focus-within:ring-[1.5px] group-focus-within:ring-kumo-focus',
            )}
          >
            {value[i] ?? ''}
          </span>
        ))}
        <Input
          id={id}
          aria-labelledby={`${id}-label`}
          aria-describedby={hint ? `${id}-hint` : undefined}
          dir="ltr"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(codeDigits(e.target.value))}
          // 16 px text, so phones do not zoom into the field.
          className="absolute inset-0 h-full w-full cursor-text text-base opacity-0"
        />
      </div>
    </FormField>
  );
}
