import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { SealCheckIcon } from '../../components/icons/regular';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { acmeKeys, getAcmeSettings, isRunning } from '../../lib/api/acme';
import { certificateKeys, getCertificates } from '../../lib/api/certificates';
import {
  addDomain,
  domainKeys,
  getDomains,
  setDomainCertificate,
  type CertChoice,
} from '../../lib/api/domains';
import { linkKeys } from '../../lib/api/rules';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { DomainForm } from './DomainForm';
import { newDomainForm } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { certificatesMessages } from './messages';
import { OrderProgress } from './OrderProgress';

/** Add domain (`/domains/new`) or Change certificate (`/domains/:id/certificate`). */
export function DomainFormPage() {
  const t = useT(domainFormMessages);
  const tc = useT(certificatesMessages);
  const { id } = useParams();
  const [query] = useSearchParams();
  const domainId = id ? Number(id) : undefined;
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains });
  const certs = useQuery({ queryKey: certificateKeys.all, queryFn: getCertificates });
  const settings = useQuery({ queryKey: acmeKeys.settings, queryFn: getAcmeSettings });
  // The domain whose order this page follows, once an order started here.
  const [ordered, setOrdered] = useState<number>();
  const all =
    domains.data && certs.data && settings.data
      ? { domains: domains.data.domains, certificates: certs.data.certificates, s: settings.data }
      : undefined;
  const load = useFirstLoad(all);
  const domain = load.data?.domains.find((d) => d.id === domainId);
  usePageTitle(domain ? t('changeTitle', { name: domain.name }) : t('newTitle'));
  const back = () => void navigate('/domains');
  const save = useMutation({
    mutationFn: ({ name, choice }: { name: string; choice: CertChoice }) =>
      domainId ? setDomainCertificate(domainId, choice) : addDomain(name, choice),
    onSuccess: (d, { choice }) => {
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
      void qc.invalidateQueries({ queryKey: linkKeys.all });
      if (choice.kind === 'acme') {
        void qc.invalidateQueries({ queryKey: acmeKeys.domain(d.id) });
        setOrdered(d.id);
        // A reload keeps following the order on the domain's own page.
        if (!domainId) void navigate(`/domains/${String(d.id)}/certificate`, { replace: true });
        return;
      }
      toasts.add({ title: domainId ? t('saved') : t('added') });
      back();
    },
  });
  const failed = [domains, certs, settings].find((q) => q.isError);
  const running = isRunning(domain?.acme?.order);
  // An order running when the page opens stays on the page once it ends, so
  // a failed one shows its words and Try again instead of the empty form.
  if (ordered === undefined && domain && running) setOrdered(domain.id);
  let body;
  if (failed) {
    body = (
      <ErrorState
        message={tc('loadFailed')}
        error={failed.error}
        onRetry={() => [domains, certs, settings].forEach((q) => void q.refetch())}
      />
    );
  } else if (!load.data) {
    body = (
      <div role="status" aria-label={t('newTitle')} className="flex flex-col gap-4">
        <Bone className="h-10 w-full max-w-[720px]" />
        <Bone className="h-72 w-full max-w-[720px]" />
      </div>
    );
  } else if (ordered !== undefined) {
    body = <OrderProgress domainId={ordered} />;
  } else if (domainId && !domain) {
    body = (
      <EmptyState
        icon={<SealCheckIcon size={32} />}
        title={t('missing')}
        action={<Button onClick={back}>{tc('title')}</Button>}
      />
    );
  } else if (domain && running) {
    body = <OrderProgress domainId={domain.id} />;
  } else {
    body = (
      <DomainForm
        key={domain?.id ?? 'new'}
        initial={newDomainForm(domain?.name ?? query.get('name') ?? '')}
        isNew={!domain}
        certificates={load.data.certificates}
        except={domain?.certificate?.id}
        needsAgreement={!load.data.s.hasAccount}
        saving={save.isPending}
        error={save.error}
        onSave={(name, choice) => save.mutate({ name, choice })}
        onReset={() => save.reset()}
        onCancel={back}
      />
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/domains"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
          >
            {tc('title')}
          </Link>
        }
        title={
          domain ? (
            <>
              {t('changeTitle', { name: '' })}
              <bdi dir="ltr">{domain.name}</bdi>
            </>
          ) : (
            t('newTitle')
          )
        }
        description={domain ? t('changeDescription') : t('newDescription')}
      />
      <div className="p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
    </div>
  );
}
