import { BarList } from '../../../components/charts/BarList';
import { Legend } from '../../../components/charts/Legend';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { PanelData, PanelStats, RangeKey } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages, type AnalyticsKey } from '../messages';
import { databaseRange } from '../range';

/** Statuses both panels use, in the order an operator reads them. */
const statuses = ['active', 'on_hold', 'limited', 'expired', 'disabled'] as const;

/** Users by status as a bar list, with sign-ups and expiries under it. */
export function StatusCard({ data }: { data: PanelData }) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const known: readonly string[] = statuses;
  const others = Object.keys(data.byStatus).filter((s) => !known.includes(s));
  const items = [...statuses, ...others].map((s) => {
    const n = data.byStatus[s] ?? 0;
    return {
      key: s,
      label: known.includes(s) ? t(`status_${s}` as AnalyticsKey) : <Machine>{s}</Machine>,
      value: n,
      display: fmt.number(n),
    };
  });
  const facts = [
    { label: t('newDay'), value: data.newDay },
    { label: t('newWeek'), value: data.newWeek },
    { label: t('expiringWeek'), value: data.expiringWeek },
  ];
  return (
    <ChartCard title={t('usersTitle')} aside={fmt.number(data.users)}>
      {data.users === 0 ? (
        <p className="text-sm text-kumo-subtle">{t('noUsers')}</p>
      ) : (
        <BarList label={t('usersTitle')} items={items} />
      )}
      <dl className="mt-auto grid grid-cols-3 gap-3 border-t border-kumo-hairline pt-4">
        {facts.map((f) => (
          <div key={f.label} className="flex min-w-0 flex-col">
            <dt className="text-sm text-kumo-subtle">{f.label}</dt>
            <dd className="text-heading font-semibold text-kumo-strong tabular">
              {fmt.number(f.value)}
            </dd>
          </div>
        ))}
      </dl>
    </ChartCard>
  );
}

/** All, active and online users over the range, from Karvan's minute readings. */
export function UsersTrendCard({
  panel,
  data,
  step,
  domain,
  range,
}: {
  panel: PanelStats;
  data: PanelData;
  step: number;
  domain: [number, number];
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const within = t(`in_${databaseRange(range)}`);
  const count = (n: number) => fmt.number(Math.round(n));
  const series = [
    {
      name: t('seriesAll'),
      color: 1 as const,
      values: panel.points.map((p) => p.users),
      value: count(data.users),
    },
    {
      name: t('seriesActive'),
      color: 2 as const,
      values: panel.points.map((p) => p.active),
      value: count(data.byStatus.active ?? 0),
    },
    {
      name: t('seriesOnline'),
      color: 3 as const,
      values: panel.points.map((p) => p.online),
      value: count(data.onlineNow),
    },
  ];
  return (
    <ChartCard title={t('chartUsers')}>
      <Legend items={series} />
      <TimeSeries
        label={t('chartLabel', { chart: t('chartUsers'), range: within })}
        times={panel.points.map((p) => p.t * 1000)}
        series={series}
        domain={domain}
        step={step}
        kind="count"
        format={count}
        empty={t('noData', { range: within })}
      />
    </ChartCard>
  );
}
