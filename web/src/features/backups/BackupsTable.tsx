import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { ArrowCounterClockwiseIcon, DotsThreeIcon, TrashIcon } from '@phosphor-icons/react';
import { DataTable, type Column } from '../../components/data/DataTable';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { panelKind } from '../../lib/api/panels';
import { backupDatabase } from '../../lib/api/safemode';
import { useFormat } from '../../lib/preferences/region';
import { JobName } from '../migrations/JobCells';
import { JobStateLabel } from '../migrations/JobLabels';
import { controlMessages, controlText } from '../panels/control';
import type { BackupRow } from './backup-rows';
import { backupsMessages } from './messages';

export interface BackupActions {
  onRestore: (r: BackupRow) => void;
  onDelete: (r: BackupRow) => void;
}

const jobPath = (r: BackupRow) => `/migrations/${encodeURIComponent(r.backup.jobId)}`;

/** Why Put back is off, in a few words under the menu item. */
function useBlockWords() {
  const t = useT(backupsMessages);
  const tc = useT(controlMessages);
  return (r: BackupRow): string | null => {
    if (r.restore === 'running') return t('running');
    if (r.restore === 'gone') return t('gone');
    if (r.restore === 'control' && r.panel?.control) return tc(controlText(r.panel.control).label);
    return null;
  };
}

/** The "..." menu of a backup: Put back and Delete, each off with why when it can't be done. */
function BackupMenu({ row, actions }: { row: BackupRow; actions: BackupActions }) {
  const t = useT(backupsMessages);
  const why = useBlockWords()(row);
  const label = t('moreFor', { id: row.backup.id });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} weight="bold" />}
            className="pointer-coarse:size-11"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-56">
        <DropdownMenu.Item
          className="gap-2"
          disabled={row.restore !== null}
          onClick={() => actions.onRestore(row)}
        >
          <ArrowCounterClockwiseIcon size={16} aria-hidden="true" />
          <span className="flex flex-col">
            {t('restore')}
            {why ? <span className="text-sm text-kumo-subtle">{why}</span> : null}
          </span>
        </DropdownMenu.Item>
        <DropdownMenu.Item
          variant="danger"
          className="gap-2"
          disabled={!row.removable}
          onClick={() => actions.onDelete(row)}
        >
          <TrashIcon size={16} aria-hidden="true" />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

function Panel({ row }: { row: BackupRow }) {
  const t = useT(backupsMessages);
  if (!row.panel) return <span className="text-kumo-subtle">{t('gone')}</span>;
  return <PanelLabel kind={panelKind(row.panel)} name={row.panel.name} className="min-w-0" />;
}

/** The database kind, and the backup id under it: the word Put back and Delete ask for. */
function Database({ row }: { row: BackupRow }) {
  return (
    <span className="flex flex-col">
      <span>{backupDatabase[row.backup.method]}</span>
      <span className="text-kumo-subtle">
        <Machine>{row.backup.id}</Machine>
      </span>
    </span>
  );
}

function Taken({ row, now }: { row: BackupRow; now: Date }) {
  const fmt = useFormat();
  const at = new Date(row.backup.createdAt);
  return (
    <time dateTime={at.toISOString()} title={fmt.dateTime(at)} className="whitespace-nowrap">
      {fmt.relative(at, now)}
    </time>
  );
}

function State({ row }: { row: BackupRow }) {
  return row.backup.jobState ? <JobStateLabel state={row.backup.jobState} /> : null;
}

/** On a phone a backup is a card: the move first, then the panel and the backup's facts. */
function BackupCard({ row, now, actions }: { row: BackupRow; now: Date; actions: BackupActions }) {
  const fmt = useFormat();
  return (
    <article className="flex flex-col gap-2 rounded-lg bg-kumo-base px-4 py-3 ring ring-kumo-hairline">
      <div className="flex items-start justify-between gap-2">
        <JobName job={row.job} to={jobPath(row)} />
        <BackupMenu row={row} actions={actions} />
      </div>
      <div className="text-sm">
        <Panel row={row} />
      </div>
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1 border-t border-kumo-hairline pt-2 text-sm">
        <Database row={row} />
        <span className="flex flex-col items-end text-kumo-subtle">
          <span className="tabular">{fmt.bytes(row.backup.size)}</span>
          <Taken row={row} now={now} />
        </span>
      </div>
      <State row={row} />
    </article>
  );
}

/** Backups newest first: the job's move, the panel, the database, size, when, the job's state. */
export function BackupsTable({
  rows,
  now,
  actions,
}: {
  rows: readonly BackupRow[];
  now: Date;
  actions: BackupActions;
}) {
  const t = useT(backupsMessages);
  const fmt = useFormat();
  const columns: Column<BackupRow>[] = [
    {
      key: 'job',
      header: t('colMigration'),
      cell: (r) => <JobName job={r.job} to={jobPath(r)} />,
      className: 'max-w-72',
    },
    { key: 'panel', header: t('colPanel'), cell: (r) => <Panel row={r} />, className: 'max-w-56' },
    { key: 'db', header: t('colDatabase'), cell: (r) => <Database row={r} /> },
    {
      key: 'size',
      header: t('colSize'),
      cell: (r) => <span className="whitespace-nowrap tabular">{fmt.bytes(r.backup.size)}</span>,
    },
    { key: 'taken', header: t('colTaken'), cell: (r) => <Taken row={r} now={now} /> },
    { key: 'state', header: t('colState'), cell: (r) => <State row={r} /> },
    {
      key: 'menu',
      header: t('colActions'),
      hideHeader: true,
      cell: (r) => <BackupMenu row={r} actions={actions} />,
      className: 'w-12 text-end',
    },
  ];
  return (
    <DataTable
      caption={t('list')}
      rows={rows}
      rowKey={(r) => r.backup.id}
      columns={columns}
      card={(r) => <BackupCard row={r} now={now} actions={actions} />}
    />
  );
}
