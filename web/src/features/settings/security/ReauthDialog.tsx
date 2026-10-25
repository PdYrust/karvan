import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { XIcon } from '@phosphor-icons/react';
import { useId, useState, type FormEvent, type ReactNode } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { FormField } from '../../../components/form/FormField';
import { PasswordInput } from '../../../components/form/PasswordInput';
import { useT } from '../../../i18n/locale';
import { ApiError } from '../../../lib/api/client';
import type { Reauth } from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { useSession } from '../../session/use-session';
import { CodeField } from './CodeField';
import { securityMessages } from './messages';

/**
 * Asks for the current password, and the code when two-step sign-in is on,
 * before a change to the login, the address or two-step sign-in itself.
 * The fields are cleared whenever the dialog closes.
 */
export function ReauthDialog({
  open,
  onOpenChange,
  title,
  description,
  action,
  danger = false,
  needsCode,
  ready = true,
  pending,
  error,
  onSubmit,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  /** The button label: what happens. */
  action: string;
  danger?: boolean;
  /** Ask for a code even when the session does not say two-step is on. */
  needsCode?: boolean;
  /** Whatever else the dialog asks for is filled in. */
  ready?: boolean;
  pending: boolean;
  error: unknown;
  onSubmit: (r: Reauth) => void;
  /** Fields of the change itself, above the password. */
  children?: ReactNode;
}) {
  const t = useT(securityMessages);
  const errorText = useErrorText();
  const session = useSession();
  const id = useId();
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [wasOpen, setWasOpen] = useState(open);
  // The caller closes the dialog after a success too, so the fields clear on any close.
  if (open !== wasOpen) {
    setWasOpen(open);
    if (!open) {
      setPassword('');
      setCode('');
    }
  }
  const askCode = needsCode ?? session.data?.operator?.totpEnabled ?? false;
  const filled = ready && password !== '' && (!askCode || code.length === 6);

  function submit(e: FormEvent) {
    e.preventDefault();
    if (filled && !pending) onSubmit({ password, code: askCode ? code : '' });
  }

  const message =
    error instanceof ApiError && error.code === 'invalid_credentials'
      ? t('wrongPassword')
      : error
        ? errorText(error)
        : null;

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
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
            <Dialog.Description className="text-kumo-subtle">{description}</Dialog.Description>
            {children}
            {message ? <InlineAlert>{message}</InlineAlert> : null}
            <FormField id={`${id}-password`} label={t('password')}>
              <PasswordInput
                id={`${id}-password`}
                aria-labelledby={`${id}-password-label`}
                autoComplete="current-password"
                showLabel={t('showPassword')}
                hideLabel={t('hidePassword')}
                value={password}
                disabled={pending}
                onChange={(e) => setPassword(e.target.value)}
                className="h-10 pointer-coarse:h-11"
              />
            </FormField>
            {askCode ? (
              <CodeField
                id={`${id}-code`}
                label={t('code')}
                value={code}
                onChange={setCode}
                disabled={pending}
              />
            ) : null}
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
              variant={danger ? 'destructive' : 'primary'}
              disabled={!filled || pending}
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
