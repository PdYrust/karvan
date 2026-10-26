import { cn } from '@cloudflare/kumo/utils';
import type { ReactNode } from 'react';

interface FormFieldProps {
  id: string;
  label: string;
  hint?: ReactNode;
  /** The label is for screen readers only, where the words around the field already say it. */
  hideLabel?: boolean;
  className?: string;
  children: ReactNode;
}

/**
 * Label above a control, with optional helper text tied to it by id. Give the
 * control aria-labelledby={`${id}-label`} as well; Kumo inputs check for it.
 */
export function FormField({ id, label, hint, hideLabel, className, children }: FormFieldProps) {
  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <label
        id={`${id}-label`}
        htmlFor={id}
        className={cn('text-sm font-medium text-kumo-strong', hideLabel && 'sr-only')}
      >
        {label}
      </label>
      {children}
      {hint ? (
        <p id={`${id}-hint`} className="text-sm text-kumo-subtle">
          {hint}
        </p>
      ) : null}
    </div>
  );
}
