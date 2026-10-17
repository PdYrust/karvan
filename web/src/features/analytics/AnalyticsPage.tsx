import { Suspense } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router';
import { PageHeader } from '../../components/layout/PageHeader';
import { RouteTabs, type RouteTab } from '../../components/layout/RouteTabs';
import { usePageTitle } from '../../components/layout/use-page-title';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { panelKind } from '../../lib/api/panels';
import { usePanels } from '../panels/use-panels';
import { analyticsMessages } from './messages';
import { RangeControl } from './RangeControl';
import { useRange } from './range';
import { TabSkeleton } from './states';
import { usePanelTabs, useRefreshAnalytics } from './use-analytics';

/** Analytics with a tab for this server and one per panel database, like the dashboard's analytics. */
export function AnalyticsPage() {
  const t = useT(analyticsMessages);
  usePageTitle(t('title'));
  const [range, setRange] = useRange();
  const { refresh, fetching } = useRefreshAnalytics();
  const { search } = useLocation();
  const known = usePanels().data ?? [];
  const panels = usePanelTabs().data ?? [];

  // Tabs keep the range, so switching between them compares the same time.
  const tabs: RouteTab[] = [
    { to: `/analytics/server${search}`, label: t('tabServer') },
    ...panels.map((p) => {
      // A panel not read yet has no kind in its reading; the panel list knows it.
      const listed = known.find((k) => k.id === p.panelId);
      const kind = p.kind ?? (listed ? panelKind(listed) : undefined);
      return {
        to: `/analytics/panels/${String(p.panelId)}${search}`,
        label: p.name,
        icon: kind ? <PanelMark kind={kind} /> : undefined,
      };
    }),
  ];

  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <RangeControl
            range={range}
            onRange={setRange}
            onRefresh={refresh}
            refreshing={fetching}
          />
        }
        tabs={<RouteTabs label={t('tabs')} tabs={tabs} />}
      />
      <div className="flex max-w-[1400px] flex-col gap-4 p-4 sm:p-8">
        {/* A tab's code loads under the header, so the header stays put. */}
        <Suspense fallback={<TabSkeleton label={t('title')} />}>
          <Outlet />
        </Suspense>
      </div>
    </div>
  );
}

/** /analytics opens the server tab, keeping the range. */
export function AnalyticsIndex() {
  const { search } = useLocation();
  return <Navigate to={`server${search}`} replace />;
}
