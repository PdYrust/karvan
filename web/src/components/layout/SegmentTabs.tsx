import { cn } from '@cloudflare/kumo/utils';
import { CaretLeftIcon, CaretRightIcon } from '@phosphor-icons/react';
import { useEffect, useState } from 'react';
import { NavLink, useLocation } from 'react-router';
import type { RouteTab } from './RouteTabs';

/** Whether the row can scroll further toward its start or its end, in the page direction. */
function useOverflow(el: HTMLElement | null) {
  const [edges, setEdges] = useState({ start: false, end: false });
  useEffect(() => {
    if (!el) return;
    const read = () => {
      // In RTL scrollLeft runs from 0 down to negative values.
      const moved = Math.abs(el.scrollLeft);
      const room = el.scrollWidth - el.clientWidth;
      setEdges({ start: moved > 1, end: room - moved > 1 });
    };
    read();
    el.addEventListener('scroll', read, { passive: true });
    const ro = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(read);
    ro?.observe(el);
    return () => {
      el.removeEventListener('scroll', read);
      ro?.disconnect();
    };
  }, [el]);
  return edges;
}

const arrow =
  'grid size-8 shrink-0 place-items-center rounded-md text-kumo-subtle hover:bg-kumo-base hover:text-kumo-default pointer-coarse:size-11';

/**
 * Tabs as a segmented control, like the dashboard's analytics: a tinted
 * group with the open tab raised. When the row is wider than the screen it
 * scrolls sideways, with small arrows at its ends.
 */
export function SegmentTabs({
  label,
  tabs,
  scrollStart,
  scrollEnd,
}: {
  label: string;
  tabs: RouteTab[];
  /** Names of the arrows, for screen readers. */
  scrollStart: string;
  scrollEnd: string;
}) {
  const { pathname } = useLocation();
  const [row, setRow] = useState<HTMLDivElement | null>(null);
  const edges = useOverflow(row);
  useEffect(() => {
    row
      ?.querySelector('[aria-current="page"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [row, pathname]);
  const scroll = (toEnd: boolean) => {
    const rtl = row ? getComputedStyle(row).direction === 'rtl' : false;
    row?.scrollBy({ left: (toEnd !== rtl ? 1 : -1) * 160, behavior: 'smooth' });
  };
  return (
    <nav aria-label={label} className="flex min-w-0 items-center gap-1 rounded-lg bg-kumo-tint p-1">
      {edges.start ? (
        <button
          type="button"
          aria-label={scrollStart}
          className={arrow}
          onClick={() => scroll(false)}
        >
          <CaretLeftIcon size={14} aria-hidden="true" className="rtl:-scale-x-100" />
        </button>
      ) : null}
      <div ref={setRow} className="flex min-w-0 gap-1 overflow-x-auto [scrollbar-width:none]">
        {tabs.map((x) => (
          <NavLink
            key={x.to}
            to={x.to}
            end={x.end}
            className={({ isActive }) =>
              cn(
                'flex h-8 shrink-0 items-center gap-2 rounded-md px-3 text-sm whitespace-nowrap pointer-coarse:h-11',
                isActive
                  ? 'bg-kumo-base font-medium text-kumo-strong ring ring-kumo-hairline'
                  : 'text-kumo-subtle hover:text-kumo-default',
              )
            }
          >
            {x.icon}
            <bdi>{x.label}</bdi>
          </NavLink>
        ))}
      </div>
      {edges.end ? (
        <button type="button" aria-label={scrollEnd} className={arrow} onClick={() => scroll(true)}>
          <CaretRightIcon size={14} aria-hidden="true" className="rtl:-scale-x-100" />
        </button>
      ) : null}
    </nav>
  );
}
