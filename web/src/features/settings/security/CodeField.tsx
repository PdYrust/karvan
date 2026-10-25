import { Input } from '@cloudflare/kumo/components/input';
import { FormField } from '../../../components/form/FormField';
import { codeDigits } from './code';

/** The 6-digit code from an authenticator app; Persian digits work too. */
export function CodeField({
  id,
  label,
  hint,
  value,
  onChange,
  disabled,
}: {
  id: string;
  label: string;
  hint?: string;
  value: string;
  onChange: (code: string) => void;
  disabled?: boolean;
}) {
  return (
    <FormField id={id} label={label} hint={hint}>
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
        className="h-10 w-full max-w-48 text-center font-mono tracking-[0.4em] pointer-coarse:h-11"
      />
    </FormField>
  );
}
