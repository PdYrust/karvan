import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { CheckCircleIcon, WarningCircleIcon } from '@phosphor-icons/react';
import { useRef, useState } from 'react';
import { PasswordInput } from '../../../components/form/PasswordInput';
import { CopyButton } from '../../../components/text/CopyButton';
import { useT } from '../../../i18n/locale';
import type { PanelKind } from '../../../lib/api/panels';
import { errorKey, errorMessages } from '../../../lib/errors/errors';
import { generatePassword, passwordRule } from '../../../lib/password/password';
import { useWizard } from '../wizard-context';
import { compareMessages } from './messages';

/**
 * The moved admin's password: the reseller's own (or one Karvan makes at
 * the run), or a new one typed or generated here. A new one lives
 * only in the wizard's state and shows once, with copy.
 */
export function PasswordChoice({
  canKeep,
  username,
  kind,
}: {
  canKeep: boolean;
  /** The admin's name on the destination, which the password may not contain. */
  username: string;
  kind: PanelKind;
}) {
  const t = useT(compareMessages);
  const te = useT(errorMessages);
  const { password, setPassword } = useWizard();
  const field = useRef<HTMLInputElement>(null);
  const [made, setMade] = useState(0);
  const rule = password.value ? passwordRule(password.value, username, kind) : null;
  const status = !password.value
    ? t('passwordEmpty')
    : rule
      ? te(errorKey('weak_password', { rule }))
      : t('passwordOk');
  return (
    <div className="flex flex-col gap-3">
      <Radio.Group
        legend={t('passwordLabel')}
        value={password.choice}
        onValueChange={(choice: 'keep' | 'new') => setPassword({ ...password, choice })}
      >
        <Radio.Item
          value="keep"
          label={
            <span className="flex flex-col">
              <span>{t(canKeep ? 'keepOwn' : 'makeOne')}</span>
              <span className="text-sm text-kumo-subtle">
                {t(canKeep ? 'keepOwnHint' : 'makeOneHint')}
              </span>
            </span>
          }
        />
        <Radio.Item
          value="new"
          label={
            <span className="flex flex-col">
              <span>{t('setNew')}</span>
              <span className="text-sm text-kumo-subtle">{t('setNewHint')}</span>
            </span>
          }
        />
      </Radio.Group>
      {password.choice === 'new' ? (
        <div className="flex flex-col gap-2 ps-7">
          <label
            id="admin-password-label"
            htmlFor="admin-password"
            className="text-sm font-medium text-kumo-strong"
          >
            {t('newPassword')}
          </label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <div className="min-w-0 flex-1">
              <PasswordInput
                key={made}
                ref={field}
                id="admin-password"
                aria-labelledby="admin-password-label"
                autoComplete="new-password"
                aria-describedby="admin-password-status"
                aria-invalid={rule !== null}
                defaultVisible={made > 0}
                showLabel={t('show')}
                hideLabel={t('hide')}
                value={password.value}
                onChange={(e) => setPassword({ choice: 'new', value: e.target.value })}
                className="font-mono pointer-coarse:h-9"
              />
            </div>
            <div className="flex gap-2">
              <Button
                className="flex-1 justify-center pointer-coarse:h-9"
                onClick={() => {
                  setPassword({ choice: 'new', value: generatePassword(username) });
                  setMade((n) => n + 1);
                }}
              >
                {t('generate')}
              </Button>
              <CopyButton
                text={password.value}
                select={() => field.current?.select()}
                disabled={!password.value}
                className="flex-1 justify-center pointer-coarse:h-9"
              />
            </div>
          </div>
          <p
            id="admin-password-status"
            aria-live="polite"
            className="flex items-start gap-1.5 text-sm text-kumo-subtle"
          >
            {rule ? (
              <WarningCircleIcon
                size={16}
                aria-hidden="true"
                className="mt-0.5 shrink-0 text-kumo-danger"
              />
            ) : password.value ? (
              <CheckCircleIcon
                size={16}
                aria-hidden="true"
                className="mt-0.5 shrink-0 text-kumo-success"
              />
            ) : null}
            <span className={rule ? 'text-kumo-default' : undefined}>{status}</span>
          </p>
        </div>
      ) : null}
    </div>
  );
}
