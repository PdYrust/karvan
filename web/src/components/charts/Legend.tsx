import { useT } from '../../i18n/locale';
import { chartMessages } from './messages';
import { chartColor, type SeriesColor } from './palette';

export interface LegendItem {
  name: string;
  color: SeriesColor;
  /** The series' current or total value, in words and digits. */
  value?: string;
  dashed?: boolean;
}

/**
 * Above a chart, like the dashboard's: a colored dot, the series name and
 * its number in bold beside it ("Download 4.5 GB"), so no number lives only
 * in the drawing. A dashed series (a peak) has a hollow dot.
 */
export function Legend({ items, more = 0 }: { items: readonly LegendItem[]; more?: number }) {
  const t = useT(chartMessages);
  return (
    <ul className="flex flex-wrap gap-x-5 gap-y-1">
      {items.map((item) => (
        <li key={item.name} className="flex min-w-0 items-center gap-1.5 text-sm">
          <span
            aria-hidden="true"
            className="size-2 shrink-0 rounded-full border-2"
            style={{
              borderColor: chartColor(item.color),
              background: item.dashed ? 'transparent' : chartColor(item.color),
            }}
          />
          <bdi className="truncate text-kumo-default">{item.name}</bdi>
          {item.value ? (
            <span className="font-semibold text-kumo-strong tabular">{item.value}</span>
          ) : null}
        </li>
      ))}
      {more > 0 ? (
        <li className="text-sm text-kumo-subtle">{t('moreSeries', { count: more })}</li>
      ) : null}
    </ul>
  );
}
