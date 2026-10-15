import { cn } from '@cloudflare/kumo/utils';
import { InfoIcon, WarningCircleIcon, WarningIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';

type Tone = 'danger' | 'warning' | 'info';

const tones: Record<Tone, { box: string; icon: ReactNode }> = {
  danger: {
    box: 'bg-kumo-danger-tint text-kumo-danger ring-kumo-danger/25',
    icon: <WarningCircleIcon size={18} weight="bold" />,
  },
  warning: {
    box: 'bg-kumo-warning-tint text-kumo-warning ring-kumo-warning/30',
    icon: <WarningIcon size={18} weight="bold" />,
  },
  info: {
    box: 'bg-kumo-info-tint text-kumo-info ring-kumo-info/25',
    icon: <InfoIcon size={18} weight="bold" />,
  },
};

/** A short message inside a form or panel. Announced to screen readers. */
export function InlineAlert({
  tone = 'danger',
  children,
  className,
}: {
  tone?: Tone;
  children: ReactNode;
  className?: string;
}) {
  const t = tones[tone];
  return (
    <div
      role="alert"
      className={cn(
        'flex items-start gap-2.5 rounded-lg px-3 py-2.5 text-sm ring',
        t.box,
        className,
      )}
    >
      <span className="mt-0.5 shrink-0">{t.icon}</span>
      <span className="text-kumo-default">{children}</span>
    </div>
  );
}
