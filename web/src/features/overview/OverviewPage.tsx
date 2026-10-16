import { Button } from '@cloudflare/kumo/components/button';
import { useEffect } from 'react';
import { useSession } from '../session/use-session';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { PageHeader } from '../../components/layout/PageHeader';
import { useLocale, useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { ApiError, offline } from '../../lib/api/client';
import { formatLongDate, formatTime } from '../../lib/format/date';
import { ActivityTable } from './ActivityTable';
import { overviewMessages } from './messages';
import { OverviewAside } from './OverviewAside';
import { OverviewSkeleton } from './OverviewSkeleton';
import { PanelsSection } from './PanelsSection';
import { useNow } from './use-now';
import { useOverview } from './use-overview';

export function OverviewPage() {
  const t = useT(overviewMessages);
  const ts = useT(sharedMessages);
  const { locale } = useLocale();
  const now = useNow();
  const overview = useOverview();
  const username = useSession().data?.operator?.username ?? '';

  useEffect(() => {
    document.title = t('pageTitle');
  }, [t]);

  const today = t('today', { date: formatLongDate(now, locale), time: formatTime(now, locale) });

  let body;
  if (overview.isPending) {
    body = <OverviewSkeleton label={t('loading')} />;
  } else if (overview.isError) {
    const isOffline = overview.error instanceof ApiError && overview.error.code === offline;
    body = (
      <div className="flex flex-col items-start gap-3 p-4 sm:p-8">
        <InlineAlert>
          {t('loadFailed')} {isOffline ? ts('errorOffline') : ts('errorServer')}
        </InlineAlert>
        <Button className="h-9 pointer-coarse:h-11" onClick={() => void overview.refetch()}>
          {ts('retry')}
        </Button>
      </div>
    );
  } else {
    body = (
      <div className="grid lg:grid-cols-[minmax(0,1fr)_22rem]">
        <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8">
          <PanelsSection />
          <ActivityTable entries={overview.data.activity} now={now} />
        </div>
        <aside className="border-t border-kumo-hairline p-4 sm:p-8 lg:border-t-0 lg:border-s">
          <OverviewAside data={overview.data} username={username} now={now} />
        </aside>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={today} />
      {body}
    </div>
  );
}
