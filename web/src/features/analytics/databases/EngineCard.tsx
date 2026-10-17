import { Bar } from '../../../components/charts/Bar';
import { TimeSeries } from '../../../components/charts/TimeSeries';
import type { Translate } from '../../../i18n/core';
import { useT } from '../../../i18n/locale';
import type { EngineStats, PanelStats, RangeKey } from '../../../lib/api/analytics';
import type { Format } from '../../../lib/format/format';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { FactList, Machine, type Fact } from '../FactList';
import { analyticsMessages, type AnalyticsKey } from '../messages';
import { databaseRange } from '../range';
import { engineLabel, isServerEngine } from './engine';

/** MySQL, MariaDB and PostgreSQL: connections, queries, cache and uptime. */
function serverFacts(e: EngineStats, p: PanelStats, t: Translate<AnalyticsKey>, fmt: Format) {
  const facts: Fact[] = [
    {
      label: t('engineConnections'),
      value: (
        <span className="flex flex-col items-end gap-1.5">
          {t('connectionsOf', { count: e.connections, max: e.maxConnections })}
          <Bar share={e.connections / (e.maxConnections || 1)} className="w-24" />
        </span>
      ),
    },
    { label: t('engineQps'), value: fmt.number(p.qps, 1) },
  ];
  if (p.cacheHit !== null) {
    facts.push({ label: t('engineCache'), value: fmt.percent(p.cacheHit, 1) });
  }
  if (e.uptime > 0) facts.push({ label: t('engineUptime'), value: fmt.duration(e.uptime) });
  return facts;
}

/** SQLite: its pages, free pages and write-ahead log. */
function fileFacts(e: EngineStats, t: Translate<AnalyticsKey>, fmt: Format) {
  const facts: Fact[] = [];
  if (e.pageSize) facts.push({ label: t('enginePageSize'), value: fmt.bytes(e.pageSize) });
  if (e.pages) facts.push({ label: t('enginePages'), value: fmt.number(e.pages) });
  if (e.freePages) facts.push({ label: t('engineFreePages'), value: fmt.number(e.freePages) });
  if (e.walSize) facts.push({ label: t('engineWal'), value: fmt.bytes(e.walSize) });
  if (e.journalMode) {
    facts.push({ label: t('engineJournal'), value: <Machine>{e.journalMode}</Machine> });
  }
  return facts;
}

/** The engine, its size and how it changed over the range, and its own counters. */
export function EngineCard({
  panel,
  step,
  domain,
  range,
}: {
  panel: PanelStats;
  step: number;
  domain: [number, number];
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const e = panel.engine;
  if (!e) return null;
  const within = t(`in_${databaseRange(range)}`);
  const first = panel.points[0]?.size;
  const change = first === undefined ? 0 : e.size - first;
  const signed =
    change === 0 ? t('noChange') : `${change > 0 ? '+' : '-'}${fmt.bytes(Math.abs(change))}`;
  const facts: Fact[] = [
    {
      label: t('engineType'),
      value: (
        <span className="flex flex-col items-end">
          <Machine>{engineLabel(e)}</Machine>
          {e.timescale ? <Machine>{`TimescaleDB ${e.timescale}`}</Machine> : null}
        </span>
      ),
    },
    { label: t('engineSize'), value: fmt.bytes(e.size) },
    { label: t('engineGrowth', { range: within }), value: signed },
    ...(isServerEngine(e) ? serverFacts(e, panel, t, fmt) : fileFacts(e, t, fmt)),
  ];
  const times = panel.points.map((p) => p.t * 1000);
  const charts = [
    {
      title: t('engineSize'),
      kind: 'bytes' as const,
      values: panel.points.map((p) => p.size),
      format: fmt.bytes,
    },
    ...(isServerEngine(e)
      ? [
          {
            title: t('engineQps'),
            kind: 'count' as const,
            values: panel.points.map((p) => p.qps),
            format: (n: number) => fmt.number(n, 1),
          },
        ]
      : []),
  ];
  return (
    <ChartCard title={t('engineTitle')}>
      <FactList facts={facts} />
      {charts.map((c) => (
        <div key={c.title} className="flex flex-col gap-2">
          <h3 className="text-sm text-kumo-subtle">{c.title}</h3>
          <TimeSeries
            label={t('chartLabel', { chart: c.title, range: within })}
            times={times}
            series={[{ name: c.title, color: 1, values: c.values }]}
            domain={domain}
            step={step}
            kind={c.kind}
            format={c.format}
            empty={t('noData', { range: within })}
            height={140}
          />
        </div>
      ))}
    </ChartCard>
  );
}
