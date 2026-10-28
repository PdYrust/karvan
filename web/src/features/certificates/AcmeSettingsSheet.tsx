import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { SheetDialog } from '../../components/dialogs/SheetDialog';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { SwitchRow } from '../../components/form/SwitchRow';
import { useT } from '../../i18n/locale';
import {
  acmeKeys,
  getAcmeSettings,
  putAcmeSettings,
  type AcmeSettings,
  type AcmeSettingsView,
} from '../../lib/api/acme';
import { errorOf } from '../../lib/errors/errors';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { TextField } from '../rules/TextField';
import { domainFormMessages } from './domain-messages';

function SettingsForm({ view, onDone }: { view: AcmeSettingsView; onDone: () => void }) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const [v, setV] = useState<AcmeSettings>({
    staging: view.staging,
    email: view.email,
    keepPort80: view.keepPort80,
  });
  const save = useMutation({
    mutationFn: putAcmeSettings,
    onSuccess: (next) => {
      qc.setQueryData(acmeKeys.settings, next);
      toasts.add({ title: t('settingsSaved') });
      onDone();
    },
  });
  const emailRefused = save.isError && errorOf(save.error).detail.field === 'email';
  return (
    <form
      noValidate
      className="flex flex-col gap-6"
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate({ ...v, email: v.email.trim() });
      }}
    >
      {view.testDirectory ? (
        <p className="text-sm text-kumo-subtle">{t('testCa')}</p>
      ) : (
        <SwitchRow
          id="acme-staging"
          label={t('staging')}
          hint={t('stagingHint')}
          checked={v.staging}
          onChange={(staging) => setV({ ...v, staging })}
        />
      )}
      <TextField
        id="acme-email"
        label={t('email')}
        hint={t('emailHint')}
        error={emailRefused ? errorText(save.error) : undefined}
        value={v.email}
        onChange={(email) => setV({ ...v, email })}
        inputMode="text"
        machine
      />
      <SwitchRow
        id="acme-keep-port"
        label={t('keepPort')}
        hint={t('keepPortHint')}
        checked={v.keepPort80}
        onChange={(keepPort80) => setV({ ...v, keepPort80 })}
      />
      {save.isError && !emailRefused ? (
        <InlineAlert>{`${t('settingsFailed')} ${errorText(save.error)}`}</InlineAlert>
      ) : null}
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" className="pointer-coarse:h-9" onClick={onDone}>
          {t('cancel')}
        </Button>
        <Button
          type="submit"
          variant="primary"
          loading={save.isPending}
          className="pointer-coarse:h-9"
        >
          {t('save')}
        </Button>
      </div>
    </form>
  );
}

/** The Let's Encrypt settings (D73) in a sheet on Domains and certificates. */
export function AcmeSettingsSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const view = useQuery({ queryKey: acmeKeys.settings, queryFn: getAcmeSettings, enabled: open });
  let body;
  if (view.isError) {
    body = <InlineAlert>{`${t('settingsFailed')} ${errorText(view.error)}`}</InlineAlert>;
  } else if (!view.data) {
    body = (
      <div role="status" aria-label={t('settingsTitle')} className="flex flex-col gap-4">
        <Bone className="h-12 w-full" />
        <Bone className="h-12 w-full" />
      </div>
    );
  } else {
    // A fresh form from the saved settings each time the sheet opens.
    body = (
      <SettingsForm
        key={open ? 'open' : 'closed'}
        view={view.data}
        onDone={() => onOpenChange(false)}
      />
    );
  }
  return (
    <SheetDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('settingsTitle')}
      description={t('settingsDescription')}
    >
      {body}
    </SheetDialog>
  );
}
