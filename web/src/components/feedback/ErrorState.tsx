import { Button } from '@cloudflare/kumo/components/button';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { ApiError, offline } from '../../lib/api/client';
import { InlineAlert } from './InlineAlert';

/**
 * A first load that failed: what did not load, whether the network or the
 * server is to blame, and a retry. Later failed refreshes keep what is shown.
 */
export function ErrorState({
  message,
  error,
  onRetry,
}: {
  message: string;
  error: unknown;
  onRetry: () => void;
}) {
  const ts = useT(sharedMessages);
  const isOffline = error instanceof ApiError && error.code === offline;
  return (
    <div className="flex flex-col items-start gap-3">
      <InlineAlert>
        {message} {isOffline ? ts('errorOffline') : ts('errorServer')}
      </InlineAlert>
      <Button className="pointer-coarse:h-11" onClick={onRetry}>
        {ts('retry')}
      </Button>
    </div>
  );
}
