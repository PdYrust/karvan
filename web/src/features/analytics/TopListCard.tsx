import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { Input } from '@cloudflare/kumo/components/input';
import { Select } from '@cloudflare/kumo/components/select';
import { ArrowsOutSimpleIcon, DotsThreeIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { useId, useState, type ReactNode } from 'react';
import { BarList } from '../../components/charts/BarList';
import { SheetDialog } from '../../components/dialogs/SheetDialog';
import { useT } from '../../i18n/locale';
import { ChartCard } from './ChartCard';
import { analyticsMessages } from './messages';
import { topRows, type TopItem, type TopSort } from './top-list';

/** How many rows the card shows; the full list has the rest. */
const cardRows = 6;
const counts = [10, 25, 50, 100] as const;

function FullList({
  title,
  items,
  sort,
  onSort,
  open,
  onOpenChange,
}: {
  title: string;
  items: readonly TopItem[];
  sort: TopSort;
  onSort: (s: TopSort) => void;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const t = useT(analyticsMessages);
  const id = useId();
  const [query, setQuery] = useState('');
  const [count, setCount] = useState<number>(25);
  const rows = topRows(items, { query, sort, count });
  const sorts = { share: t('sortShare'), name: t('sortName') };
  const sizes = Object.fromEntries(counts.map((n) => [String(n), t('itemsShown', { count: n })]));
  // Joined on desktop: the search, then the two selects, as one control.
  const top = (
    <div className="flex flex-col gap-2 sm:flex-row sm:gap-0">
      <div className="relative min-w-0 flex-1">
        <MagnifyingGlassIcon
          size={16}
          aria-hidden="true"
          className="pointer-events-none absolute start-3 top-1/2 -translate-y-1/2 text-kumo-subtle"
        />
        <Input
          type="search"
          aria-label={t('searchList', { list: title })}
          placeholder={t('searchList', { list: title })}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          autoComplete="off"
          className="w-full ps-9 sm:rounded-e-none pointer-coarse:h-9"
        />
      </div>
      <div className="grid grid-cols-2 gap-2 sm:flex sm:gap-0">
        <span id={`${id}-s`} className="sr-only">
          {t('sortBy')}
        </span>
        <Select
          aria-labelledby={`${id}-s`}
          className="w-full sm:-ms-px sm:w-36 sm:rounded-none pointer-coarse:h-9"
          value={sort}
          items={sorts}
          onValueChange={(v) => onSort(v === 'name' ? 'name' : 'share')}
        />
        <span id={`${id}-c`} className="sr-only">
          {t('itemCount')}
        </span>
        <Select
          aria-labelledby={`${id}-c`}
          className="w-full sm:-ms-px sm:w-32 sm:rounded-s-none pointer-coarse:h-9"
          value={String(count)}
          items={sizes}
          onValueChange={(v) => setCount(Number(v) || 25)}
        />
      </div>
    </div>
  );
  return (
    <SheetDialog open={open} onOpenChange={onOpenChange} title={title} top={top}>
      {rows.length > 0 ? (
        <BarList label={title} items={rows} />
      ) : (
        <p className="text-sm text-kumo-subtle">{t('listEmpty')}</p>
      )}
    </SheetDialog>
  );
}

/**
 * A "Top ..." list like the dashboard's: the largest rows with a thin
 * meter, an expand button for the full list and a "..." menu to sort.
 * Without rows it shows `empty` instead.
 */
export function TopListCard({
  title,
  items,
  empty,
}: {
  title: string;
  items: readonly TopItem[];
  empty?: ReactNode;
}) {
  const t = useT(analyticsMessages);
  const [sort, setSort] = useState<TopSort>('share');
  const [open, setOpen] = useState(false);
  const controls =
    items.length > 0 ? (
      <span className="-me-2 flex items-center">
        <Button
          variant="ghost"
          shape="square"
          size="sm"
          className="text-kumo-subtle pointer-coarse:size-9"
          aria-label={t('showAll')}
          title={t('showAll')}
          aria-haspopup="dialog"
          onClick={() => setOpen(true)}
          icon={<ArrowsOutSimpleIcon size={16} />}
        />
        <DropdownMenu>
          <DropdownMenu.Trigger
            render={
              <Button
                variant="ghost"
                shape="square"
                size="sm"
                className="text-kumo-subtle pointer-coarse:size-9"
                aria-label={t('listMenu', { list: title })}
                title={t('listMenu', { list: title })}
                icon={<DotsThreeIcon size={18} weight="bold" />}
              />
            }
          />
          <DropdownMenu.Content align="end" className="min-w-40">
            <DropdownMenu.RadioGroup value={sort} onValueChange={(v: TopSort) => setSort(v)}>
              {(['share', 'name'] as const).map((k) => (
                <DropdownMenu.RadioItem key={k} value={k} closeOnClick>
                  <span className="flex w-full items-center gap-2">
                    {t(k === 'share' ? 'sortShare' : 'sortName')}
                    <span className="ms-auto">
                      <DropdownMenu.RadioItemIndicator />
                    </span>
                  </span>
                </DropdownMenu.RadioItem>
              ))}
            </DropdownMenu.RadioGroup>
          </DropdownMenu.Content>
        </DropdownMenu>
      </span>
    ) : undefined;
  return (
    <ChartCard title={title} aside={controls}>
      {items.length > 0 ? (
        <BarList label={title} items={topRows(items, { sort, count: cardRows })} />
      ) : (
        empty
      )}
      <FullList
        title={title}
        items={items}
        sort={sort}
        onSort={setSort}
        open={open}
        onOpenChange={setOpen}
      />
    </ChartCard>
  );
}
