import { lazy, Suspense } from 'react';
import { Bone } from '../../components/feedback/PageSkeleton';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
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
      <div className="grid gap-4 lg:grid-cols-2">
        {[0, 1].map((i) => (
          <Bone key={i} className="h-80 w-full rounded-lg" />
        ))}
      </div>
    </div>
  );
}

/** Laid out like the dashboard's account home: search, three columns, a summary, then analytics. */
export function OverviewPage() {
  const t = useT(overviewMessages);
  usePageTitle(t('title'));
  const now = useNow();
  const panels = usePanels();
  const { scan, scanning } = useScanPanels();
  const [analyticsRef, nearAnalytics] = useNear<HTMLDivElement>();
  const noPanels = panels.data !== undefined && !panels.data.some((p) => p.id > 0);

  return (
    <div className="flex flex-col">
      {/* Like the dashboard home: a question, then the search; the title is for screen readers. */}
      <h1 className="sr-only">{t('title')}</h1>
      {/* Spaced like the dashboard home: 48 px (56 on desktop) under the top bar, 32 to the search, 32 to the lists. */}
      <div className="flex max-w-[1400px] flex-col gap-8 p-4 sm:p-8">
        <div className="flex flex-col gap-8 pt-8 sm:pt-6">
          <p className="text-center text-display font-semibold text-kumo-strong">{t('heading')}</p>
          <HomeSearch />
        </div>
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
