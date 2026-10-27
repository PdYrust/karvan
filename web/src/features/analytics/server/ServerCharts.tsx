import { Legend } from '../../../components/charts/Legend';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import { useT } from '../../../i18n/locale';
import type { RangeKey, Sample } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { GroupBy, useGrouping } from '../GroupBy';
import { analyticsMessages } from '../messages';
import type { ServerView } from '../server-view';
import { serverCharts, type ChartSpec } from './charts';

function ServerChart({
  spec,
  view,
  within,
}: {
  spec: ChartSpec;
  view: ServerView;
  within: string;
}) {
  const t = useT(analyticsMessages);
  const [grouped, setGrouped] = useGrouping();
  const series = spec.groups && !grouped ? spec.groups.total : spec.series;
  return (
    <ChartCard
      title={spec.title}
      aside={
        spec.groups ? (
          <GroupBy by={spec.groups.by} grouped={grouped} onChange={setGrouped} />
        ) : undefined
      }
    >
      <Legend items={series} />
      <TimeSeries
        label={t('chartLabel', { chart: spec.title, range: within })}
        times={view.times}
        series={series}
        domain={view.domain}
        step={view.step}
        spacing={view.spacing}
        kind={spec.kind}
        format={spec.format}
        empty={t('noData', { range: within })}
      />
    </ChartCard>
  );
}

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
        <ServerChart key={c.key} spec={c} view={view} within={within} />
      ))}
    </div>
  );
}
