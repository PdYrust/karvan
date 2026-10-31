import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { SettingRow, SettingsCard } from '../../../components/layout/SettingRow';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import {
  accountKeys,
  confirmAddress,
  getAddress,
  keepPort,
  startAddress,
  type AddressStart,
  type NewAddress,
  type Reauth,
} from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { ReauthDialog } from '../security/ReauthDialog';
import { accessMessages } from './messages';
import { chosenPort, type PortChoice } from './port';
import { PortFields } from './PortFields';
import { useSecondsLeft } from './use-seconds-left';
import { waitingLink } from './waiting';
import { WaitingCard } from './WaitingCard';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

type Asking = 'path' | 'port' | null;

/** The hidden path and the port, each changed through a new address that waits 10 minutes for a confirm. */
export function AddressCard() {
  const t = useT(accessMessages);
  const errorText = useErrorText();
  const toasts = useKumoToastManager();
  const queryClient = useQueryClient();
  const address = useQuery({ queryKey: accountKeys.address, queryFn: getAddress });
  const [asking, setAsking] = useState<Asking>(null);
  const [port, setPort] = useState<PortChoice>({ kind: 'random' });
  // The new path is shown once: this page's start answer, in state only.
  const [started, setStarted] = useState<NewAddress | null>(null);
  const refresh = () => queryClient.invalidateQueries({ queryKey: accountKeys.address });
  const start = useMutation({
    mutationFn: ({ change, reauth }: { change: AddressStart; reauth: Reauth }) =>
      startAddress(change, reauth),
    gcTime: 0,
    onSuccess: (to) => {
      setStarted(to);
      setAsking(null);
      void refresh();
    },
  });
  const confirm = useMutation({
    mutationFn: confirmAddress,
    onSuccess: () => {
      toasts.add({ title: t('confirmed') });
      void refresh();
    },
  });
  const pending = address.data?.pending ?? null;
  const left = useSecondsLeft(pending?.expiresAt);
  const expired = left === 0;
  useEffect(() => {
    if (expired) void queryClient.invalidateQueries({ queryKey: accountKeys.address });
  }, [expired, queryClient]);

  const load = useFirstLoad(address.data);
  if (address.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('addressTitle')}>
        <Bone className="h-44 w-full" />
      </div>
    );
  }
  if (address.isError) {
    return (
      <ErrorState
        message={t('loadFailed')}
        error={address.error}
        onRetry={() => void address.refetch()}
      />
    );
  }
  const current = address.data.port;
  const portValue = chosenPort(port, current);
  function open(next: Asking) {
    start.reset();
    setPort({ kind: 'random' });
    setAsking(next);
  }

  return (
    <Reveal on={load.reveal}>
      {pending && !expired && left !== null ? (
        <WaitingCard
          pending={pending}
          link={waitingLink(pending, started, document.baseURI)}
          secondsLeft={left}
          confirming={confirm.isPending}
          confirmError={confirm.isError ? errorText(confirm.error) : null}
          onConfirm={() => confirm.mutate()}
        />
      ) : null}
      <SettingsCard title={t('addressTitle')}>
        <SettingRow
          title={t('pathTitle')}
          description={t('pathHint')}
          control={
            <Button className="pointer-coarse:h-9" onClick={() => open('path')}>
              {t('newPath')}
            </Button>
          }
        />
        <SettingRow
          title={t('portTitle')}
          description={<CodeText text={t('portHint', { port: `\`${current}\`` })} />}
          control={
            <Button className="pointer-coarse:h-9" onClick={() => open('port')}>
              {t('changePort')}
            </Button>
          }
        />
      </SettingsCard>
      <ReauthDialog
        open={asking === 'path'}
        onOpenChange={(o) => setAsking(o ? 'path' : null)}
        title={t('pathDialogTitle')}
        description={t('pathDialogBody')}
        action={t('makePath')}
        pending={start.isPending}
        error={start.error}
        onSubmit={(reauth) => start.mutate({ change: { newPath: true, port: keepPort }, reauth })}
      />
      <ReauthDialog
        open={asking === 'port'}
        onOpenChange={(o) => setAsking(o ? 'port' : null)}
        title={t('portDialogTitle')}
        description={t('portDialogBody')}
        action={t('openPort')}
        ready={portValue !== null}
        pending={start.isPending}
        error={start.error}
        onSubmit={(reauth) =>
          portValue !== null &&
          start.mutate({ change: { newPath: false, port: portValue }, reauth })
        }
      >
        <PortFields choice={port} current={current} disabled={start.isPending} onChange={setPort} />
      </ReauthDialog>
    </Reveal>
  );
}
