import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { DotsThreeIcon, SealCheckIcon, TrashIcon } from '../../components/icons/regular';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useNavigate } from 'react-router';
import { DataTable, type Column } from '../../components/data/DataTable';
import { CardFacts, RowCard } from '../../components/data/RowCard';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { useT } from '../../i18n/locale';
import { certificateKeys } from '../../lib/api/certificates';
import { deleteDomain, domainKeys, type Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useLingering } from '../../lib/motion/use-lingering';
import { useFormat } from '../../lib/preferences/region';
import { useNow } from '../overview/use-now';
import { renewalOf } from './cert-view';
import { certificatesMessages } from './messages';
import { DomainLinesView, Mono, NameList, RuleName } from './parts';

/** The "..." menu: change the certificate, or delete while no rule holds the domain. */
function DomainMenu({ domain, onDelete }: { domain: Domain; onDelete: () => void }) {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  const label = t('moreFor', { name: domain.name });
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
        <DropdownMenu.Item
          className="gap-2"
          onClick={() => void navigate(`/domains/${String(domain.id)}/certificate`)}
        >
          <SealCheckIcon size={16} />
          {t('changeCert')}
        </DropdownMenu.Item>
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

/** The domain's certificate with its issuer and end date under it, or plain HTTP. */
function CertCell({ domain }: { domain: Domain }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const c = domain.certificate;
  if (!c) return <span className="text-kumo-subtle">{t('plainHttp')}</span>;
  const ends = c.facts.notAfter ? t('endsOn', { date: fmt.date(new Date(c.facts.notAfter)) }) : '';
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <bdi className="truncate text-kumo-default">{c.name}</bdi>
      <span className="text-sm text-kumo-subtle">
        {[c.facts.issuer, ends].filter(Boolean).join(', ')}
      </span>
    </span>
  );
}

/** A Let's Encrypt domain's running order or next renewal; nothing for the rest. */
function RenewalCell({ domain }: { domain: Domain }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const r = renewalOf(domain);
  if (!r) return <span className="text-kumo-subtle">{t('none')}</span>;
  if (r.kind === 'order') return <span>{t(`order_${r.state}`)}</span>;
  return <span>{t('renewsOn', { date: fmt.date(r.at) })}</span>;
}

const ruleNames = (d: Domain) =>
  d.rules.map((r) => ({ key: `${r.kind}:${String(r.id)}`, node: <RuleName rule={r} /> }));

/** Every domain rules can answer, with its certificate, renewal, rules and warnings (D50, D61). */
export function DomainsSection({ domains }: { domains: readonly Domain[] }) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const now = useNow();
  const errorText = useSubscriptionErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const titleId = useId();
  const [asking, setAsking] = useState<Domain | null>(null);
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
  const remove = useMutation({
    mutationFn: (d: Domain) => deleteDomain(d.id),
    onSuccess: () => {
      setAsking(null);
      toasts.add({ title: t('domainDeleted') });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
    },
  });
  const ask = (d: Domain) => {
    remove.reset();
    setAsking(d);
  };
  const columns: Column<Domain>[] = [
    {
      key: 'domain',
      header: t('colDomain'),
      cell: (d) => <Mono text={d.name} className="text-kumo-strong" />,
      className: 'max-w-56',
    },
    {
      key: 'cert',
      header: t('colCert'),
      cell: (d) => <CertCell domain={d} />,
      className: 'max-w-56',
    },
    { key: 'renewal', header: t('colRenewal'), cell: (d) => <RenewalCell domain={d} /> },
    {
      key: 'rules',
      header: t('colRules'),
      cell: (d) => <NameList names={ruleNames(d)} />,
      className: 'max-w-44',
    },
    {
      key: 'warnings',
      header: t('colWarnings'),
      cell: (d) => <DomainLinesView domain={d} now={now} />,
    },
    {
      key: 'menu',
      header: t('colMenu'),
      hideHeader: true,
      cell: (d) => <DomainMenu domain={d} onDelete={() => ask(d)} />,
      className: 'w-12 text-end',
    },
  ];
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('domainsTitle')}
        <span className="ms-2 font-normal text-kumo-subtle tabular">
          {fmt.number(domains.length)}
        </span>
      </h2>
      <DataTable
        caption={t('domainsTitle')}
        rows={domains}
        rowKey={(d) => d.id}
        columns={columns}
        card={(d) => (
          <RowCard
            lead={<Mono text={d.name} className="font-medium text-kumo-strong" />}
            menu={<DomainMenu domain={d} onDelete={() => ask(d)} />}
          >
            <CardFacts
              className="grid-cols-2"
              facts={[
                { label: t('colCert'), value: <CertCell domain={d} /> },
                { label: t('colRenewal'), value: <RenewalCell domain={d} /> },
                { label: t('colRules'), value: <NameList names={ruleNames(d)} /> },
              ]}
            />
            <DomainLinesView domain={d} now={now} />
          </RowCard>
        )}
      />
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteDomainTitle')}
        word={named?.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteDomainFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {t('deleteDomainBody')}
      </TypeToConfirm>
    </section>
  );
}
