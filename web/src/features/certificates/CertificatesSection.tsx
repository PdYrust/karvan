import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { useT } from '../../i18n/locale';
import { certificateKeys, deleteCertificate, type Certificate } from '../../lib/api/certificates';
import { domainKeys, type Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useLingering } from '../../lib/motion/use-lingering';
import { useFormat } from '../../lib/preferences/region';
import { useNow } from '../overview/use-now';
import { CertificateCard } from './CertificateCard';
import { certificatesMessages } from './messages';

/** The stored certificates, one card each, and the typed delete of one no rule uses. */
export function CertificatesSection({
  certificates,
  domains,
}: {
  certificates: readonly Certificate[];
  domains: readonly Domain[];
}) {
  const t = useT(certificatesMessages);
  const fmt = useFormat();
  const now = useNow();
  const errorText = useSubscriptionErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const titleId = useId();
  const [asking, setAsking] = useState<Certificate | null>(null);
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
  const remove = useMutation({
    mutationFn: (c: Certificate) => deleteCertificate(c.id),
    onSuccess: () => {
      setAsking(null);
      toasts.add({ title: t('certDeleted') });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
    },
  });
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <h2 id={titleId} className="text-heading font-semibold text-kumo-strong">
        {t('certsTitle')}
        <span className="ms-2 font-normal text-kumo-subtle tabular">
          {fmt.number(certificates.length)}
        </span>
      </h2>
      {certificates.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('certsEmpty')}</p>
      ) : (
        <div className="flex flex-col gap-4">
          {certificates.map((c) => (
            <CertificateCard
              key={c.id}
              cert={c}
              domains={domains}
              now={now}
              onDelete={() => {
                remove.reset();
                setAsking(c);
              }}
            />
          ))}
        </div>
      )}
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteCertTitle')}
        word={named?.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteCertFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {t('deleteCertBody')}
      </TypeToConfirm>
    </section>
  );
}
