import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { Link } from 'react-router';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { SwitchRow } from '../../components/form/SwitchRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { siteFormMessages } from './form-messages';
import { domainState, httpsAllowed } from './rule-form';

const linkClass =
  'inline-flex items-center text-sm text-kumo-default underline decoration-kumo-line underline-offset-4 hover:decoration-current pointer-coarse:min-h-9';

const mono = (text: string) => (
  <bdi dir="ltr" className="font-mono text-[0.9em] break-all">
    {text}
  </bdi>
);

/** A host the rule wants that is not registered: add it first, with its name filled in. */
export function AddFirst({ host }: { host: string }) {
  const t = useT(siteFormMessages);
  return (
    <li className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-2.5">
      <span className="text-kumo-subtle">{mono(host)}</span>
      <Link to={`/domains/new?name=${encodeURIComponent(host)}`} className={linkClass}>
        {t('addFirst')}
      </Link>
    </li>
  );
}

/**
 * Answer over HTTPS: locked off while a picked domain has no certificate,
 * with the plain HTTP warning whenever it is off. No domain: the port decides.
 */
export function HttpsField({
  domains,
  picked,
  https,
  onHttps,
}: {
  domains: readonly Domain[];
  picked: readonly string[];
  https: boolean;
  onHttps: (on: boolean) => void;
}) {
  const t = useT(siteFormMessages);
  const allowed = httpsAllowed(picked, domains);
  if (picked.length === 0) return <p className="text-sm text-kumo-subtle">{t('portCert')}</p>;
  return (
    <>
      <SwitchRow
        id="rule-https"
        label={t('httpsLabel')}
        hint={t('httpsHint')}
        checked={https && allowed}
        disabled={!allowed}
        onChange={onHttps}
      />
      {https && allowed ? null : <InlineAlert tone="warning">{t('plainWarning')}</InlineAlert>}
    </>
  );
}

/**
 * The registered domains as checkboxes, each with its certificate state, then
 * the HTTPS switch.
 */
export function DomainPicker({
  domains,
  picked,
  missing = [],
  https,
  error,
  onPick,
  onHttps,
}: {
  domains: readonly Domain[];
  picked: readonly string[];
  /** Hosts the batches or the link name that are not registered. */
  missing?: readonly string[];
  https: boolean;
  error?: string;
  onPick: (names: string[]) => void;
  onHttps: (on: boolean) => void;
}) {
  const t = useT(siteFormMessages);
  const toggle = (name: string, on: boolean) =>
    onPick(on ? [...picked, name].sort() : picked.filter((n) => n !== name));
  const state = (d: Domain) =>
    d.certificate
      ? t('state_https', { cert: d.certificate.name })
      : t(`state_${domainState(d)}` as 'state_plain');
  return (
    <fieldset className="flex flex-col gap-5">
      <legend className="mb-1 flex flex-col gap-1">
        <span className="font-medium text-kumo-strong">{t('domainsLabel')}</span>
        <span className="text-sm text-kumo-subtle">{t('domainsHint')}</span>
      </legend>
      <ul className="flex flex-col divide-y divide-kumo-hairline rounded-lg bg-kumo-base ring ring-kumo-hairline">
        {domains.map((d) => (
          <li key={d.id} className="px-4 py-2.5">
            <Checkbox
              checked={picked.includes(d.name)}
              onCheckedChange={(on) => toggle(d.name, on)}
              label={
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="text-kumo-strong">{mono(d.name)}</span>
                  <span className="text-sm text-kumo-subtle">
                    <bdi>{state(d)}</bdi>
                  </span>
                </span>
              }
            />
          </li>
        ))}
        {domains.length === 0 ? (
          <li className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-3 text-sm text-kumo-subtle">
            {t('noDomains')}
            <Link to="/domains/new" className={linkClass}>
              {t('addDomain')}
            </Link>
          </li>
        ) : null}
        {missing.map((h) => (
          <AddFirst key={h} host={h} />
        ))}
      </ul>
      {error ? (
        <p className="text-sm text-kumo-danger">
          <CodeText text={error} />
        </p>
      ) : null}
      <HttpsField domains={domains} picked={picked} https={https} onHttps={onHttps} />
    </fieldset>
  );
}
