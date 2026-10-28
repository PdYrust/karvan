import { Button } from '@cloudflare/kumo/components/button';
import { useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { CertificateInput } from '../../lib/api/certificates';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { TextField } from '../rules/TextField';
import {
  canSaveCert,
  certModes,
  certificateInputOf,
  refusedCertField,
  type CertFormField,
  type CertFormValues,
} from './cert-form';
import { CertSource } from './CertFields';
import { certificatesMessages } from './messages';

/** A stored certificate's fields in one column, as the rule form; a refused field gets its sentence. */
export function CertificateForm({
  initial,
  keepsKey,
  saving,
  error,
  onSave,
  onCancel,
}: {
  initial: CertFormValues;
  /** A stored pasted key that an empty key field keeps. */
  keepsKey: boolean;
  saving: boolean;
  error: unknown;
  onSave: (input: CertificateInput) => void;
  onCancel: () => void;
}) {
  const t = useT(certificatesMessages);
  const errorText = useSubscriptionErrorText();
  const [values, setValues] = useState(initial);
  const set = (patch: Partial<CertFormValues>) => setValues((v) => ({ ...v, ...patch }));
  const refused = refusedCertField(error);
  const reason = error ? errorText(error) : '';
  const errorOf = (f: CertFormField) => {
    if (refused !== f) return undefined;
    return f === 'name' ? t('field_name') : reason;
  };
  // Switching to files or back drops nothing, but only a paste keeps the stored key.
  const keeps = keepsKey && values.certMode !== 'files';
  return (
    <form
      noValidate
      className="flex max-w-[720px] flex-col gap-8"
      onSubmit={(e) => {
        e.preventDefault();
        onSave(certificateInputOf(values));
      }}
    >
      <TextField
        id="cert-name"
        label={t('nameLabel')}
        hint={t('nameHint')}
        error={errorOf('name')}
        value={values.name}
        onChange={(name) => set({ name })}
      />
      <fieldset className="flex flex-col gap-5">
        <CertSource
          legend={t('wayTitle')}
          modes={certModes}
          values={values}
          set={set}
          errorOf={errorOf}
          hasKey={keeps}
        />
      </fieldset>
      {error ? (
        <InlineAlert>
          <CodeText text={`${t('saveFailed')} ${refused === 'name' ? t('field_name') : reason}`} />
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
          disabled={!canSaveCert(values, keeps)}
          className="pointer-coarse:h-9"
        >
          {t('save')}
        </Button>
      </div>
    </form>
  );
}
