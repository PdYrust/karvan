import { FactList, type Fact } from '../../../components/layout/FactList';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { Sample, ServerStats } from '../../../lib/api/analytics';
import { useFormat } from '../../../lib/preferences/region';
import { useOverview } from '../../overview/use-overview';
import { ChartCard } from '../ChartCard';
import { analyticsMessages } from '../messages';

/** What the server is and how long it has run, with Karvan's own share; the overview lists them too. */
export function useServerFacts(stats: ServerStats, now: Sample): Fact[] {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  const version = useOverview().data?.version;
  const f = stats.facts;
  const karvanUp = (Date.parse(now.at) - Date.parse(f.startedAt)) / 1000;
  return [
    { label: t('factHost'), value: <Machine>{f.hostname}</Machine> },
    ...(f.os ? [{ label: t('factOs'), value: <bdi>{f.os}</bdi> }] : []),
    { label: t('factKernel'), value: <Machine>{`${f.kernel} ${f.arch}`.trim()}</Machine> },
    {
      label: t('factCpu'),
      value: (
        <span className="flex flex-col items-end">
          {f.cpuModel ? <bdi>{f.cpuModel}</bdi> : null}
          <span className="text-sm text-kumo-subtle">{t('cores', { count: f.cores })}</span>
        </span>
      ),
    },
    { label: t('factUptime'), value: fmt.duration(now.uptime) },
    {
      label: t('factProcs'),
      value: t('procsValue', { total: now.procs, running: now.procsRunning }),
    },
    ...(version ? [{ label: t('factKarvanVersion'), value: <Machine>{version}</Machine> }] : []),
    ...(Number.isFinite(karvanUp)
      ? [{ label: t('factKarvanUp'), value: fmt.duration(karvanUp) }]
      : []),
    { label: t('factKarvanMemory'), value: fmt.bytes(f.karvanRss) },
  ];
}
export function FactsCard({ stats, now }: { stats: ServerStats; now: Sample }) {
  const t = useT(analyticsMessages);
  const facts = useServerFacts(stats, now);
  return (
    <ChartCard title={t('factsTitle')}>
      <FactList facts={facts} />
    </ChartCard>
  );
}
