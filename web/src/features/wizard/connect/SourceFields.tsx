import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { Select } from '@cloudflare/kumo/components/select';
import { useMemo } from 'react';
import { FormField } from '../../../components/form/FormField';
import { PasswordInput } from '../../../components/form/PasswordInput';
import { panelNames } from '../../../components/panels/PanelLogo';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { ConnectValues, SignIn } from './form';
import { connectMessages } from './messages';

const inputClass = 'h-10 pointer-coarse:h-9';

/** The old panel: address, kind and sign-in. */
export function SourceFields({
  values,
  set,
  disabled,
}: {
  values: ConnectValues;
  set: (patch: Partial<ConnectValues>) => void;
  disabled?: boolean;
}) {
  const t = useT(connectMessages);
  const kinds = useMemo(
    () => ({ auto: t('kindAuto'), marzban: panelNames.marzban, pasarguard: panelNames.pasarguard }),
    [t],
  );
  const byKey = values.signIn === 'apiKey';
  return (
    <fieldset disabled={disabled} className="flex flex-col gap-5">
      <legend className="mb-1 flex flex-col gap-1">
        <span className="font-medium text-kumo-strong">{t('sourceTitle')}</span>
        <span className="text-sm text-kumo-subtle">{t('sourceHint')}</span>
      </legend>
      <FormField
        id="source-url"
        label={t('addressLabel')}
        hint={<CodeText text={t('addressHint')} />}
      >
        <Input
          id="source-url"
          aria-labelledby="source-url-label"
          aria-describedby="source-url-hint"
          name="source-url"
          type="url"
          dir="ltr"
          inputMode="url"
          autoComplete="url"
          autoCapitalize="off"
          spellCheck={false}
          value={values.url}
          onChange={(e) => set({ url: e.target.value })}
          className={inputClass}
        />
      </FormField>
      {byKey ? null : (
        <FormField id="source-kind" label={t('kindLabel')}>
          <Select
            id="source-kind"
            aria-labelledby="source-kind-label"
            className={`w-full sm:w-64 ${inputClass}`}
            value={values.kind}
            onValueChange={(v) => v && set({ kind: v as ConnectValues['kind'] })}
            items={kinds}
          />
        </FormField>
      )}
      <Radio.Group
        legend={t('signInLabel')}
        orientation="horizontal"
        value={values.signIn}
        onValueChange={(v: SignIn) => set({ signIn: v })}
      >
        <Radio.Item value="password" label={t('signInPassword')} />
        <Radio.Item value="apiKey" label={t('signInKey', { name: panelNames.pasarguard })} />
      </Radio.Group>
      {byKey ? (
        <FormField id="source-key" label={t('apiKeyLabel')} hint={t('signInHint')}>
          <PasswordInput
            id="source-key"
            aria-labelledby="source-key-label"
            aria-describedby="source-key-hint"
            name="source-key"
            autoComplete="off"
            showLabel={t('showKey')}
            hideLabel={t('hideKey')}
            value={values.apiKey}
            onChange={(e) => set({ apiKey: e.target.value })}
            className={inputClass}
          />
        </FormField>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2">
          <FormField id="source-user" label={t('usernameLabel')}>
            <Input
              id="source-user"
              aria-labelledby="source-user-label"
              name="source-user"
              dir="ltr"
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
              value={values.username}
              onChange={(e) => set({ username: e.target.value })}
              className={inputClass}
            />
          </FormField>
          <FormField id="source-pass" label={t('passwordLabel')}>
            <PasswordInput
              id="source-pass"
              aria-labelledby="source-pass-label"
              aria-describedby="source-sign-in-hint"
              name="source-pass"
              autoComplete="off"
              showLabel={t('showPassword')}
              hideLabel={t('hidePassword')}
              value={values.password}
              onChange={(e) => set({ password: e.target.value })}
              className={inputClass}
            />
          </FormField>
          <p id="source-sign-in-hint" className="text-sm text-kumo-subtle sm:col-span-2">
            {t('signInHint')}
          </p>
        </div>
      )}
    </fieldset>
  );
}
