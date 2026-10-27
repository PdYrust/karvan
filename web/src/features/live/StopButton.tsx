import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { cancelJob } from '../../lib/api/migrations';
import { useErrorText } from '../../lib/errors/errors';
import { liveMessages } from './messages';

/**
 * Stop asks first, and says what Safe Mode does with what was written. The
 * run stops between batches, so the button says so until it does.
 */
export function StopButton({ jobId, safeMode }: { jobId: string; safeMode: boolean }) {
  const t = useT(liveMessages);
  const errorText = useErrorText();
  const [open, setOpen] = useState(false);
  const stop = useMutation({
    mutationFn: () => cancelJob(jobId),
    onSettled: () => setOpen(false),
  });
  return (
    <div className="flex flex-col items-end gap-2">
      {stop.isSuccess ? (
        <p role="status" className="text-sm text-kumo-subtle">
          {t('stopping')}
        </p>
      ) : (
        <Button
          className="pointer-coarse:h-9"
          loading={stop.isPending}
          onClick={() => setOpen(true)}
        >
          {t('stop')}
        </Button>
      )}
      {stop.isError ? (
        <InlineAlert>{`${t('stopFailed')} ${errorText(stop.error)}`}</InlineAlert>
      ) : null}
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog className="flex flex-col gap-4 p-6" size="base">
          <Dialog.Title className="text-heading font-semibold">{t('stopTitle')}</Dialog.Title>
          <Dialog.Description className="text-kumo-subtle">
            {t(safeMode ? 'stopSafe' : 'stopOff')}
          </Dialog.Description>
          <div className="flex flex-wrap justify-end gap-2">
            <Button className="pointer-coarse:h-9" onClick={() => setOpen(false)}>
              {t('keepGoing')}
            </Button>
            <Button
              variant="destructive"
              className="pointer-coarse:h-9"
              loading={stop.isPending}
              onClick={() => stop.mutate()}
            >
              {t('stopConfirm')}
            </Button>
          </div>
        </Dialog>
      </Dialog.Root>
    </div>
  );
}
