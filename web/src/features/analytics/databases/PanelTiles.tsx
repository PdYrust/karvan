import { CompactCard } from '../../../components/charts/CompactCard';
import { CompactChart } from '../../../components/charts/CompactChart';
import { useT } from '../../../i18n/locale';
import type { PanelData, PanelStats, RangeKey } from '../../../lib/api/analytics';
import { changeOf } from '../../../lib/charts/change';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages } from '../messages';
import { databaseRange } from '../range';
import { rowChart } from '../server/ServerTiles';
import { engineLabel } from './engine';

/**
 * Users, who is online, traffic and the database's size, each with its
 * trend over the range. Not pressable: each has its detailed chart below.
 */
export function PanelTiles({
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
  const times = panel.points.map((p) => p.t * 1000);
  const series = (pick: (p: PanelStats['points'][number]) => number) => panel.points.map(pick);
  const chart = (values: readonly number[]) => (
    <CompactChart times={times} values={values} domain={domain} step={step} className={rowChart} />
  );
  const users = series((p) => p.users);
  const online = series((p) => p.online);
  const used = series((p) => p.usedTraffic);
  const sizes = series((p) => p.size);
  const traffic = fmt.bytesParts(data.usedTraffic);
  const size = fmt.bytesParts(panel.engine?.size ?? 0);
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <CompactCard
        title={t('users')}
        value={fmt.number(data.users)}
        within={within}
        change={changeOf(users)}
        sub={t('activeCount', { count: data.byStatus.active ?? 0 })}
        chart={chart(users)}
      />
      <CompactCard
        title={t('onlineNow')}
        value={fmt.number(data.onlineNow)}
        within={within}
        change={changeOf(online)}
        sub={t('onlineDay', { count: data.onlineDay })}
        chart={chart(online)}
      />
      <CompactCard
        title={t('usedTraffic')}
        value={traffic.value}
        unit={traffic.unit}
        within={within}
        change={changeOf(used)}
        sub={t('adminsCount', { count: data.admins })}
        chart={chart(used)}
      />
      <CompactCard
        title={t('dbSize')}
        value={size.value}
        unit={size.unit}
        within={within}
        change={changeOf(sizes)}
        sub={<bdi dir="ltr">{engineLabel(panel.engine)}</bdi>}
        chart={chart(sizes)}
      />
    </div>
  );
}
