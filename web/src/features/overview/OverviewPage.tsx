import { lazy, Suspense } from 'react';
import { StatTileSkeleton } from '../../components/charts/StatTile';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { useFormat } from '../../lib/preferences/region';
import { NoPanels } from '../panels/NoPanels';
import { usePanels, useScanPanels } from '../panels/use-panels';
import { MigrationsColumn } from './columns/MigrationsColumn';
import { PanelsColumn } from './columns/PanelsColumn';
import { RecentsColumn } from './columns/RecentsColumn';
import { HomeSearch } from './search/HomeSearch';
import { overviewMessages } from './messages';
import { Summary } from './summary/Summary';
import { useNear } from './use-near';
import { useNow } from './use-now';

// The charts load after the rest of the page, so the columns and the summary come first,
// and only once the reader nears them (far below the first screen on a phone).
const OverviewAnalytics = lazy(() =>
  import('./analytics/OverviewAnalytics').then((m) => ({ default: m.OverviewAnalytics })),
);

function AnalyticsSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-4">
      <Bone className="h-6 w-28" />
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <StatTileSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

/** Laid out like the dashboard's account home: search, three columns, a summary, then analytics. */
export function OverviewPage() {
  const t = useT(overviewMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const now = useNow();
  const panels = usePanels();
  const { scan, scanning } = useScanPanels();
  const [analyticsRef, nearAnalytics] = useNear<HTMLDivElement>();
  const noPanels = panels.data !== undefined && !panels.data.some((p) => p.id > 0);

  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('today', { date: fmt.longDate(now), time: fmt.time(now), zone: fmt.zone })}
      />
      <div className="flex max-w-[1400px] flex-col gap-10 p-4 sm:p-8">
        <HomeSearch />
        {noPanels ? (
          <NoPanels onScan={scan} scanning={scanning} />
        ) : (
          <>
            <div className="grid gap-8 lg:grid-cols-3">
              <PanelsColumn panels={panels} />
              <MigrationsColumn now={now} />
              <RecentsColumn panels={panels} />
            </div>
            <Summary />
          </>
        )}
        <div ref={analyticsRef}>
          {nearAnalytics ? (
            <Suspense fallback={<AnalyticsSkeleton label={t('analyticsTitle')} />}>
              <OverviewAnalytics />
            </Suspense>
          ) : (
            <AnalyticsSkeleton label={t('analyticsTitle')} />
          )}
        </div>
      </div>
    </div>
  );
}
