import { Button, RefreshButton } from '@cloudflare/kumo/components/button';
import { ArrowsClockwiseIcon, HardDrivesIcon } from '@phosphor-icons/react';
import { useState } from 'react';
import { SearchField } from '../../components/data/SearchField';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { panelsMessages } from './messages';
import { PanelsTable } from './PanelsTable';
import { ScanFinds } from './ScanFinds';
import { matchesPanel } from './search';
import { usePanels, useRefreshPanels, useScanFinds, useScanPanels } from './use-panels';

/** The toolbar and a few rows, the shape the list takes. */
function ListSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-4">
      <Bone className="h-9 w-full" />
      <Bone className="h-44 w-full rounded-lg" />
    </div>
  );
}

/** No panel registered yet: the two panels Karvan works with, and the scan. */
function NoPanels({ onScan, scanning }: { onScan: () => void; scanning: boolean }) {
  const t = useT(panelsMessages);
  return (
    <EmptyState
      icon={<HardDrivesIcon size={44} weight="light" aria-hidden="true" />}
      title={t('emptyTitle')}
      action={
        <Button
          icon={<ArrowsClockwiseIcon size={16} />}
          loading={scanning}
          onClick={onScan}
          className="pointer-coarse:h-11"
        >
          {t('scan')}
        </Button>
      }
    >
      <p>
        <CodeText text={t('emptyBody')} />
      </p>
      <ul
        aria-label={t('supported')}
        className="mt-4 flex items-center justify-center gap-5 text-sm text-kumo-default"
      >
        {(['marzban', 'pasarguard'] as const).map((k) => (
          <li key={k} className="flex items-center gap-2">
            <PanelMark kind={k} size={18} />
            {panelNames[k]}
          </li>
        ))}
      </ul>
    </EmptyState>
  );
}

/** Panels on this server, like the dashboard's list of domains. */
export function PanelsPage() {
  const t = useT(panelsMessages);
  usePageTitle(t('title'));
  const panels = usePanels();
  const { scan, scanning } = useScanPanels();
  const { refresh, refreshing } = useRefreshPanels();
  const finds = useScanFinds();
  const [query, setQuery] = useState('');

  const all = panels.data ?? [];
  const shown = all.filter((p) => matchesPanel(p, query));
  let body;
  if (!panels.data) {
    body = panels.isError ? (
      <ErrorState
        message={t('loadFailed')}
        error={panels.error}
        onRetry={() => void panels.refetch()}
      />
    ) : (
      <ListSkeleton label={t('title')} />
    );
  } else if (all.length === 0) {
    body = <NoPanels onScan={scan} scanning={scanning} />;
  } else {
    body = (
      <div className="flex flex-col gap-4">
        <div className="flex items-center gap-2">
          <SearchField label={t('search')} value={query} onChange={setQuery} className="flex-1" />
          <RefreshButton
            variant="secondary"
            aria-label={t('check')}
            title={t('check')}
            loading={refreshing}
            onClick={refresh}
            className="pointer-coarse:size-11"
          />
        </div>
        {shown.length > 0 ? (
          <PanelsTable panels={shown} />
        ) : (
          <p className="rounded-lg bg-kumo-base px-4 py-8 text-center text-kumo-subtle ring ring-kumo-hairline">
            {t('noMatch')}
          </p>
        )}
        <p className="text-sm text-kumo-subtle" aria-live="polite">
          {t('showing', { shown: shown.length, total: all.length })}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <Button
            variant="primary"
            icon={<ArrowsClockwiseIcon size={16} />}
            loading={scanning}
            onClick={scan}
            className="pointer-coarse:h-11"
          >
            {t('scan')}
          </Button>
        }
      />
      <div className="flex max-w-[1400px] flex-col gap-6 p-4 sm:p-8">
        {body}
        {finds.length > 0 ? <ScanFinds finds={finds} /> : null}
      </div>
    </div>
  );
}
