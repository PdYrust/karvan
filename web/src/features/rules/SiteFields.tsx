import type { ReactNode } from 'react';
import { SwitchRow } from '../../components/form/SwitchRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { ChoiceList } from '../certificates/ChoiceList';
import { siteFormMessages } from './form-messages';
import { PortField } from './PortField';
import {
  ruleKinds,
  type RuleField,
  type RuleKind,
  type RuleStep,
  type RuleValues,
} from './rule-form';
import { TextField } from './TextField';

export const action = 'flex-1 justify-center sm:flex-none pointer-coarse:h-9';

/** The bar at the bottom of a wizard step. */
export function Footer({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col gap-3 border-t border-kumo-hairline pt-5 sm:flex-row sm:items-center sm:justify-end">
      <div className="flex gap-2 sm:justify-end">{children}</div>
    </div>
  );
}

/** The first step: choose From an old panel or Custom. */
export function KindQuestion({
  chosen,
  onChoose,
}: {
  chosen?: RuleKind;
  onChoose: (k: RuleKind) => void;
}) {
  const t = useT(siteFormMessages);
  return (
    <ChoiceList<RuleKind>
      label={t('step_kind')}
      chosen={chosen}
      rows={ruleKinds.map((k) => ({
        value: k,
        title: t(`kind_${k}`),
        hint: t(`kind_${k}_hint`),
      }))}
      onChoose={onChoose}
    />
  );
}

export interface FieldsProps {
  values: RuleValues;
  set: (patch: Partial<RuleValues>) => void;
  /** The sentence of the field the server refused. */
  errorOf: (f: RuleField) => string | undefined;
}

const mark = '\u0000';

/** The HTTPS question's heading, the rule name kept isolated inside it. */
function HttpsTitle({ name }: { name: string }) {
  const t = useT(siteFormMessages);
  const [before, after] = t('httpsTitle', { name: mark }).split(mark);
  return (
    <>
      {before}
      <bdi>{name.trim()}</bdi>
      {after}
    </>
  );
}

/** The heading of each step in the rule wizard. */
export function StepHeading({
  step,
  kind,
  name,
}: {
  step: RuleStep;
  kind: RuleKind;
  name: string;
}) {
  const t = useT(siteFormMessages);
  if (step === 'kind') return t('kindTitle');
  if (step === 'domain') return t(`domainTitle_${kind}`);
  if (step === 'https') return <HttpsTitle name={name} />;
  return t(`detailsTitle_${kind}`);
}

/** The old path and the port the rule answers on. */
export function AnswerFields({ values, set, errorOf }: FieldsProps) {
  const t = useT(siteFormMessages);
  return (
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
      <PortField id="site-port" values={values} set={set} errorOf={errorOf} />
    </div>
  );
}

/** Details of a rule of batches: path, port, optional IP, and lazy/plain switches. */
export function BatchesDetailsFields({
  values,
  set,
  errorOf,
  refused,
  plainSource,
}: FieldsProps & { refused: RuleField | undefined; plainSource: boolean }) {
  const t = useT(siteFormMessages);
  return (
    <>
      <AnswerFields values={values} set={set} errorOf={errorOf} />
      {values.sourceIp || refused === 'sourceIp' ? (
        <TextField
          id="site-ip"
          label={t('ipLabel')}
          hint={t('ipHint')}
          error={errorOf('sourceIp')}
          value={values.sourceIp}
          onChange={(sourceIp) => set({ sourceIp })}
          machine
        />
      ) : null}
      <div className="flex flex-col gap-5">
        <SwitchRow
          id="site-lazy"
          label={t('lazyLabel')}
          hint={t('lazyHint')}
          checked={values.lazy}
          onChange={(lazy) => set({ lazy })}
        />
        {plainSource ? (
          <SwitchRow
            id="site-plain"
            label={t('plainLabel')}
            hint={t('plainHint')}
            checked={values.plainHttpOk}
            onChange={(plainHttpOk) => set({ plainHttpOk })}
          />
        ) : null}
      </div>
    </>
  );
}
