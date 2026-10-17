import { Legend } from '../../../components/charts/Legend';
import type { SeriesColor } from '../../../components/charts/palette';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import { useT } from '../../../i18n/locale';
import type { PanelDetail, RangeKey } from '../../../lib/api/analytics';
import type { AxisKind } from '../../../lib/charts/ticks';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages } from '../messages';
import { databaseRange, rangeSpan } from '../range';
import { bucketSize, day, hour, sumBuckets, type Buckets } from './history';

interface BarSeries {
  name: string;
  color: SeriesColor;
  values: number[];
  /** The total over the range, for the legend. */
  value: string;
}

/** Bars per hour or day; a series with nothing in it is drawn as missing, so the card says so. */
function BarsCard({
  title,
  buckets,
  series,
  kind,
  format,
  within,
  missing,
}: {
  title: string;
  buckets: Buckets;
  series: BarSeries[];
  kind: AxisKind;
  format: (n: number) => string;
  within: string;
  missing: string | undefined;
}) {
  const t = useT(analyticsMessages);
  const drawn = series.map((s) => ({
    ...s,
    values: s.values.every((v) => v === 0) ? s.values.map(() => null) : s.values,
  }));
  const first = buckets.times[0] ?? 0;
  const last = buckets.times.at(-1) ?? first;
  return (
    <ChartCard title={title}>
      {missing ? (
        <p className="text-sm text-kumo-subtle">{missing}</p>
      ) : (
        <>
          <Legend items={series} />
          <TimeSeries
            label={t('chartLabel', { chart: title, range: within })}
            times={buckets.times}
            series={drawn}
            domain={[first, last + buckets.step]}
            step={buckets.step}
            kind={kind}
            format={format}
            empty={t('noData', { range: within })}
            bars
          />
        </>
      )}
    </ChartCard>
  );
}

const sum = (xs: readonly number[] = []) => xs.reduce((a, b) => a + b, 0);

/** Traffic from the panel's own usage table and new users, per hour for a day, per day beyond. */
export function HistoryCards({
  detail,
  missing,
  end,
  range,
}: {
  detail: PanelDetail | undefined;
  /** Why the detail is missing, in words. */
  missing: string | undefined;
  end: number;
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const r = databaseRange(range);
  // The detail covers at least a day: one hour of traffic would be a single bar.
  const span = Math.max(rangeSpan[r], day);
  const within = t(span === day ? 'in_24h' : `in_${r}`);
  const size = bucketSize(span);
  const bucket = (items: { at: number; values: number[] }[], n: number) =>
    sumBuckets(items, end - span, end, size, fmt.timeZone, n);
  const traffic = bucket(
    (detail?.traffic ?? []).map((h) => ({ at: Date.parse(h.at), values: [h.downlink, h.uplink] })),
    2,
  );
  const signUps = bucket(
    (detail?.signUps ?? []).map((c) => ({ at: Date.parse(c.at), values: [c.n] })),
    1,
  );
  const down = traffic.series[0] ?? [];
  const up = traffic.series[1] ?? [];
  const joined = signUps.series[0] ?? [];
  const count = (n: number) => fmt.number(n);
  return (
    <>
      <BarsCard
        title={t(size === hour ? 'trafficTitle' : 'trafficPerDay')}
        buckets={traffic}
        series={[
          { name: t('seriesDownload'), color: 1, values: down, value: fmt.bytes(sum(down)) },
          { name: t('seriesUpload'), color: 2, values: up, value: fmt.bytes(sum(up)) },
        ]}
        kind="bytes"
        format={fmt.bytes}
        within={within}
        missing={missing}
      />
      <BarsCard
        title={t('signUpsTitle')}
        buckets={signUps}
        series={[{ name: t('seriesSignUps'), color: 1, values: joined, value: count(sum(joined)) }]}
        kind="count"
        format={count}
        within={within}
        missing={missing}
      />
    </>
  );
}
