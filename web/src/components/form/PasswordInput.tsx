import { Input } from '@cloudflare/kumo/components/input';
import { EyeIcon, EyeSlashIcon } from '@phosphor-icons/react';
import { useState, type ComponentProps } from 'react';

type PasswordInputProps = Omit<ComponentProps<typeof Input>, 'type'> & {
  showLabel: string;
  hideLabel: string;
};

/**
 * Password field with a show/hide button. Kumo's SensitiveInput is not used
 * because it offers to copy the value, which a sign-in form should not do.
 */
export function PasswordInput({ showLabel, hideLabel, className, ...props }: PasswordInputProps) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <Input
        {...props}
        type={visible ? 'text' : 'password'}
        dir="ltr"
        spellCheck={false}
        autoCapitalize="off"
        className={`w-full pe-11 ${className ?? ''}`}
      />
      <button
        type="button"
        aria-label={visible ? hideLabel : showLabel}
        aria-pressed={visible}
        onClick={() => setVisible((v) => !v)}
        className="absolute inset-y-0 end-0 flex w-11 items-center justify-center rounded-e-lg text-kumo-subtle transition-colors hover:text-kumo-default focus-visible:text-kumo-default"
      >
        {visible ? <EyeSlashIcon size={18} /> : <EyeIcon size={18} />}
      </button>
    </div>
  );
}
