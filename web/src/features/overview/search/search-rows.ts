import { filterCommands, type Command, type CommandGroup } from '../../../app/shell/commands';
/** How many recent pages the search lists before anything is typed. */
export const searchRecentsMax = 5;
/**
 * What the overview search lists: the recent pages while nothing is typed
 * (the pages themselves in a browser that opened none yet), then the
 * palette's groups, found the way the palette finds them.
 */
export function searchGroups(
  recents: CommandGroup,
  groups: CommandGroup[],
  query: string,
): CommandGroup[] {
  if (!query.trim()) {
    const items = recents.items.slice(0, searchRecentsMax);
    return items.length > 0 ? [{ ...recents, items }] : groups.filter((g) => g.id === 'pages');
  }
  return filterCommands(groups, query);
}
/** Every row in order, the way the arrow keys walk them. */
export function searchRows(groups: readonly CommandGroup[]): Command[] {
  return groups.flatMap((g) => g.items);
}
/** The row an arrow key moves to; it wraps at both ends, as the palette does. */
export function moveActive(current: number, key: string, count: number): number | undefined {
  if (count === 0) return undefined;
  if (key === 'ArrowDown') return (current + 1) % count;
  if (key === 'ArrowUp') return (current - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return undefined;
}
