import { StatusLabel } from '../../components/feedback/StatusBadge';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { jobOutcome, type JobRow, type JobState } from '../../lib/api/migrations';
import { jobStates } from './job-state';
import { migrationsMessages } from './messages';

/** "1,170 moved, 30 skipped"; nothing before a run ends or after a rollback took the users back. */
export function JobOutcomeText({ job }: { job: Pick<JobRow, 'state' | 'counts'> }) {
  const t = useT(migrationsMessages);
  const o = jobOutcome(job);
  if (!o) return null;
  return o.skipped > 0
    ? t('movedSkipped', { moved: o.moved, skipped: o.skipped })
    : t('moved', { count: o.moved });
}

/** A job's state as an icon and a word. */
export function JobStateLabel({ state }: { state: JobState }) {
  const t = useT(migrationsMessages);
  const look = jobStates[state];
  return <StatusLabel tone={look.tone}>{t(look.word)}</StatusLabel>;
}

/** The old panel's logo, an arrow and the new panel's logo, read out as "Marzban to PasarGuard". */
export function JobMove({ job, size }: { job: Pick<JobRow, 'source' | 'panel'>; size?: number }) {
  const t = useT(migrationsMessages);
  const from = job.source.kind;
  const to = job.panel.kind;
  if (!to) return null;
  return (
    <MoveLabel
      from={from}
      to={to}
      size={size}
      label={t('move', { from: panelNames[from], to: panelNames[to] })}
    />
  );
}
