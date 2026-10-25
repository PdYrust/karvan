import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { FormField } from '../../../components/form/FormField';
import { JalaliDateField } from '../../../components/form/JalaliDateField';
import { SignedAmountField } from '../../../components/form/SignedAmountField';
import { useT } from '../../../i18n/locale';
import type { JobSummary } from '../../../lib/api/migrations';
import { signedAmount } from '../../../lib/plan/changes';
import { useFormat } from '../../../lib/preferences/region';
import { protocolName } from '../protocols';
import type { OpForm, OpKind } from './build';
import { changesMessages, type ChangesKey } from './messages';

/** Protocols cred.new takes (api-migration.md section 5). */
const credProtocols = ['vmess', 'vless', 'trojan', 'shadowsocks', 'hysteria', 'wireguard'];

interface FieldsProps {
  form: OpForm;
  onChange: (f: OpForm) => void;
}

function TextField({
  id,
  label,
  value,
  onText,
}: {
  id: string;
  label: string;
  value: string;
  onText: (v: string) => void;
}) {
  return (
    <FormField id={id} label={label}>
      <Input
        id={id}
        aria-labelledby={`${id}-label`}
        dir="auto"
        autoComplete="off"
        spellCheck={false}
        value={value}
        onChange={(e) => onText(e.target.value)}
        className="w-full sm:w-72 pointer-coarse:h-11"
      />
    </FormField>
  );
}

/** A number above 0 with its unit: a traffic limit or a number of days. */
function AmountField({
  id,
  label,
  unit,
  form,
  onChange,
}: FieldsProps & { id: string; label: string; unit: string }) {
  return (
    <FormField id={id} label={label}>
      <div className="flex items-center gap-2">
        <Input
          id={id}
          aria-labelledby={`${id}-label`}
          dir="ltr"
          inputMode="decimal"
          autoComplete="off"
          value={form.amount}
          onChange={(e) => onChange({ ...form, amount: e.target.value })}
          className="w-28 tabular pointer-coarse:h-11"
        />
        <span className="text-kumo-subtle">{unit}</span>
      </div>
    </FormField>
  );
}

/** A list of checkboxes, such as groups or protocols, kept in form[field]. */
function Picks({
  legend,
  items,
  chosen,
  onPick,
}: {
  legend: string;
  items: readonly [string, string][];
  chosen: readonly string[];
  onPick: (ids: string[]) => void;
}) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 text-sm font-medium text-kumo-strong">{legend}</legend>
      {items.map(([id, name]) => (
        <Checkbox
          key={id}
          label={<bdi>{name}</bdi>}
          checked={chosen.includes(id)}
          onCheckedChange={(on) => onPick(on ? [...chosen, id] : chosen.filter((x) => x !== id))}
        />
      ))}
    </fieldset>
  );
}

/** The fields one operation needs, with what a signed number does said under it. */
export function OpFields({
  kind,
  summary,
  form,
  onChange,
}: FieldsProps & { kind: OpKind; summary: JobSummary }) {
  const t = useT(changesMessages);
  const fmt = useFormat();
  const set = (patch: Partial<OpForm>) => onChange({ ...form, ...patch });
  const signed = (whole: boolean, up: ChangesKey, down: ChangesKey) => {
    const v = signedAmount(form.amount, whole);
    if (v === null) return undefined;
    const n = Math.abs(Number(v));
    return t(Number(v) < 0 ? down : up, { amount: fmt.number(n, 2), count: n });
  };
  switch (kind) {
    case 'traffic.add':
      return (
        <SignedAmountField
          id="change-gb"
          label={t('trafficLabel')}
          unit={t('gb')}
          value={form.amount}
          onChange={(amount) => set({ amount })}
          hint={signed(false, 'addsGb', 'takesGb')}
        />
      );
    case 'time.add':
      return (
        <SignedAmountField
          id="change-days"
          label={t('daysLabel')}
          unit={t('days')}
          value={form.amount}
          onChange={(amount) => set({ amount })}
          hint={signed(true, 'addsDays', 'takesDays')}
        />
      );
    case 'traffic.set':
      return (
        <AmountField
          id="change-limit"
          label={t('limitLabel')}
          unit={t('gb')}
          form={form}
          onChange={onChange}
        />
      );
    case 'time.restart':
      return (
        <AmountField
          id="change-period"
          label={t('daysLabel')}
          unit={t('days')}
          form={form}
          onChange={onChange}
        />
      );
    case 'time.set':
      return (
        <JalaliDateField
          id="change-date"
          label={t('dateLabel')}
          value={form.date}
          onChange={(date) => set({ date })}
        />
      );
    case 'status.set':
      return (
        <Radio.Group
          legend={t('statusLabel')}
          value={form.status}
          onValueChange={(s) => set({ status: s as OpForm['status'] })}
        >
          <Radio.Item value="active" label={t('active')} />
          <Radio.Item value="disabled" label={t('disabled')} />
        </Radio.Group>
      );
    case 'name.prefix':
    case 'name.suffix':
    case 'note.set':
    case 'note.append':
      return (
        <TextField
          id="change-text"
          label={t(kind.startsWith('note') ? 'noteLabel' : 'textLabel')}
          value={form.text}
          onText={(text) => set({ text })}
        />
      );
    case 'name.replace':
      return (
        <div className="flex flex-wrap gap-3">
          <TextField
            id="change-from"
            label={t('fromLabel')}
            value={form.from}
            onText={(from) => set({ from })}
          />
          <TextField
            id="change-to"
            label={t('toLabel')}
            value={form.to}
            onText={(to) => set({ to })}
          />
        </div>
      );
    case 'place.targets':
      return (
        <Picks
          legend={t(summary.destKind === 'marzban' ? 'targetsLabelInbounds' : 'targetsLabelGroups')}
          items={(summary.targets ?? []).map((g) => [g.id, g.name])}
          chosen={form.ids}
          onPick={(ids) => set({ ids })}
        />
      );
    case 'cred.new':
      return (
        <div className="flex flex-col gap-4">
          <InlineAlert tone="warning">{t('credWarning')}</InlineAlert>
          <Picks
            legend={t('protocolsLabel')}
            items={credProtocols
              .filter((p) => summary.protocols[p])
              .map((p) => [p, protocolName(p)])}
            chosen={form.protocols}
            onPick={(protocols) => set({ protocols })}
          />
          <Checkbox
            label={t('credUnderstood')}
            checked={form.understood}
            onCheckedChange={(understood) => set({ understood })}
          />
        </div>
      );
    case 'skip':
      return <p className="text-kumo-subtle">{t('skipNote')}</p>;
    default:
      return null;
  }
}
