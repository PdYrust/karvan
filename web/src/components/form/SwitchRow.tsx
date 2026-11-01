import { Switch } from './Switch';
import type { ReactNode } from 'react';

/** A labelled switch with one line under its label, as the forms use them. */
export function SwitchRow({
  id,
  label,
  hint,
  checked,
  disabled,
  onChange,
}: {
  id: string;
  label: string;
  hint: ReactNode;
  checked: boolean;
  /** Locked, as when what the switch needs is missing. */
  disabled?: boolean;
  onChange: (on: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-6">
      <div className="flex min-w-0 flex-col gap-1">
        <span id={`${id}-label`} className="font-medium text-kumo-strong">
          {label}
        </span>
        <span id={`${id}-hint`} className="text-sm text-kumo-subtle">
          {hint}
        </span>
      </div>
      <Switch
        aria-labelledby={`${id}-label`}
        aria-describedby={`${id}-hint`}
        checked={checked}
        disabled={disabled}
        onCheckedChange={onChange}
      />
    </div>
  );
}
