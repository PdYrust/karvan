/** One choice in a list toolbar: a filter, or a sort when it has no `all`. */
export interface Choice {
  value: string;
  /** The value that filters nothing; missing for a sort. */
  all?: string;
}
/** How many filters narrow the list now; a sort never counts. */
export function activeFilters(choices: readonly Choice[]): number {
  return choices.filter((c) => c.all !== undefined && c.value !== c.all).length;
}
/** How a list orders its rows. */
export type ListSort = 'newest' | 'name' | 'largest';
/** What a sort reads from a row; a sort the row lacks keeps the list order. */
export interface SortKeys<T> {
  name?: (row: T) => string;
  /** A time the API sends (ISO) or an id that grows with it. */
  time?: (row: T) => string | number;
  size?: (row: T) => number;
}
const asNumber = (v: string | number): number => (typeof v === 'number' ? v : Date.parse(v) || 0);
/** The rows in `sort` order, a new array; names compare in the page's language. */
export function sortRows<T>(
  rows: readonly T[],
  sort: ListSort,
  keys: SortKeys<T>,
  locale: string,
): T[] {
  const out = [...rows];
  const { name, time, size } = keys;
  if (sort === 'name' && name) {
    const collator = new Intl.Collator(locale, { sensitivity: 'base', numeric: true });
    return out.sort((a, b) => collator.compare(name(a), name(b)));
  }
  if (sort === 'newest' && time) return out.sort((a, b) => asNumber(time(b)) - asNumber(time(a)));
  if (sort === 'largest' && size) return out.sort((a, b) => size(b) - size(a));
  return out;
}
/** A search term matched against a row's words, ignoring case; an empty term matches all. */
export function matchesTerm(term: string, words: readonly (string | undefined | null)[]): boolean {
  const q = term.trim().toLocaleLowerCase();
  if (!q) return true;
  return words.some((w) => w?.toLocaleLowerCase().includes(q));
}
