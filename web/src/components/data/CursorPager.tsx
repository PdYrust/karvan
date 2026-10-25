import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { PageButtons, pagerMessages } from './Pager';

/**
 * The footer of a list the server pages by cursor, which knows no total:
 * "Showing 21 to 40", with the pages before and after.
 */
export function CursorPager({
  offset,
  shown,
  hasNext,
  onPrevious,
  onNext,
}: {
  offset: number;
  shown: number;
  hasNext: boolean;
  onPrevious: () => void;
  onNext: () => void;
}) {
  const t = useT(pagerMessages);
  const fmt = useFormat();
  if (shown === 0) return null;
  const paged = offset > 0 || hasNext;
  return (
    <nav aria-label={t('pages')} className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-kumo-subtle tabular" aria-live="polite">
        {paged
          ? t('range', { from: fmt.number(offset + 1), to: fmt.number(offset + shown) })
          : t('showingAll', { count: shown })}
      </p>
      {paged ? (
        <PageButtons
          hasPrevious={offset > 0}
          hasNext={hasNext}
          onPrevious={onPrevious}
          onNext={onNext}
        />
      ) : null}
    </nav>
  );
}
