import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { ArrowLeftIcon } from '@phosphor-icons/react';
import { useCallback, useEffect, useRef, type FormEvent } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { FormField } from '../../components/form/FormField';
import { PasswordInput } from '../../components/form/PasswordInput';
import { useLocale, useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { formatCountdown } from '../../lib/format/date';
import { loginMessages } from './messages';
import { useCountdown } from './use-countdown';
import { useSignIn, type SignInError } from './use-sign-in';

const inputClass = 'h-11 text-base';

export function SignInForm() {
  const t = useT(loginMessages);
  const ts = useT(sharedMessages);
  const { locale } = useLocale();
  const s = useSignIn();
  const { clearLock } = s;
  const onUnlocked = useCallback(() => clearLock(), [clearLock]);
  const secondsLeft = useCountdown(s.lockedUntil, onUnlocked);
  const codeRef = useRef<HTMLInputElement>(null);
  const locked = s.lockedUntil !== null;

  useEffect(() => {
    if (s.step === 'totp') codeRef.current?.focus();
  }, [s.step]);

  const messages: Record<SignInError, string> = {
    required: t('errorRequired'),
    invalid: t('errorInvalid'),
    totpInvalid: t('errorTotpInvalid'),
    locked: t('errorLocked', { time: formatCountdown(secondsLeft, locale) }),
    offline: ts('errorOffline'),
    stale: ts('errorStale'),
    server: ts('errorServer'),
  };

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    void s.submit();
  }

  const totp = s.step === 'totp';
  return (
    <form onSubmit={onSubmit} noValidate className="flex flex-col gap-5">
      <div className="flex flex-col gap-1.5">
        <h1 className="text-title font-bold text-kumo-strong">
          {totp ? t('totpTitle') : t('title')}
        </h1>
        <p className="text-kumo-subtle">{totp ? t('totpHint') : t('subtitle')}</p>
      </div>

      {totp ? (
        <FormField id="code" label={t('totpCode')}>
          <Input
            id="code"
            aria-labelledby="code-label"
            ref={codeRef}
            name="code"
            dir="ltr"
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9]*"
            maxLength={6}
            value={s.fields.code}
            onChange={(e) => {
              const digits = e.target.value.replace(/\D/g, '').slice(0, 6);
              s.set('code', digits);
            }}
            className={`${inputClass} text-center font-mono tracking-[0.4em]`}
          />
        </FormField>
      ) : (
        <>
          <FormField id="username" label={t('username')}>
            <Input
              id="username"
              aria-labelledby="username-label"
              name="username"
              dir="ltr"
              autoComplete="username"
              autoCapitalize="off"
              spellCheck={false}
              value={s.fields.username}
              onChange={(e) => s.set('username', e.target.value)}
              className={inputClass}
            />
          </FormField>
          <FormField id="password" label={t('password')}>
            <PasswordInput
              id="password"
              aria-labelledby="password-label"
              name="password"
              autoComplete="current-password"
              showLabel={t('showPassword')}
              hideLabel={t('hidePassword')}
              value={s.fields.password}
              onChange={(e) => s.set('password', e.target.value)}
              className={inputClass}
            />
          </FormField>
        </>
      )}

      {s.error ? (
        <InlineAlert tone={s.error === 'locked' ? 'warning' : 'danger'}>
          {messages[s.error]}
        </InlineAlert>
      ) : null}

      <div className="flex flex-col gap-2">
        <Button
          type="submit"
          variant="primary"
          size="lg"
          loading={s.pending}
          disabled={locked || (totp && s.fields.code.length !== 6)}
          className="h-11 w-full justify-center"
        >
          {s.pending ? t('submitting') : totp ? t('totpSubmit') : t('submit')}
        </Button>
        {totp ? (
          <Button
            type="button"
            variant="ghost"
            onClick={s.back}
            icon={<ArrowLeftIcon size={16} className="rtl:-scale-x-100" />}
            className="h-11 w-full justify-center"
          >
            {t('totpBack')}
          </Button>
        ) : null}
      </div>
    </form>
  );
}
