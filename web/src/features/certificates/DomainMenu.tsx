import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { CheckIcon } from '@phosphor-icons/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import {
  ArrowClockwiseIcon,
  DotsThreeIcon,
  SealCheckIcon,
  TrashIcon,
} from '../../components/icons/regular';
import { useSubToggle } from '../../components/menus/use-sub-toggle';
import { useT } from '../../i18n/locale';
import { acmeKeys, startAcme, type AcmeMethod } from '../../lib/api/acme';
import { domainKeys, type Domain } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { renewMethod } from './cert-view';
import { acmeMethodsFor } from './domain-form';
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

/**
 * The "..." menu: change the certificate, renew a Let's Encrypt one now, or delete while no rule
 * holds the domain.
 */
export function DomainMenu({ domain, onDelete }: { domain: Domain; onDelete: () => void }) {
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
      <DropdownMenu.Content align="end" className="min-w-44">
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
