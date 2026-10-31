import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { SheetDialog } from '../../components/dialogs/SheetDialog';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { SwitchRow } from '../../components/form/SwitchRow';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import {
  acmeKeys,
  getAcmeSettings,
  putAcmeSettings,
  type AcmeSettings,
  type AcmeSettingsView,
} from '../../lib/api/acme';
import {
  getKarvanDomain,
  karvanKeys,
  putKarvanDomain,
  type KarvanDomain,
} from '../../lib/api/karvan';
import { errorOf } from '../../lib/errors/errors';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { TextField } from '../rules/TextField';
import { domainFormMessages } from './domain-messages';
import { karvanMessages } from './karvan-messages';

/** The two switches of the Karvan domain; each saves when it turns. */
function KarvanSwitches({ karvan }: { karvan: KarvanDomain }) {
  const t = useT(domainFormMessages);
  const tk = useT(karvanMessages);
  const errorText = useSubscriptionErrorText();
  const qc = useQueryClient();
  const save = useMutation({
    mutationFn: putKarvanDomain,
    onSuccess: (next) => qc.setQueryData(karvanKeys.all, next),
  });
  // The switch moves on the press; a failed save puts it back.
  const shown = save.isPending ? { ...karvan, ...save.variables } : karvan;
  return (
    <>
      <SwitchRow
        id="karvan-plain"
        label={tk('plainTitle')}
        hint={tk('plainHint')}
        checked={shown.plain}
        disabled={save.isPending}
        onChange={(plain) => save.mutate({ plain })}
      />
      <SwitchRow
        id="karvan-only"
        label={tk('onlyTitle')}
        hint={<CodeText text={tk('onlyHint')} />}
        checked={shown.only}
        disabled={save.isPending}
        onChange={(only) => save.mutate({ only })}
      />
      {save.isError ? (
        <InlineAlert>{`${t('settingsFailed')} ${errorText(save.error)}`}</InlineAlert>
      ) : null}
    </>
  );
}

function SettingsForm({
  view,
  karvan,
}: {
  view: AcmeSettingsView;
  /** Left out when the server cannot change the Karvan domain. */
  karvan: KarvanDomain | undefined;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const qc = useQueryClient();
  const [email, setEmail] = useState(view.email);
  const save = useMutation({
    mutationFn: putAcmeSettings,
    onSuccess: (next) => qc.setQueryData(acmeKeys.settings, next),
  });
  const shown: AcmeSettings = save.isPending ? save.variables : view;
  // Every save sends the email as typed, so a switch never puts back an older one.
  const saveWith = (change: Partial<AcmeSettings>) =>
    save.mutate({
      staging: shown.staging,
      keepPort80: shown.keepPort80,
      email: email.trim(),
      ...change,
    });
  const emailRefused = save.isError && errorOf(save.error).detail.field === 'email';
  return (
    <form
      noValidate
      className="flex flex-col gap-6"
      onSubmit={(e) => {
        e.preventDefault();
        saveWith({});
      }}
    >
      {view.testDirectory ? (
        <p className="text-sm text-kumo-subtle">{t('testCa')}</p>
      ) : (
        <SwitchRow
          id="acme-staging"
          label={t('staging')}
          hint={t('stagingHint')}
          checked={shown.staging}
          disabled={save.isPending}
          onChange={(staging) => saveWith({ staging })}
        />
      )}
      <TextField
        id="acme-email"
        label={t('email')}
        hint={t('emailHint')}
        error={emailRefused ? errorText(save.error) : undefined}
        value={email}
        onChange={setEmail}
        onBlur={() => {
          if (email.trim() !== view.email) saveWith({});
        }}
        inputMode="text"
        machine
      />
      <SwitchRow
        id="acme-keep-port"
        label={t('keepPort')}
        hint={t('keepPortHint')}
        checked={shown.keepPort80}
        disabled={save.isPending}
        onChange={(keepPort80) => saveWith({ keepPort80 })}
      />
      {save.isError && !emailRefused ? (
        <InlineAlert>{`${t('settingsFailed')} ${errorText(save.error)}`}</InlineAlert>
      ) : null}
      {karvan ? <KarvanSwitches karvan={karvan} /> : null}
    </form>
  );
}

/**
 * The settings of Domains and certificates in a sheet: Let's Encrypt, and the Karvan domain.
 * Each control saves as it changes; the email when the field is left.
 */
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
  // A server that cannot change the Karvan domain answers 404: the sheet leaves its switches out.
  const karvan = useQuery({ queryKey: karvanKeys.all, queryFn: getKarvanDomain, enabled: open });
  let body;
  if (view.isError) {
    body = <InlineAlert>{`${t('settingsFailed')} ${errorText(view.error)}`}</InlineAlert>;
  } else if (!view.data || karvan.isPending) {
    body = (
      <div role="status" aria-label={t('settingsTitle')} className="flex flex-col gap-4">
        <Bone className="h-12 w-full" />
        <Bone className="h-12 w-full" />
      </div>
    );
  } else {
    // A fresh form from the saved settings each time the sheet opens.
    body = <SettingsForm key={open ? 'open' : 'closed'} view={view.data} karvan={karvan.data} />;
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
