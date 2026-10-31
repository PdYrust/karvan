import { Radio } from '@cloudflare/kumo/components/radio';
import { useT } from '../../i18n/locale';
import { karvanMessages } from './karvan-messages';

function Choice({ title, hint }: { title: string; hint: string }) {
  return (
    <span className="flex flex-col gap-1">
      <span className="font-medium text-kumo-strong">{title}</span>
      <span className="text-sm text-kumo-subtle">{hint}</span>
    </span>
  );
}

/**
 * How the Karvan domain answers while it has no certificate: Karvan's own certificate, or plain
 * HTTP for its name only. One port carries both, so the server's IP keeps HTTPS either way.
 */
export function KarvanPlainChoice({
  plain,
  disabled,
  onChange,
}: {
  plain: boolean;
  disabled?: boolean;
  onChange: (plain: boolean) => void;
}) {
  const t = useT(karvanMessages);
  return (
    <Radio.Group
      legend={t('noCertTitle')}
      description={t('noCertHint')}
      appearance="card"
      value={plain ? 'plain' : 'own'}
      disabled={disabled}
      onValueChange={(v: string) => onChange(v === 'plain')}
    >
      <Radio.Item value="own" label={<Choice title={t('noCertOwn')} hint={t('noCertOwnHint')} />} />
      <Radio.Item
        value="plain"
        label={<Choice title={t('noCertPlain')} hint={t('noCertPlainHint')} />}
      />
    </Radio.Group>
  );
}
