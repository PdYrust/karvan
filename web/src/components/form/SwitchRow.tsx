import { Switch } from '@cloudflare/kumo/components/switch';

/** A labelled switch with one line under its label, as the forms use them. */
export function SwitchRow({
  id,
  label,
  hint,
  checked,
  onChange,
}: {
  id: string;
  label: string;
  hint: string;
  checked: boolean;
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
        onCheckedChange={onChange}
        // The switch is 18 px high; the hit area reaches 44 px on touch screens.
        className="relative shrink-0 pointer-coarse:before:absolute pointer-coarse:before:-inset-x-[9px] pointer-coarse:before:-inset-y-[13px] pointer-coarse:before:content-['']"
      />
    </div>
  );
}
