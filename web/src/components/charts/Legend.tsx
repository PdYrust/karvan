import { chartColor, type SeriesColor } from './palette';

export interface LegendItem {
  name: string;
  color: SeriesColor;
  /** The series' current or total value, in words and digits. */
  value?: string;
  dashed?: boolean;
}

/** Series names with their colors and values, above a chart, so no number lives only in the drawing. */
export function Legend({ items }: { items: readonly LegendItem[] }) {
  return (
    <ul className="flex flex-wrap gap-x-6 gap-y-2">
      {items.map((item) => (
        <li key={item.name} className="flex min-w-0 flex-col">
          <span className="flex items-center gap-2 text-sm text-kumo-subtle">
            <span
              aria-hidden="true"
              className="h-0.5 w-3 shrink-0 rounded-full"
              style={{
                background: item.dashed
                  ? `repeating-linear-gradient(90deg, ${chartColor(item.color)} 0 3px, transparent 3px 5px)`
                  : chartColor(item.color),
              }}
            />
            {item.name}
          </span>
          {item.value ? (
            <span className="ps-5 text-base font-medium text-kumo-strong tabular">
              {item.value}
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
