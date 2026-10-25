import { Input } from '@cloudflare/kumo/components/input';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useId } from 'react';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { FormField } from '../../../components/form/FormField';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import { accessMessages } from './messages';
import { chosenPort, portDigits, type PortChoice } from './port';

/** Random or typed, with the firewall and proxy warning next to it (ADR 0009). */
export function PortFields({
  choice,
  current,
  disabled,
  onChange,
}: {
  choice: PortChoice;
  current: number;
  disabled: boolean;
  onChange: (c: PortChoice) => void;
}) {
  const t = useT(accessMessages);
  const id = useId();
  const typed = choice.kind === 'typed' ? choice.text : '';
  const invalid = choice.kind === 'typed' && typed !== '' && chosenPort(choice, current) === null;
  return (
    <div className="flex flex-col gap-4">
      <Radio.Group
        legend={t('portChoice')}
        value={choice.kind}
        disabled={disabled}
        onValueChange={(v: string) =>
          onChange(v === 'typed' ? { kind: 'typed', text: typed } : { kind: 'random' })
        }
      >
        <Radio.Item value="random" label={t('portRandom')} />
        <Radio.Item value="typed" label={t('portTyped')} />
      </Radio.Group>
      {choice.kind === 'typed' ? (
        <FormField
          id={`${id}-port`}
          label={t('portNumber')}
          hint={
            invalid ? <CodeText text={t('portInvalid', { port: `\`${current}\`` })} /> : undefined
          }
        >
          <Input
            id={`${id}-port`}
            aria-labelledby={`${id}-port-label`}
            aria-describedby={invalid ? `${id}-port-hint` : undefined}
            aria-invalid={invalid}
            dir="ltr"
            inputMode="numeric"
            autoComplete="off"
            value={typed}
            disabled={disabled}
            onChange={(e) => onChange({ kind: 'typed', text: portDigits(e.target.value) })}
            className="h-10 w-32 font-mono pointer-coarse:h-11"
          />
        </FormField>
      ) : null}
      <InlineAlert tone="warning">{t('firewall')}</InlineAlert>
    </div>
  );
}
