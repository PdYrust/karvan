import type { SeriesColor } from '../../../components/charts/palette';
import type { ChartSeries } from '../../../components/charts/TimeSeries';
import type { Translate } from '../../../i18n/core';
import type { Sample } from '../../../lib/api/analytics';
import { maxValue, sumValues, type Value } from '../../../lib/charts/series';
import type { AxisKind } from '../../../lib/charts/ticks';
import type { Format } from '../../../lib/format/format';
import type { AnalyticsKey } from '../messages';
import { cpuLines, type ServerView } from '../server-view';

/** A series with the number its legend shows. */
export type LegendSeries = ChartSeries & { value?: string };

export interface ChartSpec {
  key: 'cpu' | 'load' | 'memory' | 'network' | 'disk' | 'tcp';
  title: string;
  kind: AxisKind;
  format: (v: number) => string;
  series: LegendSeries[];
  /** A chart that can be split: what it is split by, and the one line it shows when it is not. */
  groups?: { by: 'groupCore' | 'groupDirection'; total: LegendSeries[] };
}

/** A core's line color: the colors after the one every core together takes. */
export const coreColor = (core: number) => (core + 2) as SeriesColor;
export function mean(values: readonly Value[]): number | undefined {
  let sum = 0;
  let n = 0;
  for (const v of values) {
    if (v !== null && v !== undefined) {
      sum += v;
      n++;
    }
  }
  return n > 0 ? sum / n : undefined;
}

/** Bytes moved over a range: each point's average rate times the seconds it stands for. */
export function total(rates: readonly Value[], covered: readonly number[]): number {
  return rates.reduce<number>((sum, r, i) => sum + (r ?? 0) * (covered[i] ?? 0), 0);
}

/**
 * The six server charts. Live legends show the latest sample. Over a range
 * they show the average, the highest peak, and for network and disk the
 * total moved, which says more than an average rate.
 */
export function serverCharts(
  v: ServerView,
  now: Sample,
  t: Translate<AnalyticsKey>,
  fmt: Format,
): ChartSpec[] {
  const show = (f: (n: number) => string) => (n: number | undefined) =>
    n === undefined ? undefined : f(n);
  const pct = (n: number) => fmt.percent(n, 1);
  const count = (n: number) => fmt.number(n);
  const avg = (values: readonly Value[], f: (n: number) => string) => show(f)(mean(values));
  const moved = (rates: readonly Value[]) => fmt.bytes(total(rates, v.covered));
  const s = (
    name: string,
    color: SeriesColor,
    values: readonly Value[],
    value: string | undefined,
    dashed = false,
  ): LegendSeries => ({ name, color, values, value, dashed });

  const cpu = v.live
    ? cpuLines(v).map((l) => {
        if (l.line === 'all') return s(t('seriesAllCores'), 1, l.values, pct(now.cpu));
        if (l.line === 'busiest') {
          return s(t('seriesBusiest'), 2, l.values, pct(Math.max(0, ...now.cores)), true);
        }
        const i = l.core ?? 0;
        return s(t('seriesCore', { n: i + 1 }), coreColor(i), l.values, pct(now.cores[i] ?? 0));
      })
    : [
        s(t('seriesAverage'), 1, v.cpu, avg(v.cpu, pct)),
        s(t('seriesPeak'), 1, v.cpuPeak, pct(maxValue([v.cpuPeak])), true),
      ];
  const load2 = (n: number) => fmt.number(n, 2);
  const load = v.live
    ? [
        s(t('seriesLoad1'), 1, v.load1, load2(now.load[0])),
        s(t('seriesLoad5'), 2, v.load5, load2(now.load[1])),
        s(t('seriesLoad15'), 3, v.load15, load2(now.load[2])),
      ]
    : [s(t('seriesLoad1'), 1, v.load1, avg(v.load1, load2))];
  const memory = [
    s(t('seriesUsed'), 1, v.memUsed, v.live ? fmt.bytes(now.mem.used) : avg(v.memUsed, fmt.bytes)),
    s(
      t('seriesCache'),
      2,
      v.memCache,
      v.live ? fmt.bytes(now.mem.cache) : avg(v.memCache, fmt.bytes),
    ),
    s(t('seriesSwap'), 3, v.swap, v.live ? fmt.bytes(now.mem.swapUsed) : avg(v.swap, fmt.bytes)),
  ];
  const network = [
    s(t('seriesIn'), 1, v.netRx, v.live ? fmt.rate(now.netRx) : moved(v.netRx)),
    s(t('seriesOut'), 2, v.netTx, v.live ? fmt.rate(now.netTx) : moved(v.netTx)),
  ];
  const disk = [
    s(t('seriesRead'), 1, v.diskRead, v.live ? fmt.rate(now.diskRead) : moved(v.diskRead)),
    s(t('seriesWrite'), 2, v.diskWrite, v.live ? fmt.rate(now.diskWrite) : moved(v.diskWrite)),
  ];
  const tcp = v.live
    ? [
        s(t('seriesOpen'), 1, v.tcp, count(now.tcp)),
        s(t('seriesClosing'), 2, v.closing, count(now.tcpTimeWait)),
      ]
    : [
        s(
          t('seriesOpen'),
          1,
          v.tcp,
          avg(v.tcp, (n) => count(Math.round(n))),
        ),
      ];

  const sum = (a: readonly Value[], b: readonly Value[], live: number) => {
    const values = sumValues([a, b]);
    return [s(t('seriesTotal'), 1, values, v.live ? fmt.rate(live) : moved(values))];
  };
  const byCore = v.live && cpu.length > 1;
  return [
    {
      key: 'cpu',
      title: t('cpu'),
      kind: 'percent',
      format: pct,
      series: cpu,
      groups: byCore ? { by: 'groupCore', total: cpu.slice(0, 1) } : undefined,
    },
    { key: 'load', title: t('chartLoad'), kind: 'number', format: load2, series: load },
    { key: 'memory', title: t('memory'), kind: 'bytes', format: fmt.bytes, series: memory },
    {
      key: 'network',
      title: t('network'),
      kind: 'rate',
      format: fmt.rate,
      series: network,
      groups: { by: 'groupDirection', total: sum(v.netRx, v.netTx, now.netRx + now.netTx) },
    },
    {
      key: 'disk',
      title: t('chartDisk'),
      kind: 'rate',
      format: fmt.rate,
      series: disk,
      groups: {
        by: 'groupDirection',
        total: sum(v.diskRead, v.diskWrite, now.diskRead + now.diskWrite),
      },
    },
    { key: 'tcp', title: t('chartTcp'), kind: 'count', format: count, series: tcp },
  ];
}
