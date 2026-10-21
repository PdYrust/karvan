import { Button } from '@cloudflare/kumo/components/button';
import { WarningCircleIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';

/** A part of the overview that did not load, with a retry; the rest of the page stays. */
export function PartFailed({ message, onRetry }: { message: string; onRetry: () => void }) {
  const ts = useT(sharedMessages);
  return (
    <div
      role="alert"
      className="flex min-h-14 flex-wrap items-center justify-between gap-x-3 gap-y-1 rounded-lg bg-kumo-tint px-4 py-2 text-sm text-kumo-default"
    >
      <span className="flex items-center gap-2">
        <WarningCircleIcon
          size={16}
          weight="bold"
          aria-hidden="true"
          className="text-kumo-danger"
        />
        {message}
      </span>
      <Button variant="ghost" size="sm" onClick={onRetry} className="pointer-coarse:h-11">
        {ts('retry')}
      </Button>
    </div>
  );
}
