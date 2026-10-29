import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { useT } from '../../i18n/locale';
import { certificateKeys } from '../../lib/api/certificates';
import { deleteDomain, domainKeys, type Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useLingering } from '../../lib/motion/use-lingering';
import { useFormat } from '../../lib/preferences/region';
import { useNow } from '../overview/use-now';
import { DomainCard } from './DomainCard';
import { certificatesMessages } from './messages';

/** Every domain rules can answer, one short card each; problems open from the state badge's menu (D50, D61). */
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
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('domainsTitle')}
        <span className="ms-2 font-normal text-kumo-subtle tabular">
          {fmt.number(domains.length)}
        </span>
      </h2>
      <div className="flex flex-col gap-4">
        {domains.map((d) => (
          <DomainCard key={d.id} domain={d} now={now} onDelete={() => ask(d)} />
        ))}
      </div>
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
