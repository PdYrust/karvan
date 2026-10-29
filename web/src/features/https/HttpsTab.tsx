import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { ErrorState } from '../../components/feedback/ErrorState';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { SettingRow, SettingsCard } from '../../components/layout/SettingRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { accountKeys, confirmAddress, type Reauth } from '../../lib/api/account';
import { domainKeys, getDomains } from '../../lib/api/domains';
import { getHttps, httpsKeys, startHttps, type Https } from '../../lib/api/https';
import { useErrorText } from '../../lib/errors/errors';
import { httpsErrorMessages, httpsErrorText } from '../../lib/errors/https';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { ReauthDialog } from '../settings/security/ReauthDialog';
import { CopyValue } from '../settings/security/CopyValue';
import { accessMessages } from '../settings/access/messages';
import { useSecondsLeft } from '../settings/access/use-seconds-left';
import { WaitingCard } from '../settings/access/WaitingCard';
import { certifiedDomains, formOf, httpsLink, httpsStart, type HttpsForm } from './https-form';
import { HttpsFields } from './HttpsFields';
import { httpsMessages } from './messages';

/** The current mode as a sentence: HTTP, self-signed, or a domain's certificate, and the port. */
function useModeText() {
  const t = useT(httpsMessages);
  return (h: Https, domain: string) => {
    const port = `\`${String(h.port)}\``;
    if (h.tls === 'off') return t('modeOff', { port });
    if (h.tls === 'self') return t('modeSelf', { port });
    return t(h.shared ? 'modeShared' : 'modeDomain', { port, domain: `\`${domain}\`` });
  };
}

/** How the panel is served, changed through a new address that waits 10 minutes for a confirm (D7, D9). */
export function HttpsTab() {
  const t = useT(httpsMessages);
  const ta = useT(accessMessages);
  const te = useT(httpsErrorMessages);
  const shared = useErrorText();
  const modeText = useModeText();
  const toasts = useKumoToastManager();
  const queryClient = useQueryClient();
  const https = useQuery({ queryKey: httpsKeys.all, queryFn: getHttps });
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains });
  const [form, setForm] = useState<HttpsForm | null>(null);
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: httpsKeys.all });
    void queryClient.invalidateQueries({ queryKey: accountKeys.address });
  };
  const start = useMutation({
    mutationFn: ({
      f,
      reauth,
    }: {
      f: NonNullable<ReturnType<typeof httpsStart>>;
      reauth: Reauth;
    }) => startHttps(f, reauth),
    onSuccess: () => {
      setForm(null);
      refresh();
    },
  });
  const confirm = useMutation({
    mutationFn: confirmAddress,
    onSuccess: () => {
      toasts.add({ title: ta('confirmed') });
      refresh();
    },
  });
  const pending = https.data?.pending ?? null;
  const left = useSecondsLeft(pending?.expiresAt);
  const expired = left === 0;
  useEffect(() => {
    if (expired) void queryClient.invalidateQueries({ queryKey: httpsKeys.all });
  }, [expired, queryClient]);
  const load = useFirstLoad(https.data);
  if (https.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('title')}>
        <Bone className="h-44 w-full" />
      </div>
    );
  }
  if (https.isError) {
    return (
      <ErrorState
        message={t('loadFailed')}
        error={https.error}
        onRetry={() => void https.refetch()}
      />
    );
  }
  const h = https.data;
  const all = domains.data?.domains ?? [];
  const usable = certifiedDomains(all);
  const name = all.find((d) => d.id === h.domainId)?.name ?? '';
  const change = form ? httpsStart(form, h) : null;
  const open = () => {
    start.reset();
    setForm(formOf(h));
  };
  return (
    <Reveal on={load.reveal}>
      {pending && !expired && left !== null ? (
        <WaitingCard
          pending={pending}
          link={httpsLink(pending, all, document.baseURI)}
          signInNote={t('signInThere')}
          secondsLeft={left}
          confirming={confirm.isPending}
          confirmError={confirm.isError ? shared(confirm.error) : null}
          onConfirm={() => confirm.mutate()}
        />
      ) : null}
      {h.tls === 'off' ? <InlineAlert>{t('plainWarning')}</InlineAlert> : null}
      <SettingsCard title={t('title')}>
        <SettingRow
          title={t('modeTitle')}
          description={<CodeText text={modeText(h, name)} />}
          control={
            <Button className="pointer-coarse:h-9" onClick={open}>
              {t('change')}
            </Button>
          }
        />
        {h.fingerprint ? (
          <SettingRow title={t('fingerprintTitle')} description={t('fingerprintHint')}>
            <CopyValue label={t('fingerprint')} value={h.fingerprint} />
          </SettingRow>
        ) : null}
      </SettingsCard>
      <ReauthDialog
        open={form !== null}
        onOpenChange={(o) => (o ? open() : setForm(null))}
        title={t('dialogTitle')}
        description={t('dialogBody')}
        action={t('action')}
        ready={change !== null}
        pending={start.isPending}
        error={start.error}
        describe={(err) => httpsErrorText(te, shared, err)}
        onSubmit={(reauth) => change && start.mutate({ f: change, reauth })}
      >
        {form ? (
          <HttpsFields
            form={form}
            https={h}
            domains={usable}
            disabled={start.isPending}
            onChange={setForm}
          />
        ) : null}
      </ReauthDialog>
    </Reveal>
  );
}
