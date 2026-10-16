import { Button } from '@cloudflare/kumo/components/button';
import { useEffect } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { PageHeader } from '../../components/layout/PageHeader';
import { useLocale, useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { ApiError, offline } from '../../lib/api/client';
import { formatLongDate, formatTime } from '../../lib/format/date';
import { ActivitySection } from './ActivitySection';
import { overviewMessages } from './messages';
import { OverviewSkeleton } from './OverviewSkeleton';
import { PanelsSection } from './PanelsSection';
import { SafeModeSection, SecuritySection } from './StatusSections';
import { useNow } from './use-now';
import { useOverview } from './use-overview';

export function OverviewPage() {
  const t = useT(overviewMessages);
  const ts = useT(sharedMessages);
  const { locale } = useLocale();
  const now = useNow();
  const overview = useOverview();

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
      <div className="flex flex-col items-start gap-3">
        <InlineAlert>
          {t('loadFailed')} {isOffline ? ts('errorOffline') : ts('errorServer')}
        </InlineAlert>
        <Button className="h-11 sm:h-9 pointer-coarse:h-11" onClick={() => void overview.refetch()}>
          {ts('retry')}
        </Button>
      </div>
    );
  } else {
    const data = overview.data;
    body = (
      <div className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="lg:row-span-2">
          <PanelsSection />
        </div>
        <SafeModeSection on={data.safeMode} />
        <SecuritySection security={data.security} now={now} />
        <ActivitySection entries={data.activity} now={now} className="lg:col-span-2" />
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-[70rem] flex-col gap-6 px-4 py-6 sm:px-6 sm:py-8">
      <PageHeader title={t('title')} description={today} />
      {body}
    </div>
  );
}
