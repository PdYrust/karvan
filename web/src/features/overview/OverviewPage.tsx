import { Button } from '@cloudflare/kumo/components/button';
import { PlusIcon } from '@phosphor-icons/react';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { useFormat } from '../../lib/preferences/region';
import { NoPanels } from '../panels/NoPanels';
import { usePanels, useScanPanels } from '../panels/use-panels';
import { MigrationsColumn } from './columns/MigrationsColumn';
import { PanelsColumn } from './columns/PanelsColumn';
import { RecentsColumn } from './columns/RecentsColumn';
import { HomeSearch } from './HomeSearch';
import { overviewMessages } from './messages';
import { Summary } from './summary/Summary';
import { useNow } from './use-now';

/** Inert with its "Soon" tag until the wizard is built. */
function NewMigration() {
  const t = useT(overviewMessages);
  const ts = useT(sharedMessages);
  return (
    <Button disabled icon={<PlusIcon size={16} />} className="pointer-coarse:h-11">
      {t('newMigration')}
      <span className="rounded-md px-1.5 text-tiny font-normal ring ring-kumo-line">
        {ts('soon')}
      </span>
    </Button>
  );
}

/** Laid out like the dashboard's account home: search, three columns and a summary. */
export function OverviewPage() {
  const t = useT(overviewMessages);
  usePageTitle(t('title'));
  const fmt = useFormat();
  const now = useNow();
  const panels = usePanels();
  const { scan, scanning } = useScanPanels();
  const noPanels = panels.data !== undefined && !panels.data.some((p) => p.id > 0);

  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('today', { date: fmt.longDate(now), time: fmt.time(now), zone: fmt.zone })}
        actions={<NewMigration />}
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
      </div>
    </div>
  );
}
