import { cn } from '@cloudflare/kumo/utils';
import { useEffect, useRef, type ReactNode } from 'react';
import { NavLink, useLocation } from 'react-router';

export interface RouteTab {
  to: string;
  label: string;
  /** Drawn before the label, such as a panel's logo. */
  icon?: ReactNode;
  /** Matches only this exact path, for the first tab of a section. */
  end?: boolean;
}

const tab =
  'relative inline-flex h-11 shrink-0 items-center gap-2 px-3 text-base whitespace-nowrap after:absolute after:inset-x-3 after:-bottom-px after:h-0.5 after:rounded-full';

/** Kumo's underline tabs, one route per tab, so every tab has its own address. */
export function RouteTabs({ label, tabs }: { label: string; tabs: RouteTab[] }) {
  const ref = useRef<HTMLElement>(null);
  const { pathname } = useLocation();
  // On a phone the row scrolls sideways; the open tab must not sit cut off at the edge.
  useEffect(() => {
    ref.current
      ?.querySelector('[aria-current="page"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [pathname, tabs.length]);
  return (
    <nav ref={ref} aria-label={label} className="flex overflow-x-auto [scrollbar-width:none]">
      {tabs.map((x) => (
        <NavLink
          key={x.to}
          to={x.to}
          end={x.end}
          className={cn(
            tab,
            'text-kumo-subtle hover:text-kumo-default aria-[current=page]:font-medium aria-[current=page]:text-kumo-strong aria-[current=page]:after:bg-kumo-brand',
          )}
        >
          {x.icon}
          {x.label}
        </NavLink>
      ))}
    </nav>
  );
}
