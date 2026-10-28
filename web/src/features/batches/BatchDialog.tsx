import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { XIcon } from '@phosphor-icons/react';
import type { FormEvent, ReactNode } from 'react';
import { dialogMessages } from '../../components/dialogs/messages';
import { useT } from '../../i18n/locale';

/**
 * The frame of a batch dialog, as TypeToConfirm draws its own: the title
 * with close, the body, then the buttons under a hairline. With `onSubmit`
 * the body is a form, so Enter saves.
 */
export function BatchDialog({
  open,
  onOpenChange,
  title,
  busy = false,
  onSubmit,
  footer,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  busy?: boolean;
  onSubmit?: () => void;
  footer: ReactNode;
  children: ReactNode;
}) {
  const t = useT(dialogMessages);
  const inner = (
    <>
      <div className="flex items-center justify-between gap-4 border-b border-kumo-hairline px-6 py-4">
        <Dialog.Title className="min-w-0 text-heading font-semibold break-words">
          {title}
        </Dialog.Title>
        <Dialog.Close
          render={(props) => (
            <Button
              {...props}
              variant="ghost"
              shape="square"
              size="sm"
              className="pointer-coarse:size-9"
              aria-label={t('close')}
              disabled={busy}
              icon={<XIcon size={18} />}
            />
          )}
        />
      </div>
      <div className="flex flex-col gap-4 px-6 py-5">{children}</div>
      <div className="flex justify-end gap-2 border-t border-kumo-hairline px-6 py-4">{footer}</div>
    </>
  );
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!busy) onSubmit?.();
  };
  return (
    <Dialog.Root open={open} onOpenChange={(next) => !busy && onOpenChange(next)}>
      <Dialog className="p-0" size="lg">
        {onSubmit ? <form onSubmit={submit}>{inner}</form> : inner}
      </Dialog>
    </Dialog.Root>
  );
}
