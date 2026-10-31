import { DataTable, type Column } from '../../components/data/DataTable';
import { useT } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { useCurrentPanel } from './current-panel';
import { panelsMessages } from './messages';
import { PanelCard } from './PanelCard';
import { ControlState, PanelDatabase, PanelName, PanelVersion } from './PanelCells';
import { PanelMenu } from './PanelMenu';
import { PanelStatus } from './PanelStatus';

/** Every registered panel with its version, database, status and panel control. */
export function PanelsTable({ panels }: { panels: readonly Panel[] }) {
  const t = useT(panelsMessages);
  const { panel: current } = useCurrentPanel();
  const isCurrent = (p: Panel) => current?.id === p.id;
  const columns: Column<Panel>[] = [
    {
      key: 'name',
      header: t('colPanel'),
      cell: (p) => <PanelName panel={p} current={isCurrent(p)} />,
      className: 'max-w-72',
    },
    { key: 'version', header: t('colVersion'), cell: (p) => <PanelVersion panel={p} /> },
    {
      key: 'database',
      header: t('colDatabase'),
      cell: (p) => <PanelDatabase panel={p} />,
      className: 'max-w-72',
    },
    { key: 'status', header: t('colStatus'), cell: (p) => <PanelStatus panel={p} /> },
    {
      key: 'control',
      header: t('colControl'),
      cell: (p) => <ControlState control={p.control} />,
    },
    {
      key: 'menu',
      header: t('colActions'),
      hideHeader: true,
      cell: (p) => <PanelMenu panel={p} />,
      className: 'w-14 py-1.5 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('table')}
      rows={panels}
      rowKey={(p) => p.id}
      columns={columns}
      card={(p) => <PanelCard panel={p} current={isCurrent(p)} menu={<PanelMenu panel={p} />} />}
    />
  );
}
