import type { BarItem } from '../../components/charts/BarList';

export type TopSort = 'share' | 'name';

/** A row of a top list, with its name in plain text for search and sort. */
export type TopItem = BarItem & { name: string };

/** The rows a top list shows: those matching the search, largest first or by name, at most `count`. */
export function topRows(
  items: readonly TopItem[],
  { query = '', sort, count }: { query?: string; sort: TopSort; count: number },
): TopItem[] {
  const q = query.trim().toLocaleLowerCase();
  const found = q ? items.filter((i) => i.name.toLocaleLowerCase().includes(q)) : [...items];
  found.sort((a, b) => (sort === 'name' ? a.name.localeCompare(b.name) : b.value - a.value));
  return found.slice(0, count);
}
