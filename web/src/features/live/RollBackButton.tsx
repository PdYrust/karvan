import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { ArrowCounterClockwiseIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { rollbackJob } from '../../lib/api/report';
import { useErrorText } from '../../lib/errors/errors';
import { liveMessages } from './messages';

/**
 * Roll back (B5) asks first and says what it removes. Once the server
 * holds the panel, the undo follows on the job's stream and this page shows it.
 */
export function RollBackButton({ jobId, backup }: { jobId: string; backup: boolean }) {
  const t = useT(liveMessages);
  const errorText = useErrorText();
  const [open, setOpen] = useState(false);
  const roll = useMutation({
    mutationFn: () => rollbackJob(jobId),
    onSettled: () => setOpen(false),
  });
  return (
    <>
      <Button
        variant="primary"
        icon={<ArrowCounterClockwiseIcon size={16} />}
        className="pointer-coarse:h-11"
        loading={roll.isPending || roll.isSuccess}
        onClick={() => setOpen(true)}
      >
        {t('rollBack')}
      </Button>
      {roll.isError ? (
        <InlineAlert className="basis-full">{`${t('rollBackFailed')} ${errorText(roll.error)}`}</InlineAlert>
      ) : null}
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog className="flex flex-col gap-4 p-6" size="base">
          <Dialog.Title className="text-heading font-semibold">{t('rollBackTitle')}</Dialog.Title>
          <Dialog.Description className="text-kumo-subtle">
            {t(backup ? 'rollBackBackup' : 'rollBackUndo')}
          </Dialog.Description>
          <div className="flex flex-wrap justify-end gap-2">
            <Button className="pointer-coarse:h-11" onClick={() => setOpen(false)}>
              {t('cancel')}
            </Button>
            <Button
              variant="destructive"
              className="pointer-coarse:h-11"
              loading={roll.isPending}
              onClick={() => roll.mutate()}
            >
              {t('rollBack')}
            </Button>
          </div>
        </Dialog>
      </Dialog.Root>
    </>
  );
}
