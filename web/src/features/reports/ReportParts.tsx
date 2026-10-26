import { cn } from '@cloudflare/kumo/utils';
import { CheckCircleIcon, XCircleIcon } from '@phosphor-icons/react';
import type { Fact } from '../../components/layout/FactList';
import { Section } from '../../components/layout/Section';
import { PanelLabel } from '../../components/panels/PanelLabel';
import { CodeText } from '../../components/text/CodeText';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import type { JobState } from '../../lib/api/migrations';
import type { Report, StageTime } from '../../lib/api/report';
import { useFormat } from '../../lib/preferences/region';
import { liveMessages, type LiveKey } from '../live/messages';
import { reportsMessages, type ReportsKey } from './messages';
import { knownUndoCodes, undoWindow } from './report-view';

const seconds = (from: string, to: string) => (Date.parse(to) - Date.parse(from)) / 1000;

/** Times, the panels, the admin and Safe Mode as fact rows. */
export function useReportFacts(report: Report): Fact[] {
  const t = useT(reportsMessages);
  const fmt = useFormat();
  const { admin } = report;
  const took = seconds(report.startedAt, report.endedAt);
  const safe = !report.safeMode
    ? t('safeOff')
    : report.rollback === 'backup'
      ? t('safeBackup')
      : t('safeUndo');
  const facts: (Fact | null)[] = [
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
    admin.username
      ? {
          label: t('admin'),
          value: (
            <CodeText
              text={t(admin.merged ? 'adminMerged' : 'adminNew', { name: admin.username })}
            />
          ),
        }
      : null,
    admin.password ? { label: t('password'), value: t(`password_${admin.password}`) } : null,
    { label: t('started'), value: fmt.dateTime(new Date(report.startedAt)) },
    { label: t('ended'), value: fmt.dateTime(new Date(report.endedAt)) },
    took >= 0 ? { label: t('took'), value: fmt.duration(Math.max(1, took)) } : null,
    { label: t('safeMode'), value: safe },
    report.backupId ? { label: t('backup'), value: <Machine>{report.backupId}</Machine> } : null,
  ];
  return facts.filter((f): f is Fact => f !== null);
}

const knownStages = new Set(['check', 'backup', 'execute', 'verify', 'reload', 'undo', 'restore']);
/** A stage's name; one Karvan has no word for keeps its machine name. */
const stageName = (tl: (k: LiveKey) => string, s: string) =>
  knownStages.has(s) ? tl(`stage_${s}` as LiveKey) : s;

/** Each stage the run went through as a timeline: how it ended, how long it took and when it started. */
export function Stages({ stages }: { stages: readonly StageTime[] }) {
  const t = useT(reportsMessages);
  const tl = useT(liveMessages);
  const fmt = useFormat();
  return (
    <ol className="flex flex-col">
      {stages.map((s, i) => {
        const took = seconds(s.started, s.ended);
        const time = took >= 1 ? fmt.duration(took) : tl('underSecond');
        const Icon = s.ok ? CheckCircleIcon : XCircleIcon;
        return (
          <li
            // A stage can run twice (a rollback after a reload); its place keeps it apart.
            key={`${s.stage}-${String(i)}`}
            className="relative flex gap-3 pb-5 last:pb-0"
          >
            {i < stages.length - 1 ? (
              // The line from this stage's icon down to the next one's.
              <span
                aria-hidden="true"
                className="absolute start-[9.5px] top-6 bottom-1 w-px bg-kumo-line"
              />
            ) : null}
            <Icon
              size={20}
              weight="bold"
              aria-hidden="true"
              className={cn('shrink-0', s.ok ? 'text-kumo-success' : 'text-kumo-danger')}
            />
            <div className="flex min-w-0 flex-1 flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5">
              <span className="font-medium text-kumo-strong">{stageName(tl, s.stage)}</span>
              <span className="text-sm text-kumo-subtle tabular">
                {t('stageAt', { time: fmt.time(new Date(s.started)) })}
              </span>
              <span className="basis-full text-sm text-kumo-subtle">
                {tl(s.ok ? 'took' : 'failedAfter', { time })}
              </span>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

/** What a rollback cannot take back, and until when it can take the users back. */
export function UndoCard({ state, report }: { state: JobState; report: Report }) {
  const t = useT(reportsMessages);
  const fmt = useFormat();
  const codes = knownUndoCodes(report);
  const w = undoWindow(state, report);
  const line =
    w.kind === 'open'
      ? t('windowOpen')
      : w.kind === 'reloaded'
        ? t('windowReloaded', { time: fmt.dateTime(new Date(w.at)) })
        : t('windowClosed');
  return (
    <Section title={t('undoTitle')}>
      <div className="flex flex-col gap-3 px-4 pb-4">
        <p className="text-kumo-default">{line}</p>
        {report.backupId ? <p className="text-kumo-default">{t('backupKept')}</p> : null}
        {codes.length > 0 ? (
          <ul className="flex list-disc flex-col gap-1 ps-5 text-kumo-default">
            {codes.map((c) => (
              <li key={c}>{t(`undo_${c}` as ReportsKey)}</li>
            ))}
          </ul>
        ) : null}
      </div>
    </Section>
  );
}
