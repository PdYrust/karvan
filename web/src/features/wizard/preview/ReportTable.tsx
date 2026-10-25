import { ArrowRightIcon } from '@phosphor-icons/react';
import { DataTable, type Column } from '../../../components/data/DataTable';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { CheckUser } from '../../../lib/api/preflight';
import { wizardMessages } from '../messages';
import { previewMessages } from './messages';
function Name({ u }: { u: CheckUser }) {
  return (
    <span className="flex flex-wrap items-center gap-x-1.5">
      <bdi dir="ltr" className="font-medium text-kumo-strong">
        {u.sourceUsername}
      </bdi>
      {u.username !== u.sourceUsername ? (
        <>
          <ArrowRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
          <bdi dir="ltr">{u.username}</bdi>
        </>
      ) : null}
    </span>
  );
}
/** Machine values (fixes, operations, conflict types) one after another, or "None". */
function Codes({ codes }: { codes: readonly string[] }) {
  const t = useT(previewMessages);
  if (codes.length === 0) return <span className="text-kumo-subtle">{t('none')}</span>;
  return (
    <span className="flex flex-wrap gap-x-2 gap-y-0.5">
      {codes.map((c) => (
        <Machine key={c}>{c}</Machine>
      ))}
    </span>
  );
}
/** Every user in the pre-flight report, in key order. */
export function ReportTable({
  users,
  names,
}: {
  users: readonly CheckUser[];
  /** Target id to its name on the destination. */
  names: ReadonlyMap<string, string>;
}) {
  const t = useT(previewMessages);
  const tw = useT(wizardMessages);
  const outcome = (u: CheckUser) =>
    u.adopted ? t('outAdopted') : u.moving ? t('outMoves') : tw('leftOut');
  const where = (u: CheckUser) =>
    u.moving && u.targets.length > 0 ? u.targets.map((id) => names.get(id) ?? id).join(', ') : '';
  const changes = (u: CheckUser) => [...u.fixed, ...u.applied.filter((a) => !a.startsWith('fix:'))];
  const open = (u: CheckUser) => u.conflicts.map((c) => (c.fix ? `${c.type}:${c.fix}` : c.type));
  const flags = (u: CheckUser) =>
    [u.changesConfig ? t('configFlag') : '', u.changesLink ? t('linkFlag') : '']
      .filter(Boolean)
      .join(', ');
  const columns: Column<CheckUser>[] = [
    { key: 'user', header: t('colUser'), cell: (u) => <Name u={u} /> },
    {
      key: 'outcome',
      header: t('colOutcome'),
      cell: (u) => (
        <span className="flex flex-col">
          <span>{outcome(u)}</span>
          <span className="text-sm text-kumo-subtle">{flags(u)}</span>
        </span>
      ),
    },
    { key: 'where', header: t('colWhere'), cell: (u) => <bdi dir="auto">{where(u)}</bdi> },
    { key: 'changes', header: t('colChanges'), cell: (u) => <Codes codes={changes(u)} /> },
    { key: 'open', header: t('colOpen'), cell: (u) => <Codes codes={open(u)} /> },
  ];
  return (
    <DataTable
      caption={t('printTitle')}
      rows={users}
      rowKey={(u) => u.key}
      columns={columns}
      card={(u) => (
        <div className="flex flex-col gap-1 rounded-lg bg-kumo-base p-3 ring ring-kumo-hairline">
          <Name u={u} />
          <span className="flex flex-wrap gap-x-2 text-sm">
            <span>{outcome(u)}</span>
            {where(u) ? <bdi dir="auto">{where(u)}</bdi> : null}
          </span>
          {flags(u) ? <span className="text-sm text-kumo-subtle">{flags(u)}</span> : null}
          {changes(u).length > 0 ? <Codes codes={changes(u)} /> : null}
        </div>
      )}
    />
  );
}
