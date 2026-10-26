import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { useId } from 'react';
import { Link } from 'react-router';
import { useT } from '../../../i18n/locale';
import { panelKind } from '../../../lib/api/panels';
import { LoadFailed } from '../../analytics/states';
import { usePanelTabs } from '../../analytics/use-analytics';
import { usePanels } from '../../panels/use-panels';
import { overviewMessages } from '../messages';
import { DatabaseBlock } from './DatabaseBlock';
import { ServerBlock } from './ServerBlock';

/** This server and every panel database, always live, like the dashboard home's analytics. */
export function OverviewAnalytics() {
  const t = useT(overviewMessages);
  const id = useId();
  const tabs = usePanelTabs();
  // A panel not read yet has no kind in its reading; the panel list knows it.
  const known = usePanels().data ?? [];
  return (
    <section aria-labelledby={id} className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id={id} className="text-heading font-semibold text-kumo-strong">
          {t('analyticsTitle')}
        </h2>
        <Link
          to="/analytics"
          className={cn(buttonVariants({ variant: 'ghost' }), 'pointer-coarse:h-11')}
        >
          {t('allAnalytics')}
        </Link>
      </div>
      <ServerBlock />
      {tabs.isError ? <LoadFailed error={tabs.error} onRetry={() => void tabs.refetch()} /> : null}
      {(tabs.data ?? []).map((p) => {
        const listed = known.find((k) => k.id === p.panelId);
        return (
          <DatabaseBlock
            key={p.panelId}
            panelId={p.panelId}
            name={p.name}
            kind={p.kind ?? (listed ? panelKind(listed) : undefined)}
          />
        );
      })}
    </section>
  );
}
