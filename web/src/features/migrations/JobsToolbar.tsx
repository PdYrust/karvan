import type { ReactNode } from 'react';
import { ToolbarRefresh } from '../../components/data/ToolbarRefresh';
import { ListToolbar, type ToolbarChoice } from '../../components/data/ListToolbar';
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
 * The list toolbar of a jobs list: search, the state filter, the panel filter and refresh.
 * The panel filter is the switcher's current panel, so both always agree.
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
  /** Actions after the toolbar, such as starting a migration. */
  children?: ReactNode;
}) {
  const t = useT(listMessages);
  const panels = (usePanels().data ?? []).filter((p) => p.id > 0);
  const { panel, select } = useCurrentPanel();
  const choices: ToolbarChoice[] = [
    {
      label: t('stateLabel'),
      value: state,
      all: 'all',
      options: states.map((s) => {
        const text = t(`filter_${s}` as ListKey);
        return { value: s, label: text, text };
      }),
      onChange: (s) => onState(s as StateFilter),
    },
    {
      label: t('panelLabel'),
      value: panel ? String(panel.id) : 'all',
      all: 'all',
      options: [
        { value: 'all', label: t('allPanels'), text: t('allPanels') },
        ...panels.map((p) => ({
          value: String(p.id),
          label: <PanelOption panel={p} />,
          text: p.name,
        })),
      ],
      onChange: (v) => select(v === 'all' ? null : Number(v)),
    },
  ];
  return (
    <ListToolbar
      search={{ label: searchLabel, value: search, onChange: onSearch }}
      choices={choices}
      refresh={<ToolbarRefresh label={t('refresh')} loading={refreshing} onClick={onRefresh} />}
    >
      {children}
    </ListToolbar>
  );
}
