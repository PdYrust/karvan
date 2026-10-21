import { Badge } from '@cloudflare/kumo/components/badge';
import { cn } from '@cloudflare/kumo/utils';
import {
  CheckCircleIcon,
  MinusCircleIcon,
  WarningCircleIcon,
  type Icon,
} from '@phosphor-icons/react';
import type { ReactNode } from 'react';

export type StatusTone = 'success' | 'warning' | 'neutral';

const tones: Record<
  StatusTone,
  { icon: Icon; color: string; badge: 'success' | 'warning' | 'secondary' }
> = {
  success: { icon: CheckCircleIcon, color: 'text-kumo-success', badge: 'success' },
  warning: { icon: WarningCircleIcon, color: 'text-kumo-warning', badge: 'warning' },
  neutral: { icon: MinusCircleIcon, color: 'text-kumo-subtle', badge: 'secondary' },
};

/** A status as a badge: an icon, a word and Kumo's status color, never the color alone. */
export function StatusBadge({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  const look = tones[tone];
  const Glyph = look.icon;
  return (
    // Kumo's warning text on its own tint is under 4.5:1, so that word stays in the default color.
    <Badge
      variant={look.badge}
      className={tone === 'warning' ? 'text-kumo-default' : undefined}
      icon={<Glyph size={14} weight="bold" aria-hidden="true" className={look.color} />}
    >
      {children}
    </Badge>
  );
}

/** A status in a table cell or a line of text: the icon carries the color, the word stays default. */
export function StatusLabel({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  const look = tones[tone];
  const Glyph = look.icon;
  return (
    <span className="inline-flex items-center gap-1.5">
      <Glyph size={16} weight="bold" aria-hidden="true" className={cn('shrink-0', look.color)} />
      <span>{children}</span>
    </span>
  );
}
