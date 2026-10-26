import type { BarItem } from '../../../components/charts/BarList';
import type { SeriesColor } from '../../../components/charts/palette';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { ServerStats } from '../../../lib/api/analytics';
import { changeOf, latestOf, type Change } from '../../../lib/charts/change';
import { sumValues, type Value } from '../../../lib/charts/series';
import type { AxisKind } from '../../../lib/charts/ticks';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages } from '../../analytics/messages';
import { rangeSpan } from '../../analytics/range';
import { serverView } from '../../analytics/server-view';
import { overviewMessages } from '../messages';

/** At most this many series in a detail; the rest are counted as "+N more". */
const seriesMax = 5;

export interface DetailSeries {
  name: string;
  color: SeriesColor;
  values: readonly Value[];
  /** The latest value, in words and digits. */
  value: string;
}

/** One card of the overview: a line, or a list, and what its detail shows. */
export interface ServerCard {
  id: string;
  title: string;
  value?: string;
  unit?: string;
  change?: Change;
  sub?: string;
  chart?: { values: readonly Value[]; max?: number };
  list?: readonly BarItem[];
  detail?: {
    kind: AxisKind;
    format: (v: number) => string;
    series: readonly DetailSeries[];
    more: number;
    contributors: readonly BarItem[];
  };
}

/** CPU, memory, network and the disks of this server, live. */
export function useServerCards(stats: ServerStats | undefined) {
  const t = useT(analyticsMessages);
  const tOverview = useT(overviewMessages);
  const fmt = useFormat();
  const now = stats?.now;
  if (!stats || !now) return undefined;
  const view = serverView(stats, rangeSpan.live);
  const pct = (v: number) => fmt.percent(v, 0);
  const named = (
    name: string,
    color: SeriesColor,
    values: readonly Value[],
    f: (v: number) => string,
  ): DetailSeries => {
    const v = latestOf(values);
    return { name, color, values, value: v === undefined ? '' : f(v) };
  };
  const cpuSeries =
    view.cores.length > 0
      ? view.cores.map((c, i) =>
          named(t('seriesCore', { n: i + 1 }), ((i % seriesMax) + 1) as SeriesColor, c, pct),
        )
      : [
          named(t('seriesAllCores'), 1, view.cpu, pct),
          named(t('seriesBusiest'), 2, view.busiest, pct),
        ];
  const mem = fmt.bytesParts(now.mem.used);
  const netSum = sumValues([view.netRx, view.netTx]);
  const net = fmt.bytesParts(now.netRx + now.netTx);
  const root = stats.disks[0];
  const share = (used: number, total: number) => used / (total || 1);
  const cards: ServerCard[] = [
    {
      id: 'cpu',
      title: t('cpu'),
      value: pct(now.cpu),
      sub: t('cores', { count: stats.facts.cores }),
      change: changeOf(view.cpu, { points: true, risingIsBad: true }),
      chart: { values: view.cpu, max: 1 },
      detail: {
        kind: 'percent',
        format: pct,
        series: cpuSeries.slice(0, seriesMax),
        more: Math.max(0, cpuSeries.length - seriesMax),
        contributors: now.cores
          .map((c, i) => ({
            key: String(i),
            label: t('seriesCore', { n: i + 1 }),
            value: c,
            display: pct(c),
          }))
          .sort((a, b) => b.value - a.value),
      },
    },
    {
      id: 'memory',
      title: t('memory'),
      value: mem.value,
      unit: mem.unit,
      sub: t('ofTotal', { total: fmt.bytes(now.mem.total) }),
      change: changeOf(view.memUsed, { risingIsBad: true }),
      chart: { values: view.memUsed, max: now.mem.total },
      detail: {
        kind: 'bytes',
        format: fmt.bytes,
        more: 0,
        series: [
          named(t('seriesUsed'), 1, view.memUsed, fmt.bytes),
          named(t('seriesCache'), 2, view.memCache, fmt.bytes),
          ...(now.mem.swapTotal > 0 ? [named(t('seriesSwap'), 3, view.swap, fmt.bytes)] : []),
        ],
        contributors: [
          { key: 'used', label: t('seriesUsed'), value: now.mem.used },
          { key: 'cache', label: t('seriesCache'), value: now.mem.cache },
          { key: 'available', label: tOverview('memAvailable'), value: now.mem.available },
          ...(now.mem.swapTotal > 0
            ? [{ key: 'swap', label: t('seriesSwap'), value: now.mem.swapUsed }]
            : []),
        ].map((p) => ({ ...p, display: fmt.bytes(p.value) })),
      },
    },
    {
      id: 'network',
      title: t('network'),
      value: net.value,
      unit: `${net.unit}/s`,
      sub: t('netNow', { rx: fmt.rate(now.netRx), tx: fmt.rate(now.netTx) }),
      change: changeOf(netSum),
      chart: { values: netSum },
      detail: {
        kind: 'rate',
        format: fmt.rate,
        more: 0,
        series: [
          named(t('seriesIn'), 1, view.netRx, fmt.rate),
          named(t('seriesOut'), 2, view.netTx, fmt.rate),
        ],
        contributors: now.ifaces
          .map((f) => ({
            key: f.name,
            label: <Machine>{f.name}</Machine>,
            hint: t('netNow', { rx: fmt.rate(f.rx), tx: fmt.rate(f.tx) }),
            value: f.rx + f.tx,
            display: fmt.rate(f.rx + f.tx),
          }))
          .sort((a, b) => b.value - a.value),
      },
    },
    {
      id: 'disks',
      title: t('disksTitle'),
      value: root ? pct(share(root.used, root.total)) : undefined,
      sub: root
        ? t('diskOf', { used: fmt.bytes(root.used), total: fmt.bytes(root.total) })
        : undefined,
      list: stats.disks.map((d) => ({
        key: d.mount,
        label: <Machine>{d.mount}</Machine>,
        hint: t('diskOf', { used: fmt.bytes(d.used), total: fmt.bytes(d.total) }),
        value: share(d.used, d.total),
        display: pct(share(d.used, d.total)),
      })),
    },
  ];
  return { cards, view, within: t('in_live') };
}
