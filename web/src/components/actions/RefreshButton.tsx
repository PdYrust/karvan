import { RefreshButton as KumoRefreshButton } from '@cloudflare/kumo/components/button';
import type { ComponentProps } from 'react';
import { refreshTurn, useMinDuration } from '../../lib/motion/use-min-duration';

/** Kumo's refresh button, whose icon turns at least once and stops only at the end of a turn. */
export function RefreshButton({
  loading = false,
  ...props
}: ComponentProps<typeof KumoRefreshButton>) {
  const turning = useMinDuration(loading, refreshTurn, refreshTurn);
  return <KumoRefreshButton loading={turning} {...props} />;
}
