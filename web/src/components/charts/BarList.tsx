import type { ReactNode } from 'react';
import { Bar } from './Bar';

export interface BarItem {
  key: string;
  label: ReactNode;
  value: number;
  /** The value as text, such as "1,204" or "12 GB". */
  display: string;
  /** A second, quieter line under the label. */
  hint?: ReactNode;
}

/**
 * Label, a thin meter and the number, one row each (users by status, top
 * admins, largest tables). The longest bar fills its column.
 */
export function BarList({ items, label }: { items: readonly BarItem[]; label: string }) {
  const max = Math.max(0, ...items.map((i) => i.value)) || 1;
  return (
    <ul aria-label={label} className="flex flex-col">
      {items.map((item) => (
        // On a phone the bar moves under the label, which keeps the whole row for the words.
        <li
          key={item.key}
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 py-2 sm:grid-cols-[minmax(0,1fr)_minmax(3rem,35%)_auto]"
        >
          <span className="flex min-w-0 flex-col">
            <span className="truncate text-kumo-default">{item.label}</span>
            {item.hint ? (
              <span className="truncate text-sm text-kumo-subtle">{item.hint}</span>
            ) : null}
          </span>
          <Bar
            share={item.value / max}
            grow
            className="order-last col-span-2 sm:order-none sm:col-span-1"
          />
          <span className="min-w-12 text-end font-medium text-kumo-strong tabular">
            {item.display}
          </span>
        </li>
      ))}
    </ul>
  );
}
