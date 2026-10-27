import type { Translate } from '../../i18n/core';
import type { ListSort } from '../../lib/lists/choices';
import type { ToolbarChoice } from './ChoiceMenu';
import type { DataKey } from './messages';
const words = { newest: 'sortNewest', name: 'sortName', largest: 'sortLargest' } as const;
/** The "Sort by" choice of a list toolbar, offering `sorts` in that order. */
export function sortChoice(
  td: Translate<DataKey>,
  value: ListSort,
  onChange: (s: ListSort) => void,
  sorts: readonly ListSort[],
): ToolbarChoice {
  return {
    label: td('sortBy'),
    value,
    options: sorts.map((s) => ({ value: s, label: td(words[s]), text: td(words[s]) })),
    onChange: (s) => onChange(s as ListSort),
  };
}
