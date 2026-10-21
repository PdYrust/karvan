import { Sparkline } from '../../../components/charts/Sparkline';
import { StatTile } from '../../../components/charts/StatTile';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { PanelData, PanelStats } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { engineLabel, isServerEngine } from '../../analytics/databases/engine';
import { analyticsMessages } from '../../analytics/messages';
import { overviewMessages } from '../messages';

interface TilesProps {
  panel: PanelStats;
  data: PanelData;
  step: number;
  domain: [number, number];
}

function sparkOf({ panel, step, domain }: TilesProps) {
  const times = panel.points.map((p) => p.t * 1000);
  return (values: readonly number[]) => (
    <Sparkline times={times} values={values} domain={domain} step={step} />
  );
}

/** A database server: connections in use and queries per second. */
function ServerEngineTiles(props: TilesProps) {
  const t = useT(overviewMessages);
  const ta = useT(analyticsMessages);
  const fmt = useFormat();
  const spark = sparkOf(props);
  const { panel } = props;
  const e = panel.engine;
  return (
    <>
      <StatTile
        label={ta('engineConnections')}
        value={fmt.number(e?.connections ?? 0)}
        sub={ta('ofTotal', { total: fmt.number(e?.maxConnections ?? 0) })}
      >
        {spark(panel.points.map((p) => p.connections))}
      </StatTile>
      <StatTile
        label={ta('engineQps')}
        value={fmt.number(panel.qps, 1)}
        sub={
          panel.cacheHit === null
            ? undefined
            : t('cacheHits', { rate: fmt.percent(panel.cacheHit, 0) })
        }
      >
        {spark(panel.points.map((p) => p.qps))}
      </StatTile>
    </>
  );
}

/** SQLite counts neither connections nor queries: its write-ahead log and the traffic its users used instead. */
function FileEngineTiles(props: TilesProps) {
  const t = useT(overviewMessages);
  const ta = useT(analyticsMessages);
  const fmt = useFormat();
  const spark = sparkOf(props);
  const { panel, data } = props;
  const wal = fmt.bytesParts(panel.engine?.walSize ?? 0);
  const traffic = fmt.bytesParts(data.usedTraffic);
  const mode = panel.engine?.journalMode;
  return (
    <>
      <StatTile
        label={ta('engineWal')}
        value={wal.value}
        unit={wal.unit}
        sub={mode ? <CodeText text={t('journalMode', { mode })} /> : undefined}
      />
      <StatTile
        label={ta('usedTraffic')}
        value={traffic.value}
        unit={traffic.unit}
        sub={ta('adminsCount', { count: data.admins })}
      >
        {spark(panel.points.map((p) => p.usedTraffic))}
      </StatTile>
    </>
  );
}

/** A panel database's size, its engine's own numbers and who is online. */
export function DatabaseTiles(props: TilesProps) {
  const ta = useT(analyticsMessages);
  const fmt = useFormat();
  const spark = sparkOf(props);
  const { panel, data } = props;
  const size = fmt.bytesParts(panel.engine?.size ?? 0);
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatTile
        label={ta('dbSize')}
        value={size.value}
        unit={size.unit}
        sub={<bdi dir="ltr">{engineLabel(panel.engine)}</bdi>}
      >
        {spark(panel.points.map((p) => p.size))}
      </StatTile>
      {isServerEngine(panel.engine) ? (
        <ServerEngineTiles {...props} />
      ) : (
        <FileEngineTiles {...props} />
      )}
      <StatTile
        label={ta('onlineNow')}
        value={fmt.number(data.onlineNow)}
        sub={ta('onlineDay', { count: data.onlineDay })}
      >
        {spark(panel.points.map((p) => p.online))}
      </StatTile>
    </div>
  );
}
