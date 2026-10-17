import { cn } from '@cloudflare/kumo/utils';
import { NavLink } from 'react-router';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';

export interface RouteTab {
  to: string;
  label: string;
  /** Matches only this exact path, for the first tab of a section. */
  end?: boolean;
  /** Not built yet: shown, inert, with a "Soon" tag. */
  soon?: boolean;
}

const tab =
  'relative inline-flex h-11 shrink-0 items-center gap-2 px-3 text-base whitespace-nowrap after:absolute after:inset-x-3 after:-bottom-px after:h-0.5 after:rounded-full';

/** Kumo's underline tabs, one route per tab, so every tab has its own address. */
export function RouteTabs({ label, tabs }: { label: string; tabs: RouteTab[] }) {
  const ts = useT(sharedMessages);
  return (
    <nav aria-label={label} className="flex overflow-x-auto [scrollbar-width:none]">
      {tabs.map((x) =>
        x.soon ? (
          <span
            key={x.to}
            aria-disabled="true"
            className={cn(tab, 'cursor-default text-kumo-inactive')}
          >
            {x.label}
            <span className="rounded-md px-1.5 text-tiny text-kumo-subtle ring ring-kumo-line">
              {ts('soon')}
            </span>
          </span>
        ) : (
          <NavLink
            key={x.to}
            to={x.to}
            end={x.end}
            className={cn(
              tab,
              'text-kumo-subtle hover:text-kumo-default aria-[current=page]:font-medium aria-[current=page]:text-kumo-strong aria-[current=page]:after:bg-kumo-brand',
            )}
          >
            {x.label}
          </NavLink>
        ),
      )}
    </nav>
  );
}
