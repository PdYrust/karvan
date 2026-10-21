import { DataTable, type Column } from '../../components/data/DataTable';
import { useT } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { useCurrentPanel } from './current-panel';
import { panelsMessages } from './messages';
import { ControlState, PanelDatabase, PanelName, PanelStatus, PanelVersion } from './PanelCells';
import { PanelMenu } from './PanelMenu';

/** On a phone a panel is a card: name first, then version and database, status at the foot. */
function PanelCard({ panel, current }: { panel: Panel; current: boolean }) {
  const t = useT(panelsMessages);
  return (
    <article className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
      <div className="flex items-center gap-2 ps-4 pe-2 pt-2">
        <span className="min-w-0 flex-1">
          <PanelName panel={panel} current={current} />
        </span>
        <PanelMenu panel={panel} />
      </div>
      <div className="flex flex-col gap-1 px-4 pb-3 text-sm">
        <PanelVersion panel={panel} />
        <PanelDatabase panel={panel} />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-t border-kumo-hairline px-4 py-2.5 text-sm">
        <PanelStatus panel={panel} />
        <span className="flex items-center gap-2">
          <span className="text-kumo-subtle">{t('colControl')}</span>
          <ControlState control={panel.control} />
        </span>
      </div>
    </article>
  );
}

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
      card={(p) => <PanelCard panel={p} current={isCurrent(p)} />}
    />
  );
}
