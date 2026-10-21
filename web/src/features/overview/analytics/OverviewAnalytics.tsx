import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { useId } from 'react';
import { Link, useLocation } from 'react-router';
import { useT } from '../../../i18n/locale';
import type { RangeKey } from '../../../lib/api/analytics';
import { panelKind } from '../../../lib/api/panels';
import { RangeControl } from '../../analytics/RangeControl';
import { useRange } from '../../analytics/range';
import { LoadFailed } from '../../analytics/states';
import { usePanelTabs, useRefreshAnalytics } from '../../analytics/use-analytics';
import { usePanels } from '../../panels/use-panels';
import { overviewMessages } from '../messages';
import { DatabaseBlock } from './DatabaseBlock';
import { ServerBlock } from './ServerBlock';

/** The dashboard home offers fewer ranges than the analytics page. */
const homeRanges: readonly RangeKey[] = ['live', '24h', '7d', '30d'];

/** This server and every panel database, live or over a range, like the dashboard home's analytics. */
export function OverviewAnalytics() {
  const t = useT(overviewMessages);
  const id = useId();
  const [range, setRange] = useRange();
  const { refresh, fetching } = useRefreshAnalytics();
  const { search } = useLocation();
  const tabs = usePanelTabs();
  // A panel not read yet has no kind in its reading; the panel list knows it.
  const known = usePanels().data ?? [];
  return (
    <section aria-labelledby={id} className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id={id} className="text-heading font-semibold text-kumo-strong">
          {t('analyticsTitle')}
        </h2>
        <div className="flex flex-wrap items-center gap-2">
          <Link
            to={`/analytics${search}`}
            className={cn(buttonVariants({ variant: 'ghost' }), 'pointer-coarse:h-11')}
          >
            {t('allAnalytics')}
          </Link>
          <RangeControl
            range={range}
            onRange={setRange}
            onRefresh={refresh}
            refreshing={fetching}
            ranges={homeRanges}
          />
        </div>
      </div>
      <ServerBlock range={range} search={search} />
      {tabs.isError ? <LoadFailed error={tabs.error} onRetry={() => void tabs.refetch()} /> : null}
      {(tabs.data ?? []).map((p) => {
        const listed = known.find((k) => k.id === p.panelId);
        return (
          <DatabaseBlock
            key={p.panelId}
            panelId={p.panelId}
            name={p.name}
            kind={p.kind ?? (listed ? panelKind(listed) : undefined)}
            range={range}
            search={search}
          />
        );
      })}
    </section>
  );
}
