import { Input } from '@cloudflare/kumo/components/input';
import { useId, useRef } from 'react';
import { CopyButton } from '../../../components/text/CopyButton';

/** A value to copy: its label, the value read only in mono, and Copy next to it. */
export function CopyValue({
  label,
  value,
  shown,
}: {
  label: string;
  value: string;
  shown?: string;
}) {
  const id = useId();
  const field = useRef<HTMLInputElement>(null);
  return (
    <div className="flex flex-col gap-2">
      <label id={`${id}-label`} htmlFor={id} className="text-sm font-medium text-kumo-strong">
        {label}
      </label>
      <div className="flex gap-2">
        <Input
          ref={field}
          id={id}
          aria-labelledby={`${id}-label`}
          readOnly
          dir="ltr"
          spellCheck={false}
          value={shown ?? value}
          onFocus={(e) => e.currentTarget.select()}
          className="h-10 min-w-0 flex-1 font-mono pointer-coarse:h-9"
        />
        <CopyButton
          text={value}
          select={() => field.current?.select()}
          className="h-10 shrink-0 pointer-coarse:h-9"
        />
      </div>
    </div>
  );
}
