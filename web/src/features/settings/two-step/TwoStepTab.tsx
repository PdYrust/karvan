import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { StatusBadge } from '../../../components/feedback/StatusBadge';
import { SettingRow, SettingsCard } from '../../../components/layout/SettingRow';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import { beginTotp, disableTotp, type TotpSetup } from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { sessionKey, useSession } from '../../session/use-session';
import { ReauthDialog } from '../security/ReauthDialog';
import { twoStepMessages } from './messages';
import { loadQr, TotpSetupCard } from './TotpSetupCard';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

/** Two-step sign-in: set up with a QR code and a first code, or turn off with the password and a code. */
export function TwoStepTab() {
  const t = useT(twoStepMessages);
  const errorText = useErrorText();
  const session = useSession();
  const queryClient = useQueryClient();
  const toasts = useKumoToastManager();
  // The secret is a one-time value (D10): component state only, gone on unmount.
  const [setup, setSetup] = useState<TotpSetup | null>(null);
  const [turningOff, setTurningOff] = useState(false);
  const begin = useMutation({ mutationFn: beginTotp, gcTime: 0, onSuccess: setSetup });
  const disable = useMutation({
    mutationFn: disableTotp,
    gcTime: 0,
    onSuccess: () => {
      setTurningOff(false);
      toasts.add({ title: t('turnedOff') });
      void queryClient.invalidateQueries({ queryKey: sessionKey });
    },
  });

  const load = useFirstLoad(session.data);
  if (session.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('title')}>
        <Bone className="h-28 w-full" />
      </div>
    );
  }
  if (session.isError) {
    return (
      <ErrorState
        message={t('loadFailed')}
        error={session.error}
        onRetry={() => void session.refetch()}
      />
    );
  }
  const on = session.data.operator?.totpEnabled ?? false;

  return (
    <Reveal on={load.reveal}>
      <SettingsCard>
        <SettingRow
          title={t('title')}
          badge={
            <StatusBadge tone={on ? 'success' : 'neutral'}>{t(on ? 'on' : 'off')}</StatusBadge>
          }
          description={t(on ? 'hintOn' : 'hintOff')}
          control={
            on ? (
              <Button className="pointer-coarse:h-11" onClick={() => setTurningOff(true)}>
                {t('turnOff')}
              </Button>
            ) : setup ? null : (
              <Button
                variant="primary"
                className="pointer-coarse:h-11"
                loading={begin.isPending}
                onClick={() => {
                  void loadQr();
                  begin.mutate();
                }}
              >
                {t('setUp')}
              </Button>
            )
          }
        >
          {on ? (
            <p className="text-sm text-kumo-subtle">
              <CodeText text={t('lostApp')} />
            </p>
          ) : null}
        </SettingRow>
        {setup && !on ? (
          <TotpSetupCard
            setup={setup}
            onDone={() => {
              setSetup(null);
              toasts.add({ title: t('turnedOn') });
              void queryClient.invalidateQueries({ queryKey: sessionKey });
            }}
            onCancel={() => setSetup(null)}
          />
        ) : null}
      </SettingsCard>
      {begin.isError ? <InlineAlert>{errorText(begin.error)}</InlineAlert> : null}
      <ReauthDialog
        open={turningOff}
        onOpenChange={(open) => {
          setTurningOff(open);
          if (!open) disable.reset();
        }}
        title={t('turnOffTitle')}
        description={t('turnOffBody')}
        action={t('turnOff')}
        danger
        needsCode
        pending={disable.isPending}
        error={disable.error}
        onSubmit={(r) => disable.mutate(r)}
      />
    </Reveal>
  );
}
