import type { BarItem } from '../../../components/charts/BarList';
import type { SeriesColor } from '../../../components/charts/palette';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { Translate } from '../../../i18n/core';
import type { Sample, ServerStats } from '../../../lib/api/analytics';
import { changeOf, latestOf, type Change } from '../../../lib/charts/change';
import { sumValues, type Value } from '../../../lib/charts/series';
import type { AxisKind } from '../../../lib/charts/ticks';
import type { Format } from '../../../lib/format/format';
import { useFormat } from '../../../lib/preferences/region';
import { analyticsMessages, type AnalyticsKey } from '../../analytics/messages';
import { rangeSpan } from '../../analytics/range';
import { coreColor } from '../../analytics/server/charts';
import { cpuLines, serverView } from '../../analytics/server-view';
import { overviewMessages, type OverviewKey } from '../messages';

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

/**
 * CPU, memory, network and the disks of this server, live. A card's line
 * is drawn from its detail's series (the first one, or their sum), so the
 * card and its detail always span the same times.
 */
export function serverCards(
  stats: ServerStats,
  now: Sample,
  t: Translate<AnalyticsKey>,
  tOverview: Translate<OverviewKey>,
  fmt: Format,
) {
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
  // Every core together comes first: saved minutes have only that line.
  const cpuSeries = cpuLines(view).map((l) => {
    if (l.line === 'all') return named(t('seriesAllCores'), 1, l.values, pct);
    if (l.line === 'busiest') return named(t('seriesBusiest'), 2, l.values, pct);
    const i = l.core ?? 0;
    return named(t('seriesCore', { n: i + 1 }), coreColor(i), l.values, pct);
  });
  const memSeries = [
    named(t('seriesUsed'), 1, view.memUsed, fmt.bytes),
    named(t('seriesCache'), 2, view.memCache, fmt.bytes),
    ...(now.mem.swapTotal > 0 ? [named(t('seriesSwap'), 3, view.swap, fmt.bytes)] : []),
  ];
  const netSeries = [
    named(t('seriesIn'), 1, view.netRx, fmt.rate),
    named(t('seriesOut'), 2, view.netTx, fmt.rate),
  ];
  const lead = (series: readonly DetailSeries[]) => series[0]?.values ?? [];
  const mem = fmt.bytesParts(now.mem.used);
  const netSum = sumValues(netSeries.map((n) => n.values));
  const net = fmt.bytesParts(now.netRx + now.netTx);
  const root = stats.disks[0];
  const share = (used: number, total: number) => used / (total || 1);
  const cards: ServerCard[] = [
    {
      id: 'cpu',
      title: t('cpu'),
      value: pct(now.cpu),
      sub: t('cores', { count: stats.facts.cores }),
      change: changeOf(lead(cpuSeries), { points: true, risingIsBad: true }),
      chart: { values: lead(cpuSeries), max: 1 },
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
      change: changeOf(lead(memSeries), { risingIsBad: true }),
      chart: { values: lead(memSeries), max: now.mem.total },
      detail: {
        kind: 'bytes',
        format: fmt.bytes,
        more: 0,
        series: memSeries,
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
        series: netSeries,
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
/** The overview's server cards, once the server has a sample. */
export function useServerCards(stats: ServerStats | undefined) {
  const t = useT(analyticsMessages);
  const tOverview = useT(overviewMessages);
  const fmt = useFormat();
  const now = stats?.now;
  return stats && now ? serverCards(stats, now, t, tOverview, fmt) : undefined;
}
