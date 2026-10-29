import { Button } from '@cloudflare/kumo/components/button';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { SwitchRow } from '../../components/form/SwitchRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { SiteInput } from '../../lib/api/rules';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { DestinationField } from '../wizard/connect/DestinationField';
import { siteFormMessages } from './form-messages';
import { inputOf, isPlainHttp, refusedField, type SiteField, type SiteValues } from './site-form';
import { AnswerFields, OldPanelFields } from './SiteFields';
import { TextField } from './TextField';

/** The site's fields in one column; a refused field gets its sentence under it. */
export function SiteForm({
  initial,
  siteId,
  saving,
  error,
  onSave,
  onCancel,
}: {
  initial: SiteValues;
  siteId?: number;
  saving: boolean;
  error: unknown;
  onSave: (input: SiteInput) => void;
  onCancel: () => void;
}) {
  const t = useT(siteFormMessages);
  const errorText = useSubscriptionErrorText();
  const [values, setValues] = useState(initial);
  const set = (patch: Partial<SiteValues>) => setValues((v) => ({ ...v, ...patch }));
  const refused = refusedField(error);
  const errorOf = (f: SiteField) => (refused === f ? t(`field_${f}`) : undefined);
  const fields = { values, set, errorOf };
  return (
    <form
      noValidate
      className="flex max-w-[720px] flex-col gap-8"
      onSubmit={(e) => {
        e.preventDefault();
        onSave(inputOf(values));
      }}
    >
      <TextField
        id="site-name"
        label={t('nameLabel')}
        hint={t('nameHint')}
        error={errorOf('name')}
        value={values.name}
        onChange={(name) => set({ name })}
      />
      <div className="flex flex-col gap-2">
        <DestinationField value={values.panelId} onChange={(panelId) => set({ panelId })} />
        {errorOf('panelId') ? (
          <p className="text-sm text-kumo-danger">{errorOf('panelId')}</p>
        ) : null}
      </div>
      <OldPanelFields {...fields} />
      <AnswerFields {...fields} />
      <div className="flex flex-col gap-5">
        <SwitchRow
          id="site-lazy"
          label={t('lazyLabel')}
          hint={t('lazyHint')}
          checked={values.lazy}
          onChange={(lazy) => set({ lazy })}
        />
        {isPlainHttp(values.sourceUrl) ? (
          <SwitchRow
            id="site-plain"
            label={t('plainLabel')}
            hint={t('plainHint')}
            checked={values.plainHttpOk}
            onChange={(plainHttpOk) => set({ plainHttpOk })}
          />
        ) : null}
      </div>
      {error ? (
        <InlineAlert>
          <CodeText
            text={`${t('saveFailed')} ${refused ? t(`field_${refused}`) : errorText(error)}`}
          />
        </InlineAlert>
      ) : null}
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" className="pointer-coarse:h-9" onClick={onCancel}>
          {t('cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          loading={saving}
          disabled={!values.name.trim() || !values.sourceUrl.trim()}
          className="pointer-coarse:h-9"
        >
          {siteId ? t('save') : t('addSite')}
        </Button>
      </div>
    </form>
  );
}
