import { Button } from '@cloudflare/kumo/components/button';
import { CaretLeftIcon, CaretRightIcon } from '@phosphor-icons/react';
import { useT, type Messages } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';

const messages: Messages<'showing' | 'showingAll' | 'previous' | 'next' | 'pages'> = {
  en: {
    showing: 'Showing {from} to {to} of {total}',
    showingAll: { one: 'Showing {count}', other: 'Showing all {count}' },
    previous: 'Previous page',
    next: 'Next page',
    pages: 'Pages',
  },
  fa: {
    showing: 'نمایش {from} تا {to} از {total}',
    showingAll: { one: 'نمایش {count} مورد', other: 'نمایش همه {count} مورد' },
    previous: 'صفحه قبل',
    next: 'صفحه بعد',
    pages: 'صفحه ها',
  },
};

/**
 * The footer of a server-paged list (D5): how much shows, and the way to
 * the pages before and after. Kumo's own pagination has physical classes
 * and carets that do not turn in Persian, so this one is Karvan's.
 */
export function Pager({
  offset,
  shown,
  total,
  perPage,
  onOffset,
}: {
  offset: number;
  /** Rows on this page. */
  shown: number;
  total: number;
  perPage: number;
  onOffset: (offset: number) => void;
}) {
  const t = useT(messages);
  const fmt = useFormat();
  if (total === 0) return null;
  const info =
    total <= perPage && offset === 0
      ? t('showingAll', { count: total })
      : t('showing', {
          from: fmt.number(offset + 1),
          to: fmt.number(offset + shown),
          total: fmt.number(total),
        });
  return (
    <nav aria-label={t('pages')} className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-kumo-subtle tabular" aria-live="polite">
        {info}
      </p>
      {total > perPage ? (
        <div className="flex gap-2">
          <Button
            shape="square"
            aria-label={t('previous')}
            disabled={offset === 0}
            onClick={() => onOffset(Math.max(0, offset - perPage))}
            icon={<CaretLeftIcon size={16} className="rtl:-scale-x-100" />}
            className="pointer-coarse:size-11"
          />
          <Button
            shape="square"
            aria-label={t('next')}
            disabled={offset + shown >= total}
            onClick={() => onOffset(offset + perPage)}
            icon={<CaretRightIcon size={16} className="rtl:-scale-x-100" />}
            className="pointer-coarse:size-11"
          />
        </div>
      ) : null}
    </nav>
  );
}
