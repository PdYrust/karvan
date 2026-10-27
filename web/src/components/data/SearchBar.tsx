import type { ReactNode } from 'react';
import { SearchField } from './SearchField';
export interface ListSearch {
  label: string;
  value: string;
  onChange: (q: string) => void;
}
/**
 * The joined group of a list toolbar inside the overview search's halo: the search, then
 * `segments` (filters, sorts, refresh), every seam shared. `children` (actions such as +) sit after the
 * halo. A list with filters uses ListToolbar, which fills the segments.
 */
export function SearchBar({
  search,
  segments,
  children,
}: {
  search: ListSearch;
  segments?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex items-center gap-2">
      <div className="min-w-0 flex-1 rounded-xl bg-kumo-tint p-1 ring ring-kumo-line">
        <div className="k-joined flex h-9 rounded-lg bg-kumo-base ring ring-kumo-line">
          <SearchField label={search.label} value={search.value} onChange={search.onChange} />
          {segments}
        </div>
      </div>
      {children}
    </div>
  );
}
