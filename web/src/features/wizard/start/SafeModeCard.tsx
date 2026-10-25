import { buttonVariants } from '@cloudflare/kumo/components/button';
import { cn } from '@cloudflare/kumo/utils';
import { useId } from 'react';
import { Link } from 'react-router';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { StatusBadge } from '../../../components/feedback/StatusBadge';
import { useT } from '../../../i18n/locale';
import type { PanelControl } from '../../../lib/api/panels';
import type { SafeMode } from '../../../lib/api/safemode';
import { errorMessages } from '../../../lib/errors/errors';
import { startMessages } from './messages';
/** Safe Mode as the run will read it, with its rollback and the way to Settings. */
export function SafeModeCard({
  safe,
  control,
}: {
  safe: { data?: SafeMode; error: unknown; refetch: () => unknown };
  /** The destination's panel control; a backup rollback needs it. */
  control: PanelControl | undefined;
}) {
  const t = useT(startMessages);
  const te = useT(errorMessages);
  const titleId = useId();
  const s = safe.data;
  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-3 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline sm:p-5"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2
          id={titleId}
          className="flex items-center gap-2 text-heading font-semibold text-kumo-strong"
        >
          {t('safeTitle')}
          {s ? (
            <StatusBadge tone={s.enabled ? 'success' : 'warning'}>
              {t(s.enabled ? 'safeOn' : 'safeOff')}
            </StatusBadge>
          ) : null}
        </h2>
        <Link
          to="/settings/safe-mode"
          className={cn(buttonVariants({ variant: 'secondary' }), 'pointer-coarse:h-11')}
        >
          {t('safeChange')}
        </Link>
      </div>
      {safe.error ? (
        <ErrorState
          message={t('safeFailed')}
          error={safe.error}
          onRetry={() => void safe.refetch()}
        />
      ) : !s ? (
        <Bone className="h-10 w-full" />
      ) : (
        <p className="max-w-[720px] text-kumo-default">
          {!s.enabled
            ? t('safeOffBody')
            : s.rollback === 'backup'
              ? t('safeBackup')
              : t('safeUndo')}
        </p>
      )}
      {s?.enabled && s.rollback === 'backup' && control && !control.available ? (
        <InlineAlert tone="warning">{te('panel_control_unavailable')}</InlineAlert>
      ) : null}
    </section>
  );
}
