import { Badge } from '@cloudflare/kumo/components/badge';
import { cn } from '@cloudflare/kumo/utils';
import {
  CheckCircleIcon,
  MinusCircleIcon,
  PlayCircleIcon,
  WarningCircleIcon,
  XCircleIcon,
  type Icon,
} from '@phosphor-icons/react';
import type { ReactNode } from 'react';

/** `active` is something under way, such as a running migration. */
export type StatusTone = 'success' | 'warning' | 'danger' | 'active' | 'neutral';

const tones: Record<
  StatusTone,
  { icon: Icon; color: string; badge: 'success' | 'warning' | 'error' | 'info' | 'secondary' }
> = {
  success: { icon: CheckCircleIcon, color: 'text-kumo-success', badge: 'success' },
  warning: { icon: WarningCircleIcon, color: 'text-kumo-warning', badge: 'warning' },
  danger: { icon: XCircleIcon, color: 'text-kumo-danger', badge: 'error' },
  // Kumo's info blue is under 3:1 on white; the brand blue reads.
  active: { icon: PlayCircleIcon, color: 'text-(--color-kumo-brand)', badge: 'info' },
  neutral: { icon: MinusCircleIcon, color: 'text-kumo-subtle', badge: 'secondary' },
};

/** A status as a badge: an icon, a word and Kumo's status color, never the color alone. */
export function StatusBadge({ tone, children }: { tone: StatusTone; children: ReactNode }) {
  const look = tones[tone];
  const Glyph = look.icon;
  return (
    // Kumo's warning, error and info text on their own tints is under 4.5:1, so those words stay in the default color.
    // Keyed by tone, so a badge that changes state fades into the new one.
    <Badge
      key={tone}
      variant={look.badge}
      className={cn('k-fade', tone !== 'success' && tone !== 'neutral' && 'text-kumo-default')}
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
    <span key={tone} className="k-fade inline-flex items-center gap-1.5">
      <Glyph size={16} weight="bold" aria-hidden="true" className={cn('shrink-0', look.color)} />
      <span>{children}</span>
    </span>
  );
}
