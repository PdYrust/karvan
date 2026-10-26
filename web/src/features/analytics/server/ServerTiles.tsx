import { ArrowDownIcon, ArrowUpIcon } from '@phosphor-icons/react';
import { Bar } from '../../../components/charts/Bar';
import { Sparkline } from '../../../components/charts/Sparkline';
import { StatTile } from '../../../components/charts/StatTile';
import { useT } from '../../../i18n/locale';
import type { Sample, ServerStats } from '../../../lib/api/analytics';
import type { Value } from '../../../lib/charts/series';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages } from '../messages';
import type { ServerView } from '../server-view';

/** CPU, memory, the root disk and network now, each with its trend over the range. */
export function ServerTiles({
  stats,
  now,
  view,
}: {
  stats: ServerStats;
  now: Sample;
  view: ServerView;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const root = stats.disks[0];
  const mem = fmt.bytesParts(now.mem.used);
  const net = fmt.bytesParts(now.netRx + now.netTx);
  const netSum = view.netRx.map((rx, i): Value => {
    const tx = view.netTx[i];
    return rx === null || rx === undefined ? null : rx + (tx ?? 0);
  });
  const spark = (values: readonly Value[], max?: number) => (
    <Sparkline times={view.times} values={values} domain={view.domain} step={view.step} max={max} />
  );

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatTile
        label={t('cpu')}
        value={fmt.percent(now.cpu, 0)}
        sub={t('cores', { count: stats.facts.cores })}
      >
        {spark(view.cpu, 1)}
      </StatTile>
      <StatTile
        label={t('memory')}
        value={mem.value}
        unit={mem.unit}
        sub={t('ofTotal', { total: fmt.bytes(now.mem.total) })}
      >
        {spark(view.memUsed, now.mem.total)}
      </StatTile>
      {root ? (
        <StatTile
          label={t('disk')}
          value={fmt.percent(root.used / (root.total || 1), 0)}
          sub={t('diskOf', { used: fmt.bytes(root.used), total: fmt.bytes(root.total) })}
        >
          <Bar grow share={root.used / (root.total || 1)} className="mx-4 mb-4" />
        </StatTile>
      ) : null}
      <StatTile
        label={t('network')}
        value={net.value}
        unit={`${net.unit}/s`}
        sub={
          <span className="flex flex-wrap gap-x-3">
            <span className="flex items-center gap-1">
              <ArrowDownIcon size={12} aria-hidden="true" />
              <span className="sr-only">{t('seriesIn')}</span>
              {fmt.rate(now.netRx)}
            </span>
            <span className="flex items-center gap-1">
              <ArrowUpIcon size={12} aria-hidden="true" />
              <span className="sr-only">{t('seriesOut')}</span>
              {fmt.rate(now.netTx)}
            </span>
          </span>
        }
      >
        {spark(netSum)}
      </StatTile>
    </div>
  );
}
