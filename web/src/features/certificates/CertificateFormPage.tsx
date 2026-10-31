import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { SealCheckIcon } from '../../components/icons/regular';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import {
  certificateKeys,
  getCertificates,
  saveCertificate,
  type CertificateInput,
} from '../../lib/api/certificates';
import { domainKeys } from '../../lib/api/domains';
import { linkKeys } from '../../lib/api/rules';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { certFormOf } from './cert-form';
import { CertificateForm } from './CertificateForm';
import { certificatesMessages } from './messages';

/** Changes a stored certificate (`/domains/certificates/:id`); Let's Encrypt ones renew themselves. */
export function CertificateFormPage() {
  const t = useT(certificatesMessages);
  const { id } = useParams();
  const certId = Number(id);
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toasts = useKumoToastManager();
  const list = useQuery({ queryKey: certificateKeys.all, queryFn: getCertificates });
  const load = useFirstLoad(list.data);
  const found = load.data?.certificates.find((c) => c.id === certId);
  const cert = found?.origin === 'acme' ? undefined : found;
  usePageTitle(t('editTitle', { name: cert?.name ?? '' }));
  const back = () => void navigate('/domains');
  const save = useMutation({
    mutationFn: (input: CertificateInput) => saveCertificate(input, certId),
    onSuccess: () => {
      toasts.add({ title: t('saved') });
      // Domains and rules show what the certificate covers.
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void qc.invalidateQueries({ queryKey: linkKeys.all });
      back();
    },
  });
  let body;
  if (list.isError) {
    body = (
      <ErrorState
        message={t('loadFailed')}
        error={list.error}
        onRetry={() => void list.refetch()}
      />
    );
  } else if (!load.data) {
    body = (
      <div role="status" aria-label={t('editTitle', { name: '' })} className="flex flex-col gap-4">
        <Bone className="h-10 w-full max-w-[720px]" />
        <Bone className="h-40 w-full max-w-[720px]" />
      </div>
    );
  } else if (!cert) {
    body = (
      <EmptyState
        icon={<SealCheckIcon size={32} />}
        title={t('missing')}
        action={<Button onClick={back}>{t('title')}</Button>}
      />
    );
  } else {
    body = (
      <CertificateForm
        key={cert.id}
        initial={certFormOf(cert)}
        keepsKey={cert.kind === 'pasted' && cert.hasKey}
        saving={save.isPending}
        error={save.error}
        onSave={(input) => save.mutate(input)}
        onCancel={back}
      />
    );
  }
  return (
    <div className="mx-auto flex w-full max-w-form flex-col">
      <PageHeader
        eyebrow={
          <Link
            to="/domains"
            className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
          >
            {t('title')}
          </Link>
        }
        title={
          <>
            {t('editTitle', { name: '' })}
            {cert ? <bdi>{cert.name}</bdi> : null}
          </>
        }
        description={t('formDescription')}
      />
      <div className="p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
    </div>
  );
}
