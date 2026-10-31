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
import { copyMessages } from './copy-messages';
import { acmeMethodsFor } from './domain-form';
import { certificatesMessages } from './messages';
import { panelLink, type PanelUse } from '../https/panel-use';

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

/**
 * Use or Stop using the domain for the Karvan panel, on the HTTPS tab where the change is made and
 * confirmed; without a certificate for the domain, why not.
 */
function PanelItem({ domain, use }: { domain: Domain; use: PanelUse }) {
  const t = useT(copyMessages);
  const navigate = useNavigate();
  const blocked = use.kind === 'use' ? use.blocked : undefined;
  return (
    <DropdownMenu.Item
      className="gap-2"
      disabled={blocked !== undefined}
      onClick={() => void navigate(panelLink(domain, use.kind))}
    >
      <LockSimpleIcon size={16} />
      <span className="flex min-w-0 flex-col">
        <span>{t(use.kind === 'use' ? 'usePanel' : 'stopPanel')}</span>
        {blocked ? (
          <span className="text-xs whitespace-normal text-kumo-subtle">
            {t(blocked === 'no_certificate' ? 'needsCert' : 'notCovered')}
          </span>
        ) : null}
      </span>
    </DropdownMenu.Item>
  );
}

/**
 * The "..." menu: change the certificate, renew a Let's Encrypt one now, use it for the Karvan
 * panel, or delete while no rule holds the domain.
 */
export function DomainMenu({
  domain,
  onDelete,
  panel,
}: {
  domain: Domain;
  onDelete: () => void;
  /** Left out when the panel's HTTPS cannot be changed from here. */
  panel?: PanelUse;
}) {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  const toasts = useKumoToastManager();
  const acmeError = useAcmeErrorText();
  const qc = useQueryClient();
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
        {panel ? <PanelItem domain={domain} use={panel} /> : null}
        <DropdownMenu.Item
          className="gap-2"
          variant="danger"
          disabled={domain.rules.length > 0}
          onClick={onDelete}
        >
          <TrashIcon size={16} />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
