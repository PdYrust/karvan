import { cn } from '@cloudflare/kumo/utils';
import { CaretRightIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';

/**
 * A table row as a phone card, the way the dashboard stacks DNS records: a
 * top line (what the row is, then its badge and menu at the end), the body,
 * and a tinted footer ending in a chevron. The body's link should carry
 * `cardLink`, so the whole card opens it while the menu and buttons keep
 * their own taps.
 */
export function RowCard({
  lead,
  badge,
  menu,
  footer,
  children,
}: {
  lead: ReactNode;
  badge?: ReactNode;
  menu?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
}) {
  return (
    <article className="relative flex flex-col overflow-hidden rounded-lg bg-kumo-base ring ring-kumo-hairline">
      <div className="flex min-h-11 items-center gap-2 ps-4 pe-2 pt-2">
        <div className="flex min-w-0 flex-1 items-center">{lead}</div>
        {badge}
        {/* Stays at its size even without a menu, so badges line up from card to card. */}
        <div className="relative z-10 flex min-w-2 justify-end">{menu}</div>
      </div>
      <div className="flex flex-col gap-3 px-4 pt-1 pb-3">{children}</div>
      {footer ? (
        <div className="flex min-h-11 items-center gap-3 border-t border-kumo-hairline bg-kumo-elevated px-4 py-2 text-sm text-kumo-subtle">
          <div className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-x-3 gap-y-0.5">
            {footer}
          </div>
          <CaretRightIcon size={14} aria-hidden="true" className="shrink-0 rtl:-scale-x-100" />
        </div>
      ) : null}
    </article>
  );
}

/** For the card's main link: its tap area covers the whole card. */
export const cardLink = "after:absolute after:inset-0 after:content-['']";

/** Labelled facts side by side, such as size and when; a `wide` one takes the whole row. */
export function CardFacts({
  facts,
  className,
}: {
  facts: readonly { label: string; value: ReactNode; wide?: boolean }[];
  className?: string;
}) {
  return (
    <dl className={cn('grid grid-cols-3 gap-x-4 gap-y-2 text-sm', className)}>
      {facts.map((f) => (
        <div key={f.label} className={cn('flex min-w-0 flex-col gap-0.5', f.wide && 'col-span-3')}>
          <dt className="text-kumo-subtle">{f.label}</dt>
          <dd className="min-w-0 text-kumo-default">{f.value}</dd>
        </div>
      ))}
    </dl>
  );
}
