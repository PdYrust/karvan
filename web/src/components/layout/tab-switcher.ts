import type { RouteTab } from './RouteTabs';
/**
 * Whether the segmented tabs show: never on a phone, and on wider screens only
 * while their natural width fits the room the toolbar leaves them.
 */
export function tabsFit({ phone, need, room }: { phone: boolean; need: number; room: number }) {
  return !phone && need > 0 && need <= room;
}
/** A tab's address without its query (the range), as the router matches it. */
const pathOf = (to: string) => to.split(/[?#]/)[0] ?? to;
/** The open tab: the one whose path is the address or a parent of it. */
export function currentTab(tabs: readonly RouteTab[], pathname: string): RouteTab | undefined {
  return tabs.find((x) => {
    const path = pathOf(x.to);
    return pathname === path || (!x.end && pathname.startsWith(`${path}/`));
  });
}
