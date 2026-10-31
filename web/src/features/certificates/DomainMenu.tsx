import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { CheckIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import {
  ArrowClockwiseIcon,
  DotsThreeIcon,
  LockSimpleIcon,
  SealCheckIcon,
  TrashIcon,
} from '../../components/icons/regular';
import { useSubToggle } from '../../components/menus/use-sub-toggle';
import { useT } from '../../i18n/locale';
import { acmeKeys, startAcme, type AcmeMethod } from '../../lib/api/acme';
import { domainKeys, type Domain } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { renewMethod } from './cert-view';
import {
  isIpName,
  karvanKeys,
  karvanUseOf,
  putKarvanDomain,
  type KarvanDomain,
} from '../../lib/api/karvan';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { acmeMethodsFor } from './domain-form';
import { karvanMessages } from './karvan-messages';
import { certificatesMessages } from './messages';

/**
 * Renew now as a submenu, like the account menu's Language, that opens under its row:
 * automatic or by DNS, the way it was last ordered checked. A wildcard renews only by DNS.
 */
function RenewSub({
  name,
  current,
  busy,
  onRenew,
}: {
  name: string;
  current: AcmeMethod;
  busy: boolean;
  onRenew: (m: AcmeMethod) => void;
}) {
  const t = useT(certificatesMessages);
  const { root, trigger } = useSubToggle();
  const allowed = acmeMethodsFor(name);
  const item = (m: AcmeMethod, label: string) => (
    <DropdownMenu.Item disabled={busy || !allowed.includes(m)} onClick={() => onRenew(m)}>
      <span className="flex w-full items-center gap-2">
        {label}
        <span className="ms-auto flex w-4 justify-center">
          {m === current ? <CheckIcon size={14} aria-hidden="true" /> : null}
        </span>
      </span>
    </DropdownMenu.Item>
  );
  return (
    <DropdownMenu.Sub {...root}>
      <DropdownMenu.SubTrigger
        className="gap-2 rtl:[&>svg:last-child]:-scale-x-100"
        disabled={busy}
        {...trigger}
      >
        <ArrowClockwiseIcon size={16} />
        {t('renewNow')}
      </DropdownMenu.SubTrigger>
      <DropdownMenu.SubContent
        side="bottom"
        align="start"
        sideOffset={4}
        className="w-(--anchor-width) min-w-0"
      >
        {item('http-01', t('renewHttp'))}
        {item('dns-01', t('renewDns'))}
      </DropdownMenu.SubContent>
    </DropdownMenu.Sub>
  );
}

/** Turns the domain on or off for Karvan at once, and says what the change does. */
function useKarvanToggle(domain: Domain, on: boolean) {
  const t = useT(karvanMessages);
  const toasts = useKumoToastManager();
  const errorText = useSubscriptionErrorText();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => putKarvanDomain({ domainId: on ? 0 : domain.id }),
    onSuccess: (next) => {
      qc.setQueryData(karvanKeys.all, next);
      const name = domain.name;
      toasts.add(
        on
          ? { title: t('karvanOff', { name }), description: t('karvanOffHint') }
          : {
              title: t('karvanOn', { name }),
              description: t(`karvanOn_${karvanUse(domain, next)}`),
            },
      );
    },
    onError: (e) =>
      toasts.add({ title: t('karvanFailed'), description: errorText(e), variant: 'error' }),
  });
}

/** How Karvan uses a domain it was just turned on for. */
const karvanUse = (d: Domain, k: KarvanDomain) => {
  const use = karvanUseOf(d, k);
  return use === 'none' ? 'own' : use;
};

/**
 * The "..." menu: change the certificate, renew a Let's Encrypt one now, use it for Karvan or stop,
 * or delete while no rule holds the domain and Karvan does not use it.
 */
export function DomainMenu({
  domain,
  onDelete,
  karvan,
}: {
  domain: Domain;
  onDelete: () => void;
  /** Left out while the Karvan domain is not known, or the server cannot change it. */
  karvan?: KarvanDomain;
}) {
  const t = useT(certificatesMessages);
  const tk = useT(karvanMessages);
  const navigate = useNavigate();
  const toasts = useKumoToastManager();
  const acmeError = useAcmeErrorText();
  const qc = useQueryClient();
  const used = karvan?.domainId === domain.id;
  const toggle = useKarvanToggle(domain, used);
  const label = t('moreFor', { name: domain.name });
  const page = `/domains/${String(domain.id)}/certificate`;
  const method = renewMethod(domain);
  // The order runs on the domain's own page, which follows it to its end.
  const renew = useMutation({
    mutationFn: (m: NonNullable<typeof method>) => startAcme(domain.id, m),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: acmeKeys.domain(domain.id) });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void navigate(page);
    },
    onError: (e) =>
      toasts.add({ title: t('renewFailed'), description: acmeError(e), variant: 'error' }),
  });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} />}
            className="shrink-0 pointer-coarse:size-9"
          />
        }
      />
      <DropdownMenu.Content align="end" className="max-w-[min(20rem,calc(100vw-2rem))] min-w-44">
        <DropdownMenu.Item className="gap-2" onClick={() => void navigate(page)}>
          <SealCheckIcon size={16} />
          {t('changeCert')}
        </DropdownMenu.Item>
        {method ? (
          <RenewSub
            name={domain.name}
            current={method}
            busy={renew.isPending}
            onRenew={(m) => renew.mutate(m)}
          />
        ) : null}
        {karvan && !isIpName(domain.name) ? (
          <DropdownMenu.Item
            className="gap-2"
            disabled={toggle.isPending}
            onClick={() => toggle.mutate()}
          >
            <LockSimpleIcon size={16} />
            {tk(used ? 'stopKarvan' : 'useKarvan')}
          </DropdownMenu.Item>
        ) : null}
        <DropdownMenu.Item
          className="gap-2"
          variant="danger"
          disabled={domain.rules.length > 0 || used}
          onClick={onDelete}
        >
          <TrashIcon size={16} />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
