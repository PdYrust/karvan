/** What a paged list shows: its skeleton, the empty card alone, or the toolbar with the rows. */
export type ListView = 'loading' | 'empty' | 'list';

/**
 * The skeleton until the first page is there (`useFirstLoad`), then the empty
 * card alone when nothing exists at all. A narrowed list (a filter, a search,
 * a later page, or a page still on its old data) keeps its toolbar even when
 * nothing matches, so the narrowing can be undone.
 */
export function listView(rows: readonly unknown[] | undefined, narrowed: boolean): ListView {
  if (!rows) return 'loading';
  return rows.length === 0 && !narrowed ? 'empty' : 'list';
}
