import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useT } from '../../i18n/locale';
import type { AcmeMethod } from '../../lib/api/acme';
import { acmeMethodsFor } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { linkClass } from './parts';

/** The subscriber agreement the first order of a directory accepts (D66). */
const agreementUrl = 'https://letsencrypt.org/repository/';

/** How Let's Encrypt checks the domain, and the agreement before the first order. */
export function AcmeFields({
  name,
  method,
  agreeTos,
  needsAgreement,
  set,
}: {
  name: string;
  method: AcmeMethod;
  agreeTos: boolean;
  needsAgreement: boolean;
  set: (patch: { method?: AcmeMethod; agreeTos?: boolean }) => void;
}) {
  const t = useT(domainFormMessages);
  const allowed = acmeMethodsFor(name);
  const item = (m: AcmeMethod, key: 'http' | 'dns') => (
    <Radio.Item
      value={m}
      disabled={!allowed.includes(m)}
      label={
        <span className="flex flex-col gap-0.5">
          <span className="font-medium text-kumo-strong">{t(`method_${key}`)}</span>
          <span className="text-sm text-kumo-subtle">
            {allowed.includes(m) ? t(`methodHint_${key}`) : t('wildcardDns')}
          </span>
        </span>
      }
    />
  );
  return (
    <div className="flex flex-col gap-4">
      <Radio.Group
        legend={t('way_acme')}
        value={method}
        onValueChange={(m: AcmeMethod) => set({ method: m })}
        className="[&_legend]:sr-only"
      >
        {item('http-01', 'http')}
        {item('dns-01', 'dns')}
      </Radio.Group>
      {needsAgreement ? (
        <div className="flex flex-col items-start gap-1.5">
          <Checkbox
            label={t('agree')}
            checked={agreeTos}
            onCheckedChange={(on) => set({ agreeTos: on })}
          />
          <a
            href={agreementUrl}
            target="_blank"
            rel="noreferrer"
            className={`ms-6 text-sm ${linkClass}`}
          >
            {t('agreeLink')}
          </a>
        </div>
      ) : null}
    </div>
  );
}
