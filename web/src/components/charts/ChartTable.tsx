import { spreadIndexes, type Value } from '../../lib/charts/series';
import { useT } from '../../i18n/locale';
import type { ChartSeries } from './TimeSeries';
import { chartMessages } from './messages';

/** Rows a summary table keeps; enough to follow the shape of a long series. */
const maxRows = 12;

/** The chart as a table for screen readers: times spread over the range, every series. */
export function ChartTable({
  caption,
  times,
  series,
  timeLabel,
  valueText,
}: {
  caption: string;
  times: readonly number[];
  series: readonly ChartSeries[];
  timeLabel: (t: number) => string;
  valueText: (v: Value) => string;
}) {
  const t = useT(chartMessages);
  return (
    <table className="sr-only">
      <caption>{caption}</caption>
      <thead>
        <tr>
          <th scope="col">{t('time')}</th>
          {series.map((s) => (
            <th key={s.name} scope="col">
              {s.name}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {spreadIndexes(times.length, maxRows).map((i) => (
          <tr key={times[i]}>
            <th scope="row">{timeLabel(times[i] as number)}</th>
            {series.map((s) => (
              <td key={s.name}>{valueText(s.values[i])}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
