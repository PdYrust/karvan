import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretUpDownIcon } from '@phosphor-icons/react';
import { useLayoutEffect, useState, type ReactNode, type RefCallback } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { useMedia } from '../../lib/media/use-media';
import type { RouteTab } from './RouteTabs';
import { SegmentTabs } from './SegmentTabs';
import { currentTab, tabsFit } from './tab-switcher';
/** The width an element is given, kept up to date. */
function useWidth(): [number, RefCallback<HTMLElement>] {
  const [el, setEl] = useState<HTMLElement | null>(null);
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    if (!el) return;
    const read = () => setWidth(el.getBoundingClientRect().width);
    read();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(read);
    ro.observe(el);
    return () => ro.disconnect();
  }, [el]);
  return [width, setEl];
}
/** One button with the open tab; it opens the list of every tab. */
function TabMenu({ label, tabs, icon }: { label: string; tabs: RouteTab[]; icon: ReactNode }) {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const open = currentTab(tabs, pathname);
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={`${label}: ${open?.label ?? ''}`}
            className="flex h-8 max-w-full min-w-0 items-center gap-2 rounded-lg bg-kumo-base px-3 text-sm font-medium text-kumo-strong ring ring-kumo-line hover:bg-kumo-tint pointer-coarse:h-9"
          >
            <span className="shrink-0">{open?.icon ?? icon}</span>
            <bdi className="min-w-0 truncate">{open?.label}</bdi>
            <CaretUpDownIcon size={14} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
          </button>
        }
      />
      <DropdownMenu.Content align="start" className="min-w-56">
        <DropdownMenu.Group>
          <DropdownMenu.Label>{label}</DropdownMenu.Label>
          <DropdownMenu.RadioGroup
            value={open?.to ?? ''}
            onValueChange={(to: string) => void navigate(to)}
          >
            {tabs.map((x) => (
              <DropdownMenu.RadioItem key={x.to} value={x.to} closeOnClick>
                <span className="flex w-full items-center gap-2">
                  <span className="shrink-0">{x.icon ?? icon}</span>
                  <bdi className="min-w-0 truncate">{x.label}</bdi>
                  <span className="ms-auto flex w-4 shrink-0 justify-center">
                    <DropdownMenu.RadioItemIndicator />
                  </span>
                </span>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Group>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
/**
 * Route tabs that stay segmented while they fit and become one button with a
 * list otherwise, and always on phones. Each tab's `to` carries its query, so
 * choosing one keeps it.
 */
export function TabSwitcher({
  label,
  tabs,
  icon,
}: {
  label: string;
  tabs: RouteTab[];
  /** For a tab with no icon of its own, in the button and the list. */
  icon: ReactNode;
}) {
  // Under Tailwind's sm breakpoint.
  const phone = useMedia('(max-width: 39.98rem)');
  const [room, roomRef] = useWidth();
  const [need, needRef] = useWidth();
  const fit = tabsFit({ phone, need, room });
  return (
    <div ref={roomRef} className="relative flex min-w-0 flex-1">
      {/* Laid out out of sight, so the natural width is known while the button shows. */}
      <div
        aria-hidden="true"
        inert
        className="pointer-events-none invisible absolute inset-0 overflow-hidden"
      >
        <div ref={needRef} className="w-max">
          <SegmentTabs label={label} tabs={tabs} />
        </div>
      </div>
      {fit ? (
        <SegmentTabs label={label} tabs={tabs} />
      ) : (
        <TabMenu label={label} tabs={tabs} icon={icon} />
      )}
    </div>
  );
}
