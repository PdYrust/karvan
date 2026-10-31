import { StatusLabel } from '../../components/feedback/StatusBadge';
import { AsideBlock } from '../../components/layout/AsideBlock';
import { FactList, type Fact } from '../../components/layout/FactList';
import { useT } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import type { Domain } from '../../lib/api/domains';
import type { KarvanDomain, KarvanUse } from '../../lib/api/karvan';
import { useFormat } from '../../lib/preferences/region';
import { certOrigin } from './cert-view';
import { domainPageMessages } from './domain-page-messages';
import { KarvanAccess } from './KarvanAccess';
import { certificatesMessages } from './messages';
import { Mono, useAcmeErrorWords } from './parts';

function DetailsBlock({ cert }: { cert: Certificate }) {
  const t = useT(domainPageMessages);
  const facts: Fact[] = [
    { label: t('name'), value: <bdi>{cert.name}</bdi> },
    { label: t('origin'), value: t(`origin_${certOrigin(cert)}`) },
  ];
  if (cert.certFile) facts.push({ label: t('certFile'), value: <Mono text={cert.certFile} /> });
  if (cert.keyFile) facts.push({ label: t('keyFile'), value: <Mono text={cert.keyFile} /> });
  return (
    <AsideBlock title={t('details')}>
      <FactList facts={facts} />
    </AsideBlock>
  );
}

/** A Let's Encrypt certificate's next renewal, how it renews, and its last error. */
function RenewalBlock({ acme }: { acme: NonNullable<Domain['acme']> }) {
  const t = useT(domainPageMessages);
  const tc = useT(certificatesMessages);
  const fmt = useFormat();
  const acmeWords = useAcmeErrorWords();
  const none = <span className="text-kumo-subtle">{t('none')}</span>;
  const facts: Fact[] = [
    {
      label: t('nextRenewal'),
      value: acme.nextRenewal ? fmt.dateTime(new Date(acme.nextRenewal)) : none,
    },
    {
      label: t('method'),
      value: acme.method ? tc(acme.method === 'dns-01' ? 'renewDns' : 'renewHttp') : none,
    },
  ];
  return (
    <AsideBlock title={t('renewal')}>
      <FactList facts={facts} />
      <div className="flex flex-col gap-1 text-sm">
        <span className="text-kumo-subtle">{t('lastError')}</span>
        {acme.lastError ? (
          <StatusLabel tone="warning">{acmeWords(acme.lastError)}</StatusLabel>
        ) : (
          none
        )}
      </div>
    </AsideBlock>
  );
}

/** The domain page's aside: its certificate's details, its renewal, and the Karvan panel. */
export function DomainAside({
  domain,
  cert,
  karvan,
  access,
}: {
  domain: Domain;
  cert: Certificate | undefined;
  /** How Karvan uses the domain; undefined while it is not known. */
  karvan: KarvanUse | undefined;
  /** The Karvan domain's settings and who the panel opens for. */
  access: KarvanDomain | undefined;
}) {
  const t = useT(domainPageMessages);
  return (
    <div className="flex flex-col">
      {cert ? <DetailsBlock cert={cert} /> : null}
      {domain.acme && domain.certificate?.origin === 'acme' ? (
        <RenewalBlock acme={domain.acme} />
      ) : null}
      {karvan === undefined ? null : (
        <AsideBlock title={t('panel')}>
          <p className="text-sm">
            {karvan === 'none' ? (
              <span className="text-kumo-subtle">{t('panelNot')}</span>
            ) : (
              <StatusLabel tone={karvan === 'plain' ? 'warning' : 'success'}>
                {t(`panel_${karvan}`)}
              </StatusLabel>
            )}
          </p>
          {karvan !== 'none' && access ? <KarvanAccess karvan={access} /> : null}
        </AsideBlock>
      )}
    </div>
  );
}
