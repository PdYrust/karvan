import type { ToolbarChoice } from '../../components/data/ListToolbar';
import { useT } from '../../i18n/locale';
import { batchesMessages } from './messages';
import { batchStateFilters, type BatchStateFilter } from './use-batches';

/** The toolbar's state choice, for every old panel or for one. */
export function useStateChoice(
  state: BatchStateFilter,
  setState: (s: BatchStateFilter) => void,
): ToolbarChoice {
  const t = useT(batchesMessages);
  return {
    label: t('stateLabel'),
    value: state,
    all: 'all',
    options: batchStateFilters.map((s) => {
      const text = t(`filter_${s}`);
      return { value: s, label: text, text };
    }),
    onChange: (s) => setState(s as BatchStateFilter),
  };
}
