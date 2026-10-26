import { HourglassIcon } from '@phosphor-icons/react';
import { useT } from '../../../i18n/locale';
import { analyticsMessages } from '../messages';
import { rangeSpan, useRange } from '../range';
import { serverView } from '../server-view';
import { LoadFailed, Notice, TabSkeleton } from '../states';
import { useServerStats } from '../use-analytics';
import { DisksCard } from './DisksCard';
import { FactsCard } from './FactsCard';
import { InterfacesCard } from './InterfacesCard';
import { ServerCharts } from './ServerCharts';
import { ServerTiles } from './ServerTiles';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** This server: tiles, six charts, its disks and its facts. */
export function ServerTab() {
  const t = useT(analyticsMessages);
  const [range] = useRange();
  const q = useServerStats(range);

  const load = useFirstLoad(q.data);
  if (!load.data) {
    return q.isError ? (
      <LoadFailed error={q.error} onRetry={() => void q.refetch()} />
    ) : (
      <TabSkeleton label={t('tabServer')} />
    );
  }
  const stats = load.data;
  if (!stats.now) {
    return (
      <Notice icon={<HourglassIcon size={40} weight="light" aria-hidden="true" />}>
        {t('serverWaiting')}
      </Notice>
    );
  }
  const view = serverView(stats, rangeSpan[range]);
  return (
    <Reveal on={load.reveal}>
      <ServerTiles stats={stats} now={stats.now} view={view} range={range} />
      <ServerCharts now={stats.now} view={view} range={range} />
      <div className="grid items-start gap-4 lg:grid-cols-2">
        <div className="flex flex-col gap-4">
          {stats.disks.length > 0 ? <DisksCard disks={stats.disks} /> : null}
          {stats.now.ifaces.length > 0 ? <InterfacesCard ifaces={stats.now.ifaces} /> : null}
        </div>
        <FactsCard stats={stats} now={stats.now} />
      </div>
    </Reveal>
  );
}
