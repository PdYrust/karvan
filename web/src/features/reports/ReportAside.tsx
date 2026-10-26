import { StatusBadge, type StatusTone } from '../../components/feedback/StatusBadge';
import { AsideBlock } from '../../components/layout/AsideBlock';
import { FactList, type Fact } from '../../components/layout/FactList';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { CodeText } from '../../components/text/CodeText';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import type { JobState } from '../../lib/api/migrations';
import type { Report } from '../../lib/api/report';
import { useFormat } from '../../lib/preferences/region';
import { useReload } from '../live/meaning';
import { liveMessages } from '../live/messages';
import { reportsMessages } from './messages';

/** How the admin landed on the new panel, and whose password it has. */
function AdminBlock({ report }: { report: Report }) {
  const t = useT(reportsMessages);
  const { admin } = report;
  if (!admin.username) return null;
  return (
    <AsideBlock title={t('admin')}>
      <p className="text-kumo-default">
        <CodeText text={t(admin.merged ? 'adminMerged' : 'adminNew', { name: admin.username })} />
      </p>
      {admin.password ? (
        <p className="text-sm text-kumo-subtle">
          {`${t('password')}: ${t(`password_${admin.password}`)}`}
        </p>
      ) : null}
    </AsideBlock>
  );
}

/** Whether Safe Mode was on, how a rollback works then, and the backup it took. */
function SafeModeBlock({ report }: { report: Report }) {
  const t = useT(reportsMessages);
  const on = report.safeMode;
  return (
    <AsideBlock
      title={t('safeMode')}
      badge={
        <StatusBadge tone={on ? 'success' : 'neutral'}>{t(on ? 'safeOn' : 'safeOff')}</StatusBadge>
      }
    >
      {on ? (
        <p className="text-sm text-kumo-default">
          {t(report.rollback === 'backup' ? 'rollbackBackup' : 'rollbackUndo')}
        </p>
      ) : null}
      {report.backupId ? (
        <FactList facts={[{ label: t('backup'), value: <Machine>{report.backupId}</Machine> }]} />
      ) : null}
    </AsideBlock>
  );
}

const reloadTones = {
  ok: 'success',
  partial: 'warning',
  error: 'danger',
  manual: 'neutral',
} as const;

/** Whether the panel loaded the new users; a reload left to do shows its command. */
function ReloadBlock({ report }: { report: Report }) {
  const t = useT(reportsMessages);
  const tl = useT(liveMessages);
  const r = useReload(report);
  if (!r) return null;
  const tone: StatusTone = reloadTones[r.kind];
  return (
    <AsideBlock
      title={tl('reloadTitle')}
      badge={<StatusBadge tone={tone}>{t(`reload_${r.kind}`)}</StatusBadge>}
    >
      <p className="text-sm text-kumo-default">
        <CodeText text={r.text} />
      </p>
    </AsideBlock>
  );
}

/** Where the move went and when it started and ended; the header says how long it took. */
function DetailsBlock({ report }: { report: Report }) {
  const t = useT(reportsMessages);
  const fmt = useFormat();
  const facts: Fact[] = [
    { label: t('oldPanel'), value: <Machine>{report.source.origin}</Machine> },
    {
      label: t('newPanel'),
      value: (
        <PanelLabel
          kind={report.destination.kind || undefined}
          name={report.destination.name || undefined}
        />
      ),
    },
    { label: t('started'), value: fmt.dateTime(new Date(report.startedAt)) },
    { label: t('ended'), value: fmt.dateTime(new Date(report.endedAt)) },
  ];
  return (
    <AsideBlock title={t('facts')}>
      <FactList facts={facts} />
    </AsideBlock>
  );
}

/** The report's aside: the admin, Safe Mode and its backup, the reload, then the details. */
export function ReportAside({ state, report }: { state: JobState; report: Report }) {
  return (
    <div className="flex flex-col">
      <AdminBlock report={report} />
      <SafeModeBlock report={report} />
      {state === 'done' ? <ReloadBlock report={report} /> : null}
      <DetailsBlock report={report} />
    </div>
  );
}
