import { Button } from '@cloudflare/kumo/components/button';
import { LinkSimpleIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { dialogMessages } from '../../components/dialogs/messages';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { useT } from '../../i18n/locale';
import { showBatchLink, type BatchUser } from '../../lib/api/batches';
import { batchErrorMessages, batchErrorText } from '../../lib/errors/batches';
import { useErrorText } from '../../lib/errors/errors';
import { useLingering } from '../../lib/motion/use-lingering';
import { CopyValue } from '../settings/security/CopyValue';
import { BatchDialog } from './BatchDialog';
import { batchesMessages } from './messages';

/**
 * Show new link: each press reads the link from the new panel
 * once and shows it with Copy. Closing the dialog drops it from the page.
 * Render `dialog` once next to the list and call `show` from a row.
 */
export function useShowLink(batchId: number) {
  const t = useT(batchesMessages);
  const td = useT(dialogMessages);
  const tb = useT(batchErrorMessages);
  const shared = useErrorText();
  const [user, setUser] = useState<BatchUser | null>(null);
  // The name stays while the dialog fades out.
  const named = useLingering(user);
  const read = useMutation({ mutationFn: (u: BatchUser) => showBatchLink(batchId, u.id) });
  const close = () => {
    setUser(null);
    read.reset();
  };
  const show = (u: BatchUser) => {
    setUser(u);
    read.mutate(u);
  };
  let body;
  if (read.isError) {
    body = (
      <InlineAlert>{`${t('linkFailed')} ${batchErrorText(tb, shared, read.error)}`}</InlineAlert>
    );
  } else if (read.data) {
    body = <CopyValue label={t('linkLabel')} value={read.data} />;
  } else {
    body = (
      <div role="status" aria-label={t('linkLabel')} className="flex flex-col gap-2">
        <Bone className="h-4 w-32" />
        <Bone className="h-10 w-full" />
      </div>
    );
  }
  const dialog = (
    <BatchDialog
      open={user !== null}
      onOpenChange={(open) => !open && close()}
      title={t('linkTitle', { name: named?.username ?? '' })}
      footer={
        <Button className="pointer-coarse:h-9" onClick={close}>
          {td('close')}
        </Button>
      }
    >
      <p className="text-kumo-subtle">{t('linkBody')}</p>
      {body}
    </BatchDialog>
  );
  return { show, dialog };
}

/** A row's Show new link button. */
export function ShowLinkButton({ onShow }: { onShow: () => void }) {
  const t = useT(batchesMessages);
  return (
    <Button
      size="sm"
      icon={<LinkSimpleIcon size={16} />}
      className="whitespace-nowrap pointer-coarse:h-9"
      onClick={onShow}
    >
      {t('showLink')}
    </Button>
  );
}
