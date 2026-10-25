import type { ReactNode } from 'react';
import { useT } from '../../../i18n/locale';
import { filterOf, isChangeNode } from '../../../lib/plan/changes';
import type { PlanNode } from '../../../lib/plan/plan';
import { changesMessages, type ChangesKey } from './messages';

/**
 * A category's title as text (for its label) and as shown. The ones the
 * changes made read in the page's language; a filter's value is a machine
 * value kept left to right inside a Persian title.
 */
export function useCategoryTitle() {
  const t = useT(changesMessages);
  return (n: PlanNode): { label: string; shown: ReactNode } => {
    const plain = (label: string) => ({ label, shown: <bdi>{label}</bdi> });
    if (!isChangeNode(n)) return plain(n.name);
    if (n.id === 'change-all') return plain(t('whoAll'));
    const f = filterOf(n);
    if (f) {
      const filter = t(`filter_${f.key}` as ChangesKey);
      const word = f.key === 'status';
      const value = word ? t(f.value as ChangesKey) : f.value;
      return {
        label: t('filterName', { filter, value }),
        shown: (
          <>
            {filter}: {word ? value : <bdi dir="ltr">{value}</bdi>}
          </>
        ),
      };
    }
    const users = n.conditions?.find((c) => c.field === 'users')?.list;
    return plain(users ? t('pickedName', { count: users.length }) : n.name);
  };
}
