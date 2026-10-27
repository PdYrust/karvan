import { RefreshButton } from '../../components/actions/RefreshButton';
import { useState } from 'react';
import { SearchField } from '../../components/data/SearchField';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { panelsMessages } from './messages';
import { NoPanels } from './NoPanels';
import { PanelsTable } from './PanelsTable';
import { ScanFinds } from './ScanFinds';
import { matchesPanel } from './search';
import { usePanels, useScanFinds, useScanPanels } from './use-panels';
import { Reveal } from '../../components/feedback/Reveal';
import { useFirstLoad } from '../../lib/motion/use-first-load';

/** The toolbar and a few rows, the shape the list takes. */
function ListSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-4">
      <Bone className="h-8 w-full" />
      <Bone className="h-44 w-full rounded-lg" />
    </div>
  );
}

/** Panels on this server, like the dashboard's list of domains. */
export function PanelsPage() {
  const t = useT(panelsMessages);
  usePageTitle(t('title'));
  const panels = usePanels();
  const { scan, scanning } = useScanPanels();
  const finds = useScanFinds();
  const [query, setQuery] = useState('');

  const load = useFirstLoad(panels.data);
  const all = load.data ?? [];
  const shown = all.filter((p) => matchesPanel(p, query));
  let body;
  if (!load.data) {
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
            aria-label={t('scanAndRefresh')}
            title={t('scanAndRefresh')}
            loading={scanning}
            onClick={scan}
            className="pointer-coarse:size-9"
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
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex max-w-[1400px] flex-col gap-6 p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
        {finds.length > 0 ? <ScanFinds finds={finds} /> : null}
      </div>
    </div>
  );
}
