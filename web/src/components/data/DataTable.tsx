import { cn } from '@cloudflare/kumo/utils';
import type { Key, ReactNode } from 'react';

export interface Column<T> {
  key: string;
  header: string;
  /** Keeps the header for screen readers only, as for a column of row menus. */
  hideHeader?: boolean;
  cell: (row: T) => ReactNode;
  /** Applied to the header and every cell of the column, such as a width. */
  className?: string;
}

/**
 * A list the way the dashboard shows one: a ringed frame, a tinted header
 * row and hairlines between rows about 48 px high. Under 640 px every row
 * becomes the card `card` draws, with its main value first.
 */
export function DataTable<T>({
  caption,
  rows,
  rowKey,
  columns,
  card,
}: {
  /** Names the table for screen readers. */
  caption: string;
  rows: readonly T[];
  rowKey: (row: T) => Key;
  columns: readonly Column<T>[];
  card: (row: T) => ReactNode;
}) {
  return (
    <>
      <div className="hidden overflow-x-auto rounded-lg bg-kumo-base ring ring-kumo-hairline sm:block">
        <table className="w-full text-start">
          <caption className="sr-only">{caption}</caption>
          <thead>
            <tr className="bg-kumo-elevated">
              {columns.map((c) => (
                <th
                  key={c.key}
                  scope="col"
                  className={cn(
                    'px-4 py-2.5 text-start text-sm font-medium whitespace-nowrap text-kumo-subtle',
                    c.className,
                  )}
                >
                  {c.hideHeader ? <span className="sr-only">{c.header}</span> : c.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={rowKey(r)} className="k-fade border-t border-kumo-hairline">
                {columns.map((c) => (
                  <td key={c.key} className={cn('px-4 py-3 align-middle', c.className)}>
                    {c.cell(r)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ul aria-label={caption} className="flex flex-col gap-3 sm:hidden">
        {rows.map((r) => (
          <li key={rowKey(r)} className="k-fade">
            {card(r)}
          </li>
        ))}
      </ul>
    </>
  );
}
