import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { StatusIcon, StatusLabel } from '../../components/feedback/StatusBadge';
import { Section } from '../../components/layout/Section';
import { useT } from '../../i18n/locale';
import {
  acmeKeys,
  cancelAcme,
  getAcme,
  isRunning,
  startAcme,
  type AcmeMethod,
  type AcmeOrder,
} from '../../lib/api/acme';
import { certificateKeys } from '../../lib/api/certificates';
import { domainKeys } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { DnsStep } from './DnsStep';
import { orderPoll, orderSteps } from './order-view';
import { domainFormMessages } from './domain-messages';
import { isPortError } from './domain-form';
import { useAcmeErrorWords } from './parts';

function Steps({ order }: { order: AcmeOrder }) {
  const t = useT(domainFormMessages);
  return (
    <ol className="flex flex-col gap-2 sm:flex-row sm:gap-6">
      {orderSteps(order).map((s) => (
        <li
          key={s.id}
          aria-current={s.state === 'current' ? 'step' : undefined}
          className={
            s.state === 'current'
              ? 'flex items-center gap-2 font-medium text-(--color-kumo-brand)'
              : s.state === 'done'
                ? 'flex items-center gap-2 text-kumo-default'
                : 'flex items-center gap-2 text-kumo-subtle'
          }
        >
          <StatusIcon
            tone={s.state === 'done' ? 'success' : s.state === 'current' ? 'active' : 'neutral'}
            size={18}
          />
          {t(`step_${s.id}`)}
        </li>
      ))}
    </ol>
  );
}

/**
 * A domain's Let's Encrypt order (D69): its steps, read every 2 seconds while
 * it runs, the TXT record of a manual order, and its end.
 */
export function OrderProgress({ domainId }: { domainId: number }) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const words = useAcmeErrorWords();
  const navigate = useNavigate();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const acme = useQuery({
    queryKey: acmeKeys.domain(domainId),
    queryFn: () => getAcme(domainId),
    refetchInterval: (q) => orderPoll(q.state.data?.order),
  });
  const order = acme.data?.order ?? null;
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: acmeKeys.domain(domainId) });
    void qc.invalidateQueries({ queryKey: domainKeys.all });
    void qc.invalidateQueries({ queryKey: certificateKeys.all });
  };
  const again = useMutation({
    mutationFn: (m: AcmeMethod) => startAcme(domainId, m),
    onSuccess: refresh,
  });
  const cancel = useMutation({
    mutationFn: () => cancelAcme(domainId),
    onSuccess: () => {
      toasts.add({ title: t('orderCancelled') });
      refresh();
    },
  });
  const back = () => void navigate('/domains');
  let body;
  if (acme.isError) {
    body = <InlineAlert>{`${t('orderLoadFailed')} ${errorText(acme.error)}`}</InlineAlert>;
  } else if (!order) {
    body = (
      <div role="status" aria-label={t('orderTitle')} className="flex flex-col gap-3">
        <Bone className="h-6 w-full" />
        <Bone className="h-6 w-2/3" />
      </div>
    );
  } else {
    const running = isRunning(order);
    body = (
      <>
        <Steps order={order} />
        {order.state === 'waiting_dns' && order.dns ? (
          <DnsStep domainId={domainId} dns={order.dns} onChecked={refresh} />
        ) : null}
        {order.state === 'done' ? <StatusLabel tone="success">{t('orderDone')}</StatusLabel> : null}
        {order.state === 'failed' || order.state === 'cancelled' ? (
          <InlineAlert>
            {order.error ? `${t('orderFailed')} ${words(order.error)}` : t('orderFailed')}
          </InlineAlert>
        ) : null}
        {again.isError ? <InlineAlert>{errorText(again.error)}</InlineAlert> : null}
        {cancel.isError ? <InlineAlert>{errorText(cancel.error)}</InlineAlert> : null}
        <div className="flex flex-wrap justify-end gap-2">
          {running ? (
            <Button
              variant="ghost"
              loading={cancel.isPending}
              className="pointer-coarse:h-9"
              onClick={() => cancel.mutate()}
            >
              {t('cancelOrder')}
            </Button>
          ) : null}
          {!running && order.state !== 'done' ? (
            <>
              {isPortError(again.error) || order.error?.code.startsWith('acme_port_') ? (
                <Button className="pointer-coarse:h-9" onClick={() => again.mutate('dns-01')}>
                  {t('useDns')}
                </Button>
              ) : null}
              <Button
                loading={again.isPending}
                className="pointer-coarse:h-9"
                onClick={() => again.mutate(order.method)}
              >
                {t('tryAgain')}
              </Button>
            </>
          ) : null}
          <Button
            variant={order.state === 'done' ? 'primary' : 'secondary'}
            className="pointer-coarse:h-9"
            onClick={back}
          >
            {t('backToDomains')}
          </Button>
        </div>
      </>
    );
  }
  return (
    <Section title={t('orderTitle')} className="max-w-[720px]">
      <div className="flex flex-col gap-5 px-4 pt-2 pb-4" aria-live="polite">
        {body}
      </div>
    </Section>
  );
}
