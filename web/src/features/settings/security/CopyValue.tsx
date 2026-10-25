import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { CheckIcon, CopyIcon } from '@phosphor-icons/react';
import { useEffect, useId, useRef, useState } from 'react';
import { useT } from '../../../i18n/locale';
import { securityMessages } from './messages';

/** Copies text, or selects it in its field where the browser has no clipboard (plain HTTP). */
async function copy(text: string, field: HTMLInputElement | null): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    field?.select();
    return false;
  }
}

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
  const t = useT(securityMessages);
  const id = useId();
  const field = useRef<HTMLInputElement>(null);
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);
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
          className="h-10 min-w-0 flex-1 font-mono pointer-coarse:h-11"
        />
        <Button
          className="h-10 shrink-0 pointer-coarse:h-11"
          icon={copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
          onClick={() => void copy(value, field.current).then(setCopied)}
        >
          {t(copied ? 'copied' : 'copy')}
        </Button>
      </div>
    </div>
  );
}
