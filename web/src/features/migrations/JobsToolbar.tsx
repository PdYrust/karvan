import { RefreshButton } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { Select } from '@cloudflare/kumo/components/select';
import { useId, type ReactNode } from 'react';
import { SearchField } from '../../components/data/SearchField';
import { useT } from '../../i18n/locale';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelKind, type Panel } from '../../lib/api/panels';
import { useCurrentPanel } from '../panels/current-panel';
import { usePanels } from '../panels/use-panels';
import type { StateFilter } from './job-place';
import { listMessages, type ListKey } from './list-messages';

/** A panel's logo and name, as every panel name is shown. */
function PanelOption({ panel }: { panel: Panel }) {
  const kind = panelKind(panel);
  return (
    <span className="flex min-w-0 items-center gap-2">
      {kind ? <PanelMark kind={kind} /> : null}
      <span className="truncate">{panel.name}</span>
    </span>
  );
}

/**
 * Search, the state filter, the panel filter and refresh, across the top
 * of a jobs list. The panel filter is the switcher's current panel, so
 * both always agree.
 */
export function JobsToolbar({
  search,
  onSearch,
  searchLabel,
  states,
  state,
  onState,
  refreshing,
  onRefresh,
  children,
}: {
  search: string;
  onSearch: (q: string) => void;
  searchLabel: string;
  states: readonly StateFilter[];
  state: StateFilter;
  onState: (s: StateFilter) => void;
  refreshing: boolean;
  onRefresh: () => void;
  /** Actions after refresh, such as starting a migration. */
  children?: ReactNode;
}) {
  const t = useT(listMessages);
  const id = useId();
  const panels = (usePanels().data ?? []).filter((p) => p.id > 0);
  const { panel, select } = useCurrentPanel();
  return (
    <div className="flex flex-wrap items-center gap-2">
      <SearchField
        label={searchLabel}
        value={search}
        onChange={onSearch}
        // On phones an action (+) shares the search row; filters and refresh go under it.
        className={cn(
          'min-w-0 flex-1 max-sm:order-first sm:min-w-56 sm:basis-0',
          children ? 'basis-[calc(100%-4rem)]' : 'basis-full',
        )}
      />
      <span id={`${id}-state`} className="sr-only">
        {t('stateLabel')}
      </span>
      <Select
        aria-labelledby={`${id}-state`}
        className="min-w-0 flex-1 sm:w-44 sm:flex-none pointer-coarse:h-11"
        value={state}
        items={Object.fromEntries(states.map((s) => [s, t(`filter_${s}` as ListKey)]))}
        onValueChange={(s) => s && onState(s as StateFilter)}
      />
      <span id={`${id}-panel`} className="sr-only">
        {t('panelLabel')}
      </span>
      <Select
        aria-labelledby={`${id}-panel`}
        className="min-w-0 flex-1 sm:w-56 sm:flex-none pointer-coarse:h-11"
        value={panel ? String(panel.id) : 'all'}
        renderValue={(v: string) => {
          const p = panels.find((x) => String(x.id) === v);
          return p ? <PanelOption panel={p} /> : t('allPanels');
        }}
        onValueChange={(v) => select(!v || v === 'all' ? null : Number(v))}
      >
        <Select.Option value="all">{t('allPanels')}</Select.Option>
        {panels.map((p) => (
          <Select.Option key={p.id} value={String(p.id)}>
            <PanelOption panel={p} />
          </Select.Option>
        ))}
      </Select>
      <RefreshButton
        variant="secondary"
        aria-label={t('refresh')}
        title={t('refresh')}
        loading={refreshing}
        onClick={onRefresh}
        className="pointer-coarse:size-11"
      />
      {children ? <div className="flex max-sm:-order-1">{children}</div> : null}
    </div>
  );
}
