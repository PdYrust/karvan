import { cn } from '@cloudflare/kumo/utils';
import { NavLink } from 'react-router';
import type { RouteTab } from './RouteTabs';
/**
 * Tabs as a segmented control, like the dashboard's analytics: a tinted
 * group with the open tab raised. `TabSwitcher` shows it only while it fits.
 */
export function SegmentTabs({ label, tabs }: { label: string; tabs: RouteTab[] }) {
  return (
    <nav aria-label={label} className="flex w-max gap-1 rounded-lg bg-kumo-tint p-1">
      {tabs.map((x) => (
        <NavLink
          key={x.to}
          to={x.to}
          end={x.end}
          className={({ isActive }) =>
            cn(
              'flex h-8 shrink-0 items-center gap-2 rounded-md px-3 text-sm whitespace-nowrap pointer-coarse:h-9',
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
    </nav>
  );
}
