import { Input, InputArea } from '@cloudflare/kumo/components/input';
import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';
import { FormField } from '../../components/form/FormField';
import { CodeText } from '../../components/text/CodeText';

const inputClass = 'h-10 pointer-coarse:h-9';

/**
 * A labelled text field of the site form. Machine values (addresses, paths,
 * PEM) are typed left to right; a refused value shows its sentence under it.
 */
export function TextField({
  id,
  label,
  hint,
  error,
  value,
  onChange,
  machine,
  rows,
  inputMode,
  placeholder,
}: {
  id: string;
  label: string;
  hint?: ReactNode;
  error?: string;
  value: string;
  onChange: (v: string) => void;
  /** Typed left to right, with no autocorrect. */
  machine?: boolean;
  /** A text area this many rows high. */
  rows?: number;
  inputMode?: 'url' | 'numeric' | 'text';
  placeholder?: string;
}) {
  const described = [hint ? `${id}-hint` : '', error ? `${id}-error` : ''].join(' ').trim();
  const common = {
    id,
    name: id,
    value,
    placeholder,
    'aria-labelledby': `${id}-label`,
    'aria-describedby': described || undefined,
    'aria-invalid': error ? true : undefined,
    ...(machine
      ? { dir: 'ltr', autoComplete: 'off', autoCapitalize: 'off', spellCheck: false }
      : { autoComplete: 'off' }),
  };
  return (
    <FormField id={id} label={label} hint={hint}>
      {rows ? (
        <InputArea
          {...common}
          rows={rows}
          className={cn('font-mono text-sm', error && 'ring-kumo-danger')}
          onChange={(e) => onChange(e.target.value)}
        />
      ) : (
        <Input
          {...common}
          inputMode={inputMode}
          className={cn(inputClass, error && 'ring-kumo-danger')}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
      {error ? (
        <p id={`${id}-error`} className="text-sm text-kumo-danger">
          <CodeText text={error} />
        </p>
      ) : null}
    </FormField>
  );
}
