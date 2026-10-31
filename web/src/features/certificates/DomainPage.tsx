import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { StatusBadge, StatusLabel } from '../../components/feedback/StatusBadge';
import { SealCheckIcon } from '../../components/icons/regular';
import { PageHeader } from '../../components/layout/PageHeader';
import { Section } from '../../components/layout/Section';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { certificateKeys, getCertificates, type Certificate } from '../../lib/api/certificates';
import { domainKeys, getDomains, type Domain } from '../../lib/api/domains';
import { getHttps, httpsKeys } from '../../lib/api/https';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useNow } from '../overview/use-now';
import { CertFactsView } from './CertFactsView';
import { domainStateOf, ownLines } from './cert-view';
import { DomainAside } from './DomainAside';
import { domainPageMessages } from './domain-page-messages';
import { certificatesMessages } from './messages';
import { Mono, RuleName, useAcmeErrorWords } from './parts';

/** The domain's own problems, its certificate and the rules that use it. */
function Main({ domain }: { domain: Domain }) {
  const t = useT(domainPageMessages);
  const tc = useT(certificatesMessages);
  const acmeWords = useAcmeErrorWords();
  const now = useNow();
  const lines = ownLines(domain, now);
  const error = domain.acme?.lastError;
  return (
    <div className="flex min-w-0 flex-col gap-6 p-4 sm:p-8">
      {lines.length > 0 ? (
        <Section title={t('problems')}>
          <ul className="flex flex-col gap-1.5 px-4 pb-4 text-sm">
            {lines.map((l) => (
              <li key={l.key}>
                <StatusLabel tone={l.tone}>
                  {tc(`dwarn_${l.word}`, { ...l.params, reason: error ? acmeWords(error) : '' })}
                </StatusLabel>
              </li>
            ))}
          </ul>
        </Section>
      ) : null}
      <Section title={t('certificate')}>
        <div className="px-4 pb-4">
          {domain.certificate ? (
            <CertFactsView facts={domain.certificate.facts} />
          ) : (
            <p className="text-kumo-subtle">{t('noCertificate')}</p>
          )}
        </div>
      </Section>
      <Section title={t('rules')}>
        {domain.rules.length === 0 ? (
          <p className="px-4 pb-4 text-kumo-subtle">{t('noRules')}</p>
        ) : (
          <ul className="flex flex-col px-4 pb-2">
            {domain.rules.map((r) => (
              <li
                key={`${r.kind}:${String(r.id)}`}
                className="border-b border-kumo-hairline py-3 last:border-0"
              >
                <RuleName rule={r} />
              </li>
            ))}
          </ul>
        )}
      </Section>
    </div>
  );
}

function Loaded({
  domain,
  cert,
  servesPanel,
  reveal,
}: {
  domain: Domain;
  cert: Certificate | undefined;
  servesPanel: boolean | undefined;
  reveal: boolean;
}) {
  const tc = useT(certificatesMessages);
  const now = useNow();
  const { state, tone } = domainStateOf(domain, now);
  return (
    <>
      <PageHeader
        eyebrow={<Eyebrow />}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <Mono text={domain.name} className="min-w-0" />
            <StatusBadge tone={tone}>{tc(`state_${state}`)}</StatusBadge>
          </span>
        }
      />
      <Reveal on={reveal}>
        <div className="grid lg:grid-cols-[minmax(0,1fr)_22rem]">
          <Main domain={domain} />
          <aside
            aria-label={tc('colCert')}
            className="border-t border-kumo-hairline p-4 sm:p-8 lg:col-start-2 lg:row-start-1 lg:border-s lg:border-t-0"
          >
            <DomainAside domain={domain} cert={cert} servesPanel={servesPanel} />
          </aside>
        </div>
      </Reveal>
    </>
  );
}

function Eyebrow() {
  const t = useT(certificatesMessages);
  return (
    <Link
      to="/domains"
      className="inline-flex items-center hover:text-kumo-default pointer-coarse:min-h-9"
    >
      {t('title')}
    </Link>
  );
}

/** One domain, read only: its certificate, renewal, rules and whether it serves the Karvan panel. */
export function DomainPage() {
  const { id = '' } = useParams();
  const t = useT(domainPageMessages);
  const tc = useT(certificatesMessages);
  const num = Number(id);
  const valid = Number.isInteger(num) && num > 0;
  const domains = useQuery({ queryKey: domainKeys.all, queryFn: getDomains, enabled: valid });
  const certs = useQuery({
    queryKey: certificateKeys.all,
    queryFn: getCertificates,
    enabled: valid,
  });
  const https = useQuery({ queryKey: httpsKeys.all, queryFn: getHttps, enabled: valid });
  const both =
    domains.data && certs.data
      ? { domain: domains.data.domains.find((d) => d.id === num), certs: certs.data.certificates }
      : undefined;
  const load = useFirstLoad(both);
  const domain = load.data?.domain;
  usePageTitle(domain?.name ?? tc('title'));
  if (domain) {
    const h = https.data;
    return (
      <div className="flex flex-col">
        <Loaded
          domain={domain}
          cert={load.data?.certs.find((c) => c.id === domain.certificate?.id)}
          servesPanel={h ? h.tls === 'domain' && h.domainId === domain.id : undefined}
          reveal={load.reveal}
        />
      </div>
    );
  }
  const failed = domains.isError ? domains : certs.isError ? certs : undefined;
  const missing = !valid || (load.data !== undefined && !domain);
  return (
    <div className="flex flex-col">
      <PageHeader eyebrow={<Eyebrow />} title={tc('colDomain')} />
      <div className="flex flex-col gap-6 p-4 sm:p-8">
        {missing ? (
          <EmptyState icon={<SealCheckIcon size={32} />} title={t('notFound')} />
        ) : failed ? (
          <ErrorState
            message={t('loadFailed')}
            error={failed.error}
            onRetry={() => {
              void domains.refetch();
              void certs.refetch();
            }}
          />
        ) : (
          <div role="status" aria-label={tc('colDomain')} className="flex flex-col gap-4">
            <Bone className="h-24 w-full" />
            <Bone className="h-64 w-full" />
          </div>
        )}
      </div>
    </div>
  );
}
