import { CpuIcon } from '@phosphor-icons/react';
import { Suspense } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router';
import type { RouteTab } from '../../components/layout/RouteTabs';
import { TabSwitcher } from '../../components/layout/TabSwitcher';
import { usePageTitle } from '../../components/layout/use-page-title';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useT } from '../../i18n/locale';
import { panelKind } from '../../lib/api/panels';
import { usePanels } from '../panels/use-panels';
import { analyticsMessages } from './messages';
import { RangeControl } from './RangeControl';
import { useRange } from './range';
import { TabSkeleton } from './states';
import { usePanelTabs } from './use-analytics';

/** Analytics with a tab for this server and one per panel database, like the dashboard's analytics. */
export function AnalyticsPage() {
  const t = useT(analyticsMessages);
  usePageTitle(t('title'));
  const [range, setRange] = useRange();
  const { search, pathname } = useLocation();
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
      {/* Like the dashboard's zone analytics: no title block, one sticky row of tabs and controls. */}
      <h1 className="sr-only">{t('title')}</h1>
      <div className="sticky top-[calc(3.5rem_+_env(safe-area-inset-top))] z-[5] flex items-center gap-3 border-b border-kumo-hairline bg-kumo-canvas px-4 py-2 transition-transform duration-200 ease-out sm:px-8 [[data-bar-hidden]_&]:-translate-y-14">
        <TabSwitcher
          label={t('tabs')}
          tabs={tabs}
          icon={<CpuIcon size={16} aria-hidden="true" className="text-kumo-subtle" />}
        />
        <RangeControl range={range} onRange={setRange} />
      </div>
      <div className="flex max-w-[1400px] flex-col gap-4 p-4 sm:p-8">
        {/* A tab's code loads under the header, so the header stays put. */}
        <Suspense fallback={<TabSkeleton label={t('title')} />}>
          {/* Keyed by tab, so a new tab's content fades in. */}
          <div key={pathname} className="k-reveal">
            <Outlet />
          </div>
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
