import { Legend } from '../../../components/charts/Legend';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import { useT } from '../../../i18n/locale';
import type { RangeKey, Sample } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages } from '../messages';
import type { ServerView } from '../server-view';
import { serverCharts } from './charts';

/** CPU, load, memory, network, disk activity and connections over the range. */
export function ServerCharts({
  now,
  view,
  range,
}: {
  now: Sample;
  view: ServerView;
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const within = t(`in_${range}`);
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {serverCharts(view, now, t, fmt).map((c) => (
        <ChartCard key={c.key} title={c.title}>
          <Legend items={c.series} />
          <TimeSeries
            label={t('chartLabel', { chart: c.title, range: within })}
            times={view.times}
            series={c.series}
            domain={view.domain}
            step={view.step}
            kind={c.kind}
            format={c.format}
            empty={t('noData', { range: within })}
          />
        </ChartCard>
      ))}
    </div>
  );
}
