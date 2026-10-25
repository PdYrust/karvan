import { Select } from '@cloudflare/kumo/components/select';
import { FormField } from '../../components/form/FormField';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { latinDigits } from '../../lib/format/number';
import type { PanelKind } from '../../lib/api/panels';
import { siteFormMessages } from './form-messages';
import type { SiteField, SiteValues } from './site-form';
import { TextField } from './TextField';

export interface FieldsProps {
  values: SiteValues;
  set: (patch: Partial<SiteValues>) => void;
  /** The sentence of the field the server refused. */
  errorOf: (f: SiteField) => string | undefined;
}

function Legend({ title, hint }: { title: string; hint?: string }) {
  return (
    <legend className="mb-1 flex flex-col gap-1">
      <span className="font-medium text-kumo-strong">{title}</span>
      {hint ? <span className="text-sm text-kumo-subtle">{hint}</span> : null}
    </legend>
  );
}

/** The old panel: its kind, its address, the address Karvan reaches it by and its fingerprint. */
export function OldPanelFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  return (
    <fieldset className="flex flex-col gap-5">
      <Legend title={t('oldTitle')} hint={t('oldHint')} />
      <FormField id="site-kind" label={t('kindLabel')}>
        <Select
          id="site-kind"
          aria-labelledby="site-kind-label"
          className="h-10 w-full sm:w-64 pointer-coarse:h-11"
          value={values.sourceKind}
          onValueChange={(v) => v && set({ sourceKind: v as PanelKind })}
          items={panelNames}
        />
      </FormField>
      <TextField
        id="site-url"
        label={t('urlLabel')}
        hint={<CodeText text={t('urlHint')} />}
        error={errorOf('sourceUrl')}
        value={values.sourceUrl}
        onChange={(sourceUrl) => set({ sourceUrl })}
        machine
        inputMode="url"
      />
      <div className="grid gap-5 sm:grid-cols-2">
        <TextField
          id="site-ip"
          label={t('ipLabel')}
          hint={t('ipHint')}
          error={errorOf('sourceIp')}
          value={values.sourceIp}
          onChange={(sourceIp) => set({ sourceIp })}
          machine
        />
        <TextField
          id="site-pin"
          label={t('pinLabel')}
          hint={<CodeText text={t('pinHint')} />}
          error={errorOf('sourcePin')}
          value={values.sourcePin}
          onChange={(sourcePin) => set({ sourcePin })}
          machine
        />
      </div>
    </fieldset>
  );
}

/** What the site answers: the old path, its domains and its port. */
export function AnswerFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  return (
    <fieldset className="flex flex-col gap-5">
      <Legend title={t('answerTitle')} />
      <div className="grid gap-5 sm:grid-cols-[1fr_10rem]">
        <TextField
          id="site-path"
          label={t('pathLabel')}
          hint={<CodeText text={t('pathHint')} />}
          error={errorOf('oldPath')}
          value={values.oldPath}
          onChange={(oldPath) => set({ oldPath })}
          machine
        />
        <TextField
          id="site-port"
          label={t('portLabel')}
          error={errorOf('port')}
          value={values.port}
          onChange={(port) => set({ port: latinDigits(port).replace(/\D/g, '') })}
          machine
          inputMode="numeric"
        />
      </div>
      <TextField
        id="site-domains"
        label={t('domainsLabel')}
        hint={<CodeText text={t('domainsHint')} />}
        error={errorOf('domains')}
        value={values.domains}
        onChange={(domains) => set({ domains })}
        machine
        rows={3}
      />
    </fieldset>
  );
}
