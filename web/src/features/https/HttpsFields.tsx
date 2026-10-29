import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { Select } from '@cloudflare/kumo/components/select';
import { useId } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { FormField } from '../../components/form/FormField';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import type { Https, TlsMode } from '../../lib/api/https';
import { accessMessages } from '../settings/access/messages';
import { sharedPortsOf, typedDigits, type HttpsForm, type Where } from './https-form';
import { httpsMessages } from './messages';

const modes: readonly TlsMode[] = ['off', 'self', 'domain'];

/** The mode as radio cards, the domain in domain mode, then the port (D9, D10, D12). */
export function HttpsFields({
  form,
  https,
  domains,
  disabled,
  onChange,
}: {
  form: HttpsForm;
  https: Https;
  /** The domains with a certificate for their name. */
  domains: readonly Domain[];
  disabled: boolean;
  onChange: (f: HttpsForm) => void;
}) {
  const t = useT(httpsMessages);
  const ta = useT(accessMessages);
  const id = useId();
  const portsFor = (f: HttpsForm) =>
    f.tls === 'domain'
      ? sharedPortsOf(
          https,
          domains.find((d) => d.id === f.domainId),
        )
      : [];
  // A shared port follows the domain; without one the choice goes back to this port.
  const set = (f: Partial<HttpsForm>) => {
    const next = { ...form, ...f };
    const ports = portsFor(next);
    if (next.where === 'shared' && !ports.includes(next.sharedPort))
      Object.assign(
        next,
        ports.length > 0 ? { sharedPort: ports[0] } : { where: 'keep', sharedPort: 0 },
      );
    onChange(next);
  };
  const isDomain = form.tls === 'domain';
  const shared = portsFor(form);
  const { where } = form;
  const typedBad =
    where === 'typed' &&
    form.typed !== '' &&
    !(Number(form.typed) >= 1 && Number(form.typed) <= 65535);
  return (
    <div className="flex flex-col gap-5">
      <Radio.Group
        legend={t('modeChoice')}
        appearance="card"
        value={form.tls}
        disabled={disabled}
        onValueChange={(v: TlsMode) => set({ tls: v })}
        className="w-full"
      >
        {modes.map((m) => (
          <Radio.Item
            key={m}
            value={m}
            label={
              <span className="flex flex-col gap-0.5">
                <span className="font-medium text-kumo-strong">{t(m)}</span>
                <span className="text-sm text-kumo-subtle">{t(`${m}Hint`)}</span>
              </span>
            }
          />
        ))}
      </Radio.Group>
      {form.tls === 'off' ? <InlineAlert>{t('plainWarning')}</InlineAlert> : null}
      {isDomain && domains.length === 0 ? (
        <InlineAlert tone="info">{t('noDomains')}</InlineAlert>
      ) : null}
      {isDomain && domains.length > 0 ? (
        <div className="flex flex-col gap-2">
          <span id={`${id}-domain`} className="text-sm font-medium text-kumo-strong">
            {t('domainLabel')}
          </span>
          <Select
            aria-labelledby={`${id}-domain`}
            placeholder={t('pickDomain')}
            className="w-full sm:w-72 pointer-coarse:h-9"
            disabled={disabled}
            value={form.domainId > 0 ? String(form.domainId) : null}
            items={Object.fromEntries(domains.map((d) => [String(d.id), d.name]))}
            onValueChange={(v) => set({ domainId: Number(v ?? 0) })}
          />
        </div>
      ) : null}
      <Radio.Group
        legend={t('whereChoice')}
        value={where}
        disabled={disabled}
        onValueChange={(v: Where) => set({ where: v })}
      >
        <Radio.Item value="keep" label={t('whereKeep', { port: String(https.port) })} />
        <Radio.Item value="typed" label={t('whereTyped')} />
        {shared.length > 0 ? <Radio.Item value="shared" label={t('whereShared')} /> : null}
      </Radio.Group>
      {where === 'typed' ? (
        <FormField
          id={`${id}-port`}
          label={t('portNumber')}
          hint={typedBad ? t('portInvalid') : undefined}
        >
          <Input
            id={`${id}-port`}
            aria-labelledby={`${id}-port-label`}
            aria-describedby={typedBad ? `${id}-port-hint` : undefined}
            aria-invalid={typedBad}
            dir="ltr"
            inputMode="numeric"
            autoComplete="off"
            value={form.typed}
            disabled={disabled}
            onChange={(e) => set({ typed: typedDigits(e.target.value) })}
            className="h-10 w-32 font-mono pointer-coarse:h-9"
          />
        </FormField>
      ) : null}
      {where === 'shared' ? (
        <div className="flex flex-col gap-2">
          <span id={`${id}-rule`} className="text-sm font-medium text-kumo-strong">
            {t('rulePort')}
          </span>
          <Select
            aria-labelledby={`${id}-rule`}
            className="w-32 pointer-coarse:h-9"
            disabled={disabled}
            value={String(form.sharedPort)}
            items={Object.fromEntries(shared.map((p) => [String(p), String(p)]))}
            onValueChange={(v) => set({ sharedPort: Number(v ?? 0) })}
          />
        </div>
      ) : null}
      {where === 'typed' ? <InlineAlert tone="warning">{ta('firewall')}</InlineAlert> : null}
    </div>
  );
}
