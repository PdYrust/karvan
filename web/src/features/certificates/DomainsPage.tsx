import { Button } from '@cloudflare/kumo/components/button';
import { PlusIcon, SlidersHorizontalIcon } from '@phosphor-icons/react';
import { SealCheckIcon } from '../../components/icons/regular';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ToolbarAdd } from '../../components/data/ToolbarAdd';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { ListSkeleton } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { certificateKeys, getCertificates } from '../../lib/api/certificates';
import { domainKeys, getDomains } from '../../lib/api/domains';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { AcmeSettingsSheet } from './AcmeSettingsSheet';
import { CertificatesSection } from './CertificatesSection';
import { DomainsSection } from './DomainsSection';
import { certificatesMessages } from './messages';

/** The list toolbar's actions: Let's Encrypt settings, then add, as on the Migrations page. */
function ToolbarActions({ onSettings }: { onSettings: () => void }) {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  return (
    <>
      <Button
        shape="square"
        aria-label={t('acmeSettings')}
        title={t('acmeSettings')}
        icon={<SlidersHorizontalIcon size={16} />}
        className="pointer-coarse:size-9"
        onClick={onSettings}
      />
      <ToolbarAdd label={t('addDomain')} onClick={() => void navigate('/domains/new')} />
    </>
  );
}
function AddButton() {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  return (
    <Button
      variant="primary"
      icon={<PlusIcon size={16} />}
      className="pointer-coarse:h-9"
      onClick={() => void navigate('/domains/new')}
    >
      {t('addDomain')}
    </Button>
  );
}

/** Domains and certificates (D46, D61): the domains with their certificates, then the certificates. */
export function DomainsPage() {
  const t = useT(certificatesMessages);
  usePageTitle(t('title'));
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains });
  const certs = useQuery({ queryKey: certificateKeys.all, queryFn: getCertificates });
  const [settings, setSettings] = useState(false);
  const both =
    domains.data && certs.data
      ? { domains: domains.data.domains, certificates: certs.data.certificates }
      : undefined;
  const load = useFirstLoad(both);
  const failed = domains.isError ? domains : certs.isError ? certs : undefined;
  let body;
  if (failed) {
    body = (
      <ErrorState
        message={t('loadFailed')}
        error={failed.error}
        onRetry={() => {
          void domains.refetch();
          void certs.refetch();
        }}
      />
    );
  } else if (!load.data) {
    body = (
      <div className="flex flex-col gap-8">
        <ListSkeleton label={t('domainsTitle')} />
        <ListSkeleton label={t('certsTitle')} />
      </div>
    );
  } else {
    body = (
      <div className="flex flex-col gap-8">
        {load.data.domains.length === 0 ? (
          <EmptyState
            icon={<SealCheckIcon size={32} />}
            title={t('emptyTitle')}
            action={
              <div className="flex flex-wrap justify-center gap-2">
                <AddButton />
                <Button
                  icon={<SlidersHorizontalIcon size={16} />}
                  className="pointer-coarse:h-9"
                  onClick={() => setSettings(true)}
                >
                  {t('acmeSettings')}
                </Button>
              </div>
            }
          >
            {t('emptyBody')}
          </EmptyState>
        ) : (
          <DomainsSection
            domains={load.data.domains}
            actions={<ToolbarActions onSettings={() => setSettings(true)} />}
          />
        )}
        {load.data.certificates.length === 0 && load.data.domains.length === 0 ? null : (
          <CertificatesSection certificates={load.data.certificates} />
        )}
      </div>
    );
  }
  return (
    <div className="flex flex-col">
      <PageHeader title={t('title')} description={t('description')} />
      <div className="flex flex-col gap-4 p-4 sm:p-8">
        <Reveal on={load.reveal}>{body}</Reveal>
      </div>
      <AcmeSettingsSheet open={settings} onOpenChange={setSettings} />
    </div>
  );
}
