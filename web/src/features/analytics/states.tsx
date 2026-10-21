import { Button } from '@cloudflare/kumo/components/button';
import { WarningCircleIcon } from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { StatTileSkeleton } from '../../components/charts/StatTile';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { ApiError, offline } from '../../lib/api/client';
import { findingMessages, findingText } from '../panels/findings';
import { analyticsMessages } from './messages';

/** The icon of a notice: large and light, as in an empty state. */
export const noticeIcon = { size: 40, weight: 'light', 'aria-hidden': true } as const;

/** A tab while its first numbers load: tiles and two cards, the shape it will take. */
export function TabSkeleton({ label }: { label: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <StatTileSkeleton key={i} />
        ))}
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Bone className="h-72 w-full rounded-lg" />
        <Bone className="h-72 w-full rounded-lg" />
      </div>
    </div>
  );
}

/** The first load failed; later failed polls keep the numbers already on screen. */
export function LoadFailed({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT(analyticsMessages);
  const ts = useT(sharedMessages);
  const isOffline = error instanceof ApiError && error.code === offline;
  return (
    <div className="flex flex-col items-start gap-3">
      <InlineAlert>
        {t('loadFailed')} {isOffline ? ts('errorOffline') : ts('errorServer')}
      </InlineAlert>
      <Button className="pointer-coarse:h-11" onClick={onRetry}>
        {ts('retry')}
      </Button>
    </div>
  );
}

/** A tab with nothing to draw yet: an icon, why, and what to do. */
export function Notice({
  icon,
  children,
  action,
}: {
  icon: ReactNode;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-lg bg-kumo-base px-6 py-12 text-center ring ring-kumo-hairline">
      <span className="text-kumo-subtle">{icon}</span>
      <p className="max-w-md text-kumo-default">{children}</p>
      {action}
    </div>
  );
}

/** The finding that keeps Karvan out of a panel's database, with how to fix it. */
export function DatabaseProblem({ code }: { code: string }) {
  const t = useT(analyticsMessages);
  const tf = useT(findingMessages);
  const text = findingText(code);
  return (
    <Notice icon={<WarningCircleIcon {...noticeIcon} />}>
      {text ? <CodeText text={tf(text.what)} /> : t('problemOther')}
      {text ? (
        <span className="mt-1 block text-sm text-kumo-subtle">
          <CodeText text={tf(text.fix)} />
        </span>
      ) : null}
    </Notice>
  );
}
