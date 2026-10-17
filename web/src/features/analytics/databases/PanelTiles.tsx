import { Sparkline } from '../../../components/charts/Sparkline';
import { StatTile } from '../../../components/charts/StatTile';
import { useT } from '../../../i18n/locale';
import type { PanelData, PanelStats } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages } from '../messages';
import { engineLabel } from './engine';

/** Users, who is online, traffic and the database's size, each with its trend. */
export function PanelTiles({
  panel,
  data,
  step,
  domain,
}: {
  panel: PanelStats;
  data: PanelData;
  step: number;
  domain: [number, number];
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const times = panel.points.map((p) => p.t * 1000);
  const spark = (values: readonly number[]) => (
    <Sparkline times={times} values={values} domain={domain} step={step} />
  );
  const traffic = fmt.bytesParts(data.usedTraffic);
  const size = fmt.bytesParts(panel.engine?.size ?? 0);
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatTile
        label={t('users')}
        value={fmt.number(data.users)}
        sub={t('activeCount', { count: data.byStatus.active ?? 0 })}
      >
        {spark(panel.points.map((p) => p.users))}
      </StatTile>
      <StatTile
        label={t('onlineNow')}
        value={fmt.number(data.onlineNow)}
        sub={t('onlineDay', { count: data.onlineDay })}
      >
        {spark(panel.points.map((p) => p.online))}
      </StatTile>
      <StatTile
        label={t('usedTraffic')}
        value={traffic.value}
        unit={traffic.unit}
        sub={t('adminsCount', { count: data.admins })}
      >
        {spark(panel.points.map((p) => p.usedTraffic))}
      </StatTile>
      <StatTile
        label={t('dbSize')}
        value={size.value}
        unit={size.unit}
        sub={<bdi dir="ltr">{engineLabel(panel.engine)}</bdi>}
      >
        {spark(panel.points.map((p) => p.size))}
      </StatTile>
    </div>
  );
}
