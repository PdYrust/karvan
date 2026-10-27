import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { Select } from '@cloudflare/kumo/components/select';
import { useState, type FormEvent } from 'react';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { PanelKind } from '../../../lib/api/panels';
import type { ProblemGroup } from '../../../lib/api/migrations';
import { setFix, type Resolution } from '../../../lib/plan/plan';
import { useWizard } from '../wizard-context';
import { conflictMessages } from './conflict-messages';
import { compareMessages } from './messages';
import { fixForAll, fixParam, fixWords } from './problems';

/** Shadowsocks methods each destination takes (panel rules). */
const methods: Record<PanelKind, string[]> = {
  pasarguard: ['aes-128-gcm', 'aes-256-gcm', 'chacha20-ietf-poly1305', 'xchacha20-poly1305'],
  marzban: ['aes-128-gcm', 'aes-256-gcm', 'chacha20-ietf-poly1305'],
};

/** The value a prefix or suffix needs, saved when the operator applies it. */
function TextParam({ type, option, value }: { type: string; option: string; value: string }) {
  const t = useT(compareMessages);
  const { plan } = useWizard();
  const [draft, setDraft] = useState(value);
  const id = `fix-${type}-text`;
  const apply = (e: FormEvent) => {
    e.preventDefault();
    plan.edit(setFix(type, { option, params: { text: draft } }));
  };
  return (
    <form onSubmit={apply} className="flex flex-col gap-1.5 ps-7">
      <label id={`${id}-label`} htmlFor={id} className="text-sm text-kumo-default">
        {t('textLabel')}
      </label>
      <div className="flex gap-2">
        <Input
          id={id}
          aria-labelledby={`${id}-label`}
          dir="ltr"
          autoComplete="off"
          spellCheck={false}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          className="w-full sm:w-56 pointer-coarse:h-9"
        />
        <Button
          type="submit"
          disabled={draft === value && option !== 'prefix'}
          className="pointer-coarse:h-9"
        >
          {t('apply')}
        </Button>
      </div>
      {option === 'suffix' ? (
        <p className="text-sm text-kumo-subtle">
          <CodeText text={t('suffixHint')} />
        </p>
      ) : null}
    </form>
  );
}

/** One fix for every user of a group: the fixes best first, and the value the chosen one needs. */
export function FixForAll({ group, kind }: { group: ProblemGroup; kind: PanelKind }) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { plan } = useWizard();
  const current = plan.plan ? fixForAll(plan.plan, group) : undefined;
  // A prefix needs its text before it means anything, so it is picked here first.
  const [picked, setPicked] = useState<string | null>(null);
  const shown = picked ?? current?.option ?? '';
  const options = group.options.filter((o) => o.id !== 'rename');
  const choose = (option: string) => {
    if (option === 'prefix') return setPicked(option);
    setPicked(null);
    const fix: Resolution =
      option === 'change_method'
        ? { option, params: { method: current?.params?.method ?? methods[kind][0] ?? '' } }
        : { option };
    plan.edit(setFix(group.type, fix));
  };
  const only = options.length === 1 ? options[0] : undefined;
  if (only) {
    return (
      <p className="text-kumo-default">
        {t('onlyFix', { fix: tc(fixWords(only.id), { fix: only.id }) })}
      </p>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      <Radio.Group legend={t('fixForAll')} value={shown} onValueChange={choose}>
        {options.map((o) => (
          <Radio.Item
            key={o.id}
            value={o.id}
            label={
              <span className="flex flex-wrap items-center gap-2">
                {tc(fixWords(o.id), { fix: o.id })}
                {o.changesConfig ? <Badge variant="outline">{t('configFlag')}</Badge> : null}
              </span>
            }
          />
        ))}
      </Radio.Group>
      {fixParam[shown] === 'text' ? (
        <TextParam
          key={`${shown}:${current?.params?.text ?? ''}`}
          type={group.type}
          option={shown}
          value={current?.option === shown ? (current.params?.text ?? '') : ''}
        />
      ) : null}
      {shown === 'change_method' && current?.option === 'change_method' ? (
        <div className="flex flex-col gap-1.5 ps-7">
          <span id={`fix-${group.type}-method`} className="text-sm text-kumo-default">
            {t('methodLabel')}
          </span>
          <Select
            aria-labelledby={`fix-${group.type}-method`}
            className="w-full sm:w-64 pointer-coarse:h-9"
            value={current.params?.method ?? ''}
            items={Object.fromEntries(methods[kind].map((m) => [m, m]))}
            onValueChange={(m) =>
              m && plan.edit(setFix(group.type, { option: 'change_method', params: { method: m } }))
            }
          />
        </div>
      ) : null}
    </div>
  );
}
