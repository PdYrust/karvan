import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { ArrowsOutSimpleIcon, DotsThreeIcon } from '@phosphor-icons/react';
import { useState, type ReactNode } from 'react';
import { BarList } from '../../components/charts/BarList';
import { ListToolbar } from '../../components/data/ListToolbar';
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
  const [query, setQuery] = useState('');
  const [count, setCount] = useState<number>(25);
  const rows = topRows(items, { query, sort, count });
  const top = (
    <ListToolbar
      search={{ label: t('searchList', { list: title }), value: query, onChange: setQuery }}
      choices={[
        {
          label: t('sortBy'),
          value: sort,
          options: (['share', 'name'] as const).map((v) => {
            const text = t(v === 'name' ? 'sortName' : 'sortShare');
            return { value: v, label: text, text };
          }),
          onChange: (v) => onSort(v === 'name' ? 'name' : 'share'),
        },
        {
          label: t('itemCount'),
          value: String(count),
          options: counts.map((n) => {
            const text = t('itemsShown', { count: n });
            return { value: String(n), label: text, text };
          }),
          onChange: (v) => setCount(Number(v) || 25),
        },
      ]}
    />
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
          className="size-7 text-kumo-subtle"
          aria-label={t('showAll')}
          title={t('showAll')}
          aria-haspopup="dialog"
          onClick={() => setOpen(true)}
          icon={<ArrowsOutSimpleIcon size={14} />}
        />
        <DropdownMenu>
          <DropdownMenu.Trigger
            render={
              <Button
                variant="ghost"
                shape="square"
                size="sm"
                className="size-7 text-kumo-subtle"
                aria-label={t('listMenu', { list: title })}
                title={t('listMenu', { list: title })}
                icon={<DotsThreeIcon size={16} weight="bold" />}
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
