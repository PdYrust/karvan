import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon, CaretRightIcon } from '@phosphor-icons/react';
import { DotsThreeIcon, PencilSimpleIcon, TrashIcon } from '../../components/icons/regular';
import { useId, useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { Collapse } from '../../components/layout/Collapse';
import { FactList } from '../../components/layout/FactList';
import { useT } from '../../i18n/locale';
import type { Certificate } from '../../lib/api/certificates';
import type { Domain } from '../../lib/api/domains';
import { useFormat } from '../../lib/preferences/region';
import { CertFactsView } from './CertFactsView';
import { daysLeft, domainWarningTone, sortedWarnings, uncoveredDomains } from './cert-view';
import { certFactMessages } from './fact-messages';
import { certificatesMessages } from './messages';
import { Mono, RuleName } from './parts';

/** Nothing uses it, so it can go; Karvan refuses the delete otherwise. */
const isUnused = (c: Certificate) => c.domains.length === 0 && c.rules.length === 0;

/** How it is kept: files, pasted, or from Let's Encrypt, which Karvan renews itself. */
const kindOf = (c: Certificate) =>
  c.origin === 'acme' ? 'kind_acme' : (`kind_${c.kind}` as const);

function CertMenu({ cert, onDelete }: { cert: Certificate; onDelete: () => void }) {
  const t = useT(certificatesMessages);
  const navigate = useNavigate();
  const label = t('moreFor', { name: cert.name });
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
        {cert.origin === 'acme' ? null : (
          <DropdownMenu.Item
            className="gap-2"
            onClick={() => void navigate(`/domains/certificates/${String(cert.id)}`)}
          >
            <PencilSimpleIcon size={16} />
            {t('edit')}
          </DropdownMenu.Item>
        )}
        {isUnused(cert) ? (
          <DropdownMenu.Item className="gap-2" variant="danger" onClick={onDelete}>
            <TrashIcon size={16} />
            {t('delete')}
          </DropdownMenu.Item>
        ) : null}
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

/** A titled part of the card, split from the next by a hairline. */
function Part({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2 px-4 py-3">
      <h3 id={id} className="text-sm font-medium text-kumo-strong">
        {title}
      </h3>
      {children}
    </section>
  );
}

/** The closed card's line: the names it covers, the end date, and its worst warning. */
function Summary({ cert, now }: { cert: Certificate; now: Date }) {
  const t = useT(certificatesMessages);
  const tf = useT(certFactMessages);
  const fmt = useFormat();
  const { facts } = cert;
  const first = sortedWarnings(facts.warnings)[0];
  const left = daysLeft(facts.notAfter, now);
  let warning: ReactNode = null;
  if (first === 'expires_soon' && left !== undefined) {
    warning = <StatusLabel tone="warning">{t('expiresIn', { count: left })}</StatusLabel>;
  } else if (first) {
    warning = <StatusLabel tone={domainWarningTone(first)}>{tf(`warn_${first}`)}</StatusLabel>;
  }
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
      {facts.domains.length > 0 ? (
        <span className="min-w-0 truncate">
          <Mono text={facts.domains.join(', ')} />
        </span>
      ) : null}
      {facts.notAfter ? (
        <span>{t('endsOn', { date: fmt.date(new Date(facts.notAfter)) })}</span>
      ) : null}
      {warning ? <span className="text-kumo-default">{warning}</span> : null}
    </div>
  );
}

/** The domains and rules that use the certificate, each with the domains of it the certificate misses. */
function CertUsers({ cert, domains }: { cert: Certificate; domains: readonly Domain[] }) {
  const t = useT(certificatesMessages);
  if (isUnused(cert)) return <p className="text-sm text-kumo-subtle">{t('notUsed')}</p>;
  const missing = (d: string) => (
    <StatusLabel key={d} tone="warning">
      {t('notCover', { domain: d })}
    </StatusLabel>
  );
  return (
    <ul className="flex flex-col gap-2 text-sm">
      {cert.domains.map((d) => (
        <li key={`domain:${String(d.id)}`} className="flex flex-col items-start gap-1">
          <Mono text={d.name} />
          {d.warnings.includes('domain_mismatch') ? missing(d.name) : null}
        </li>
      ))}
      {cert.rules.map((r) => (
        <li key={`${r.kind}:${String(r.id)}`} className="flex flex-col items-start gap-1">
          <RuleName rule={r} />
          {r.warnings.includes('domain_mismatch')
            ? uncoveredDomains(cert.id, r, domains).map(missing)
            : null}
        </li>
      ))}
    </ul>
  );
}

/** One stored certificate, closed to its summary; its title row opens the details in place. */
export function CertificateCard({
  cert,
  domains,
  now,
  onDelete,
}: {
  cert: Certificate;
  domains: readonly Domain[];
  now: Date;
  onDelete: () => void;
}) {
  const t = useT(certificatesMessages);
  const id = useId();
  const [open, setOpen] = useState(false);
  const Caret = open ? CaretDownIcon : CaretRightIcon;
  const facts = [
    { label: t('kindLabel'), value: t(kindOf(cert)) },
    ...(cert.certFile ? [{ label: t('certFile'), value: <Mono text={cert.certFile} /> }] : []),
    ...(cert.keyFile ? [{ label: t('keyFile'), value: <Mono text={cert.keyFile} /> }] : []),
  ];
  return (
    <article aria-labelledby={id} className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
      {/* The name's button covers the whole row, so a tap anywhere on it opens or closes. */}
      <header className="relative flex items-start gap-3 py-2 ps-4 pe-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1 py-1.5">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <h3 id={id} className="min-w-0 font-semibold text-kumo-strong">
              <button
                type="button"
                aria-expanded={open}
                aria-controls={`${id}-details`}
                onClick={() => setOpen((o) => !o)}
                className="text-start outline-none after:absolute after:inset-0 after:rounded-t-lg after:content-[''] focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-kumo-focus"
              >
                <bdi>{cert.name}</bdi>
              </button>
            </h3>
            <span className="text-sm text-kumo-subtle">{t(kindOf(cert))}</span>
          </div>
          <Summary cert={cert} now={now} />
        </div>
        <Caret
          size={16}
          aria-hidden="true"
          className="mt-2.5 shrink-0 text-kumo-subtle rtl:-scale-x-100"
        />
        <div className="relative">
          <CertMenu cert={cert} onDelete={onDelete} />
        </div>
      </header>
      <Collapse id={`${id}-details`} open={open}>
        <div className="grid divide-kumo-hairline border-t border-kumo-hairline lg:grid-cols-2 lg:divide-x">
          <div className="flex flex-col divide-y divide-kumo-hairline">
            <div className="px-4 py-1">
              <FactList facts={facts} />
            </div>
            <div className="px-4 py-3">
              <CertFactsView facts={cert.facts} />
            </div>
          </div>
          <div className="flex flex-col border-t border-kumo-hairline lg:border-t-0">
            <Part title={t('usedBy')}>
              <CertUsers cert={cert} domains={domains} />
            </Part>
          </div>
        </div>
      </Collapse>
    </article>
  );
}
