import { Radio } from '@cloudflare/kumo/components/radio';
import { panelNames } from '../../components/panels/PanelLogo';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { PanelKind } from '../../lib/api/panels';
import { siteFormMessages } from './form-messages';
import type { RuleField, RuleValues } from './rule-form';
import { PortField } from './PortField';
import { TextField } from './TextField';

export interface FieldsProps {
  values: RuleValues;
  set: (patch: Partial<RuleValues>) => void;
  /** The sentence of the field the server refused. */
  errorOf: (f: RuleField) => string | undefined;
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
      {/* Two kinds: a radio pair reads faster than a menu. */}
      <Radio.Group
        legend={t('kindLabel')}
        value={values.sourceKind}
        onValueChange={(sourceKind: PanelKind) => set({ sourceKind })}
      >
        {(['marzban', 'pasarguard'] as const).map((k) => (
          <Radio.Item key={k} value={k} label={panelNames[k]} />
        ))}
      </Radio.Group>
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

/** The old path (unless the batches know it) and the port the rule answers on. */
export function AnswerFields({
  values,
  set,
  errorOf,
  path = true,
}: FieldsProps & { path?: boolean }) {
  const t = useT(siteFormMessages);
  return (
    <div className="grid gap-5 sm:grid-cols-[1fr_10rem]">
      {path ? (
        <TextField
          id="site-path"
          label={t('pathLabel')}
          hint={<CodeText text={t('pathHint')} />}
          error={errorOf('oldPath')}
          value={values.oldPath}
          onChange={(oldPath) => set({ oldPath })}
          machine
        />
      ) : (
        <span aria-hidden="true" className="hidden sm:block" />
      )}
      <PortField id="site-port" values={values} set={set} errorOf={errorOf} />
    </div>
  );
}
