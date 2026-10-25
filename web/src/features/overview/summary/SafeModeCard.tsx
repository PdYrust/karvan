import { CaretRightIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { StatusBadge } from '../../../components/feedback/StatusBadge';
import { FactList } from '../../../components/layout/FactList';
import { useT } from '../../../i18n/locale';
import type { Backup, SafeMode } from '../../../lib/api/safemode';
import { useFormat } from '../../../lib/preferences/region';
import { useSafeMode } from '../../settings/use-safe-mode';
import { overviewMessages } from '../messages';
import { PartFailed } from '../PartFailed';
import { useBackups } from '../use-overview';
import { failureFacts } from './safe-mode-facts';
import { SummaryCard, SummarySkeleton } from './SummaryCard';

/** "2 backups, 3.4 MB"; a list that failed to load reads as a dash. */
function BackupsValue({ backups, failed }: { backups: Backup[] | undefined; failed: boolean }) {
  const t = useT(overviewMessages);
  const fmt = useFormat();
  if (!backups) return failed ? '-' : <Bone className="h-4 w-24" />;
  if (backups.length === 0) return t('backupsNone');
  const size = backups.reduce((n, b) => n + b.size, 0);
  return (
    <Link to="/backups" className="text-(--color-kumo-brand) hover:underline">
      {t('backupsValue', { count: backups.length, size: fmt.bytes(size) })}
    </Link>
  );
}

function Facts({ safe }: { safe: SafeMode }) {
  const t = useT(overviewMessages);
  const backups = useBackups();
  return (
    <>
      <FactList
        facts={[
          ...failureFacts(safe, t),
          {
            label: t('backups'),
            value: <BackupsValue backups={backups.data} failed={backups.isError} />,
          },
        ]}
      />
      <Link
        to="/settings/safe-mode"
        className="mt-auto inline-flex items-center gap-1 self-start text-sm text-(--color-kumo-brand) hover:underline pointer-coarse:min-h-11"
      >
        {t('safeModeSettings')}
        <CaretRightIcon size={12} aria-hidden="true" className="rtl:-scale-x-100" />
      </Link>
    </>
  );
}

/** Safe Mode at a glance: on or off, what a failed migration does, and the backups it keeps. */
export function SafeModeCard() {
  const t = useT(overviewMessages);
  const q = useSafeMode();
  const badge = q.data ? (
    <StatusBadge tone={q.data.enabled ? 'success' : 'warning'}>
      {t(q.data.enabled ? 'on' : 'off')}
    </StatusBadge>
  ) : undefined;
  let body;
  if (q.data) {
    body = <Facts safe={q.data} />;
  } else if (q.isError) {
    body = <PartFailed message={t('cardFailed')} onRetry={() => void q.refetch()} />;
  } else {
    body = <SummarySkeleton label={t('safeModeTitle')} />;
  }
  return (
    <SummaryCard title={t('safeModeTitle')} badge={badge}>
      {body}
    </SummaryCard>
  );
}
