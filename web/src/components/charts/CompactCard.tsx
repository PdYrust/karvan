import { cn } from '@cloudflare/kumo/utils';
import { ArrowDownRightIcon, ArrowUpRightIcon } from '@phosphor-icons/react';
import { useId, type KeyboardEvent, type ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import type { Change } from '../../lib/charts/change';
import { useFormat } from '../../lib/preferences/region';
import { Bone } from '../feedback/PageSkeleton';
import { chartMessages } from './messages';

/** The change since the start of the window: a trend arrow and the figure, colored by whether it is good. */
function ChangeFigure({ change, within }: { change: Change; within: string }) {
  const t = useT(chartMessages);
  const fmt = useFormat();
  const by = fmt.percent(change.by, change.by < 0.1 ? 1 : 0);
  const Arrow = change.up ? ArrowUpRightIcon : ArrowDownRightIcon;
  return (
    <span
      className={cn(
        'flex items-center gap-1 text-sm font-medium tabular',
        change.good ? 'text-kumo-success' : 'text-kumo-danger',
      )}
    >
      <Arrow size={14} weight="bold" aria-hidden="true" className="shrink-0 rtl:-scale-x-100" />
      <span aria-hidden="true">{by}</span>
      <span className="sr-only">{t(change.up ? 'changeUp' : 'changeDown', { by, within })}</span>
    </span>
  );
}

/**
 * A compact card, like the dashboard's: the label, the value large, the
 * change, then its chart along the bottom edge (or a list, or facts). With
 * `onOpen` the whole card is one button that opens its detail; without it
 * the card shows everything and takes no press.
 */
export function CompactCard({
  title,
  value,
  unit,
  change,
  within,
  sub,
  chart,
  children,
  onOpen,
}: {
  title: string;
  value?: string;
  unit?: string;
  change?: Change;
  /** The time the change covers, in words ("the last hour"), for screen readers. */
  within: string;
  sub?: ReactNode;
  /** Full bleed along the bottom edge (a CompactChart). */
  chart?: ReactNode;
  /** Padded content under the header: a bar list, facts or a meter. */
  children?: ReactNode;
  onOpen?: () => void;
}) {
  const id = useId();
  const press = onOpen
    ? {
        role: 'button',
        tabIndex: 0,
        'aria-haspopup': 'dialog' as const,
        'aria-labelledby': `${id}-t ${id}-v`,
        onClick: onOpen,
        onKeyDown: (e: KeyboardEvent) => {
          if (e.key !== 'Enter' && e.key !== ' ') return;
          e.preventDefault();
          onOpen();
        },
      }
    : { role: 'group', 'aria-labelledby': `${id}-t` };
  return (
    <div
      {...press}
      className={cn(
        'flex min-w-0 flex-col overflow-hidden rounded-lg bg-kumo-base ring ring-kumo-hairline',
        onOpen &&
          'cursor-pointer outline-none hover:ring-kumo-line focus-visible:ring-2 focus-visible:ring-kumo-focus',
      )}
    >
      <div className="flex min-w-0 flex-col gap-0.5 px-4 pt-3">
        <span id={`${id}-t`} className="truncate text-sm text-kumo-subtle">
          {title}
        </span>
        {/* The change sits on the value line, right after it, as in the dashboard. */}
        {value ? (
          <span className="flex min-w-0 flex-wrap items-baseline gap-x-2">
            <span id={`${id}-v`} className="flex min-w-0 items-baseline gap-1">
              <span className="truncate text-title font-semibold text-kumo-strong tabular">
                {value}
              </span>
              {unit ? <span className="shrink-0 text-sm text-kumo-subtle">{unit}</span> : null}
            </span>
            {change ? <ChangeFigure change={change} within={within} /> : null}
          </span>
        ) : null}
        {sub ? <span className="line-clamp-2 text-sm text-kumo-subtle">{sub}</span> : null}
      </div>
      {children ? <div className="px-4 pt-2 pb-3">{children}</div> : null}
      {chart ? <div className="mt-auto pt-3">{chart}</div> : null}
    </div>
  );
}

/** A compact card while its number loads, the same size as the real one. */
export function CompactCardSkeleton({ tall = false }: { tall?: boolean }) {
  return (
    <div
      className={cn(
        'flex flex-col gap-2 rounded-lg bg-kumo-base px-4 pt-3 ring ring-kumo-hairline',
        tall ? 'h-60' : 'h-36',
      )}
    >
      <Bone className="h-4 w-20" />
      <Bone className="h-7 w-28" />
      <Bone className="h-4 w-16" />
    </div>
  );
}
