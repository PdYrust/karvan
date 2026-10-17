import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { Input } from '@cloudflare/kumo/components/input';
import { XIcon } from '@phosphor-icons/react';
import { useId, useState, type FormEvent, type ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { InlineAlert } from '../feedback/InlineAlert';
import { dialogMessages } from './messages';

/**
 * Kumo's delete-resource block, rebuilt with logical classes: the dangerous
 * action stays disabled until the exact word is typed.
 */
export function TypeToConfirm({
  open,
  onOpenChange,
  title,
  children,
  word,
  action,
  pending = false,
  error,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** What happens, in one or two short sentences. */
  children: ReactNode;
  /** What the operator must type, exactly. */
  word: string;
  /** The button label: what happens ("Turn off Safe Mode"). */
  action: string;
  pending?: boolean;
  error?: string;
  onConfirm: () => void;
}) {
  const t = useT(dialogMessages);
  const id = useId();
  const [typed, setTyped] = useState('');
  const matches = typed.trim() === word;

  function close(next: boolean) {
    if (!next) setTyped('');
    onOpenChange(next);
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (matches && !pending) onConfirm();
  }

  return (
    <Dialog.Root open={open} onOpenChange={close}>
      <Dialog className="p-0" size="lg">
        <form onSubmit={submit}>
          <div className="flex items-center justify-between gap-4 border-b border-kumo-hairline px-6 py-4">
            <Dialog.Title className="text-heading font-semibold">{title}</Dialog.Title>
            <Dialog.Close
              render={(props) => (
                <Button
                  {...props}
                  variant="ghost"
                  shape="square"
                  size="sm"
                  aria-label={t('close')}
                  disabled={pending}
                  icon={<XIcon size={18} />}
                />
              )}
            />
          </div>
          <div className="flex flex-col gap-4 px-6 py-5">
            <Dialog.Description className="text-kumo-subtle">{children}</Dialog.Description>
            {error ? <InlineAlert>{error}</InlineAlert> : null}
            <div className="flex flex-col gap-2">
              <label htmlFor={id} id={`${id}-label`} className="text-sm text-kumo-default">
                {t('typeToConfirm')}
                <bdi
                  dir="ltr"
                  className="ms-1 rounded-md bg-kumo-tint px-1.5 py-0.5 font-mono text-sm"
                >
                  {word}
                </bdi>
              </label>
              <Input
                id={id}
                aria-labelledby={`${id}-label`}
                dir="ltr"
                autoComplete="off"
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                value={typed}
                disabled={pending}
                onChange={(e) => setTyped(e.target.value)}
                className="h-10 font-mono pointer-coarse:h-11"
              />
            </div>
          </div>
          <div className="flex justify-end gap-2 border-t border-kumo-hairline px-6 py-4">
            <Dialog.Close
              render={(props) => (
                <Button {...props} variant="secondary" disabled={pending}>
                  {t('cancel')}
                </Button>
              )}
            />
            <Button
              type="submit"
              variant="destructive"
              disabled={!matches || pending}
              loading={pending}
            >
              {action}
            </Button>
          </div>
        </form>
      </Dialog>
    </Dialog.Root>
  );
}
