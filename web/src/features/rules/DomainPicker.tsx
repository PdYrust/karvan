import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { Link } from 'react-router';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { ChoiceList } from '../certificates/ChoiceList';
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
 * The HTTPS step question: with HTTPS (locked when picked domains lack a
 * certificate) or plain HTTP; a press answers and advances immediately.
 */
export function RuleHttpsQuestion({
  domains,
  picked,
  chosen,
  error,
  onChoose,
}: {
  domains: readonly Domain[];
  picked: readonly string[];
  chosen?: 'with' | 'without';
  error?: string;
  onChoose: (https: boolean) => void;
}) {
  const t = useT(siteFormMessages);
  const allowed = httpsAllowed(picked, domains);
  const noDomains = picked.length === 0;
  const withHint = noDomains
    ? t('httpsHint_port')
    : allowed
      ? t('httpsHint_with')
      : t('httpsHint_missing');
  return (
    <div className="flex flex-col gap-3">
      <ChoiceList<'with' | 'without'>
        label={t('step_https')}
        chosen={chosen}
        rows={[
          {
            value: 'with',
            title: t('https_with'),
            hint: withHint,
            badge: allowed ? t('recommended') : undefined,
            disabled: !noDomains && !allowed,
          },
          {
            value: 'without',
            title: t('https_without'),
            hint: t('httpsHint_without'),
          },
        ]}
        onChoose={(choice) => onChoose(choice === 'with')}
      />
      {error ? (
        <p className="text-sm text-kumo-danger">
          <CodeText text={error} />
        </p>
      ) : null}
    </div>
  );
}

/**
 * The registered domains as checkboxes, each with its certificate state.
 */
export function DomainPicker({
  domains,
  picked,
  missing = [],
  error,
  onPick,
}: {
  domains: readonly Domain[];
  picked: readonly string[];
  /** Hosts the batches name that are not registered. */
  missing?: readonly string[];
  error?: string;
  onPick: (names: string[]) => void;
}) {
  const t = useT(siteFormMessages);
  const toggle = (name: string, on: boolean) =>
    onPick(on ? [...picked, name].sort() : picked.filter((n) => n !== name));
  const state = (d: Domain) =>
    d.certificate
      ? t('state_https', { cert: d.certificate.name })
      : t(`state_${domainState(d)}` as 'state_plain');
  return (
    <fieldset className="flex flex-col gap-3">
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
    </fieldset>
  );
}
