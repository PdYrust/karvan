import { ArrowDownIcon, ArrowUpIcon } from '@phosphor-icons/react';
import { Bar } from '../../../components/charts/Bar';
import { CompactCard } from '../../../components/charts/CompactCard';
import { CompactChart } from '../../../components/charts/CompactChart';
import { useT } from '../../../i18n/locale';
import type { RangeKey, Sample, ServerStats } from '../../../lib/api/analytics';
import { changeOf } from '../../../lib/charts/change';
import { sumValues, type Value } from '../../../lib/charts/series';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages } from '../messages';
import type { ServerView } from '../server-view';

/** The height of a compact card's chart in the row at the top of a tab: taller on phones, where cards stack. */
export const rowChart = 'h-18 sm:h-14';

/**
 * CPU, memory, the root disk, network and connections now, each with its
 * trend over the range. Not pressable: each has its detailed chart below.
 */
export function ServerTiles({
  stats,
  now,
  view,
  range,
}: {
  stats: ServerStats;
  now: Sample;
  view: ServerView;
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const within = t(`in_${range}`);
  const root = stats.disks[0];
  const mem = fmt.bytesParts(now.mem.used);
  const net = fmt.bytesParts(now.netRx + now.netTx);
  const netSum = sumValues([view.netRx, view.netTx]);
  const chart = (values: readonly Value[], max?: number) => (
    <CompactChart
      times={view.times}
      values={values}
      domain={view.domain}
      step={view.step}
      spacing={view.spacing}
      max={max}
      className={rowChart}
    />
  );
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <CompactCard
        title={t('cpu')}
        value={fmt.percent(now.cpu, 0)}
        within={within}
        change={changeOf(view.cpu, { points: true, risingIsBad: true })}
        sub={t('cores', { count: stats.facts.cores })}
        chart={chart(view.cpu, 1)}
      />
      <CompactCard
        title={t('memory')}
        value={mem.value}
        unit={mem.unit}
        within={within}
        change={changeOf(view.memUsed, { risingIsBad: true })}
        sub={t('ofTotal', { total: fmt.bytes(now.mem.total) })}
        chart={chart(view.memUsed, now.mem.total)}
      />
      {root ? (
        <CompactCard
          title={t('disk')}
          value={fmt.percent(root.used / (root.total || 1), 0)}
          within={within}
          sub={t('diskOf', { used: fmt.bytes(root.used), total: fmt.bytes(root.total) })}
        >
          <Bar grow share={root.used / (root.total || 1)} className="mt-2" />
        </CompactCard>
      ) : null}
      <CompactCard
        title={t('network')}
        value={net.value}
        unit={`${net.unit}/s`}
        within={within}
        change={changeOf(netSum)}
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
        chart={chart(netSum)}
      />
      <CompactCard
        title={t('chartTcp')}
        value={fmt.number(now.tcp)}
        within={within}
        change={changeOf(view.tcp, { risingIsBad: true })}
        sub={t('closingCount', { count: now.tcpTimeWait })}
        chart={chart(view.tcp)}
      />
    </div>
  );
}
