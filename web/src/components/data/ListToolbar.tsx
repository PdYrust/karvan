import { cn } from '@cloudflare/kumo/utils';
import { XIcon } from '@phosphor-icons/react';
import { FilterIcon } from '../icons/regular';
import { lazy, Suspense, useState, type ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { activeFilters } from '../../lib/lists/choices';
import { useFormat } from '../../lib/preferences/region';
import { ChoiceMenu, type ToolbarChoice } from './ChoiceMenu';
import { dataMessages } from './messages';
import { SearchBar, type ListSearch } from './SearchBar';
import { segment } from './segment';
export type { ChoiceOption, ToolbarChoice } from './ChoiceMenu';
// The sheet only opens on a phone, on a press: it loads then, so lists do not carry it.
const FilterSheet = lazy(() => import('./FilterSheet'));
const square = 'w-9 justify-center';
/**
 * The toolbar across the top of a list, like the dashboard's Workers list: the search bar with a
 * segment per filter and sort. On phones the segments give way to a filter button (with a count
 * while filters are on) that opens a bottom sheet, and a clear filters button.
 */
export function ListToolbar({
  search,
  choices,
  refresh,
  children,
}: {
  search: ListSearch;
  choices: readonly ToolbarChoice[];
  /** The list's refresh, a ToolbarRefresh, last in the group. */
  refresh?: ReactNode;
  children?: ReactNode;
}) {
  const t = useT(dataMessages);
  const fmt = useFormat();
  const [sheet, setSheet] = useState(false);
  const [used, setUsed] = useState(false);
  const active = activeFilters(choices);
  const clear = () => {
    for (const c of choices) if (c.all !== undefined && c.value !== c.all) c.onChange(c.all);
  };
  const filterName = active ? `${t('filter')}, ${t('filtersOn', { count: active })}` : t('filter');
  const segments = (
    <>
      {choices.map((c) => (
        <ChoiceMenu key={c.label} choice={c} className="max-sm:hidden" />
      ))}
      <button
        type="button"
        aria-label={filterName}
        title={filterName}
        onClick={() => {
          setUsed(true);
          setSheet(true);
        }}
        className={cn(segment, square, 'relative sm:hidden')}
      >
        <FilterIcon />
        {active ? (
          <span className="absolute end-0.5 top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-kumo-brand px-1 text-xs leading-none text-on-brand tabular">
            {fmt.number(active)}
          </span>
        ) : null}
      </button>
      {active ? (
        <button
          type="button"
          aria-label={t('clearFilters')}
          title={t('clearFilters')}
          onClick={clear}
          className={cn(segment, square, 'k-fade sm:hidden')}
        >
          <XIcon size={18} aria-hidden="true" />
        </button>
      ) : null}
      {refresh}
    </>
  );
  return (
    <>
      <SearchBar search={search} segments={segments}>
        {children}
      </SearchBar>
      {used ? (
        <Suspense fallback={null}>
          <FilterSheet choices={choices} open={sheet} onOpenChange={setSheet} />
        </Suspense>
      ) : null}
    </>
  );
}
