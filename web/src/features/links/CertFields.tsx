import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { FileArrowUpIcon, SealCheckIcon } from '@phosphor-icons/react';
import { useMutation } from '@tanstack/react-query';
import { useRef, useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { inspectCert } from '../../lib/api/links';
import { useErrorText } from '../../lib/errors/errors';
import { CertFactsView } from './CertFactsView';
import { siteFormMessages } from './form-messages';
import { canCheckCert, certInputOf, type CertMode } from './site-form';
import type { FieldsProps } from './SiteFields';
import { TextField } from './TextField';

const modes: readonly CertMode[] = ['files', 'upload', 'paste', 'none'];

/** A file read in this browser; only its text goes to Karvan, as PEM. */
function PemFile({ label, onText }: { label: string; onText: (text: string) => void }) {
  const t = useT(siteFormMessages);
  const input = useRef<HTMLInputElement>(null);
  const [name, setName] = useState('');
  const [failed, setFailed] = useState(false);
  return (
    <div className="flex flex-col items-start gap-1.5">
      <input
        ref={input}
        type="file"
        accept=".pem,.crt,.cer,.key,text/plain"
        className="sr-only"
        tabIndex={-1}
        aria-hidden="true"
        onChange={(e) => {
          const file = e.target.files?.[0];
          e.target.value = '';
          if (!file) return;
          file.text().then(
            (text) => {
              setName(file.name);
              setFailed(false);
              onText(text);
            },
            () => setFailed(true),
          );
        }}
      />
      <Button
        icon={<FileArrowUpIcon size={16} />}
        className="pointer-coarse:h-9"
        onClick={() => input.current?.click()}
      >
        {label}
      </Button>
      <span className="text-sm text-kumo-subtle">
        {name ? <bdi dir="ltr">{name}</bdi> : t('noFile')}
      </span>
      {failed ? <span className="text-sm text-kumo-danger">{t('readFailed')}</span> : null}
    </div>
  );
}

/** The chosen way's fields. A saved key never comes back, so an empty key field keeps it. */
function ModeFields({ values, set, errorOf, hasKey }: FieldsProps & { hasKey: boolean }) {
  const t = useT(siteFormMessages);
  const keyHint = hasKey ? t('keyKeep') : undefined;
  switch (values.certMode) {
    case 'files':
      return (
        <>
          <TextField
            id="site-cert-file"
            label={t('certFileLabel')}
            hint={<CodeText text={t('certFileHint')} />}
            error={errorOf('certFile')}
            value={values.certFile}
            onChange={(certFile) => set({ certFile })}
            machine
          />
          <TextField
            id="site-key-file"
            label={t('keyFileLabel')}
            error={errorOf('keyFile')}
            value={values.keyFile}
            onChange={(keyFile) => set({ keyFile })}
            machine
          />
        </>
      );
    case 'upload':
      return (
        <div className="flex flex-col gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <PemFile label={t('chooseCert')} onText={(certPem) => set({ certPem })} />
            <PemFile label={t('chooseKey')} onText={(keyPem) => set({ keyPem })} />
          </div>
          {keyHint ? <p className="text-sm text-kumo-subtle">{keyHint}</p> : null}
          {[errorOf('certPem'), errorOf('keyPem')].filter(Boolean).map((e) => (
            <p key={e} className="text-sm text-kumo-danger">
              {e}
            </p>
          ))}
        </div>
      );
    case 'paste':
      return (
        <>
          <TextField
            id="site-cert-pem"
            label={t('certPemLabel')}
            error={errorOf('certPem')}
            value={values.certPem}
            onChange={(certPem) => set({ certPem })}
            machine
            rows={5}
          />
          <TextField
            id="site-key-pem"
            label={t('keyPemLabel')}
            hint={keyHint ?? t('keyHidden')}
            error={errorOf('keyPem')}
            value={values.keyPem}
            onChange={(keyPem) => set({ keyPem })}
            machine
            rows={5}
          />
        </>
      );
    case 'none':
      return <InlineAlert tone="warning">{t('noneWarning')}</InlineAlert>;
  }
}

/** How the site gets its certificate, and a check of it before saving. */
export function CertFields(props: FieldsProps & { siteId?: number; hasKey: boolean }) {
  const { values, set, siteId } = props;
  const t = useT(siteFormMessages);
  const errorText = useErrorText();
  const check = useMutation({ mutationFn: inspectCert });
  // Facts only show for the certificate they were read from.
  const asked = JSON.stringify(certInputOf(values, siteId));
  const fresh = check.variables && JSON.stringify(check.variables) === asked;
  return (
    <fieldset className="flex flex-col gap-5">
      <Radio.Group
        legend={t('certTitle')}
        appearance="card"
        value={values.certMode}
        onValueChange={(v: CertMode) => set({ certMode: v })}
      >
        {modes.map((m) => (
          <Radio.Item
            key={m}
            value={m}
            label={
              <span className="flex flex-col gap-0.5">
                <span className="font-medium text-kumo-strong">{t(`mode_${m}`)}</span>
                <span className="text-sm text-kumo-subtle">{t(`mode_${m}_hint`)}</span>
              </span>
            }
          />
        ))}
      </Radio.Group>
      <ModeFields {...props} />
      {values.certMode === 'none' ? null : (
        <div className="flex flex-col items-start gap-3">
          <Button
            icon={<SealCheckIcon size={16} />}
            loading={check.isPending}
            disabled={!canCheckCert(values)}
            className="pointer-coarse:h-9"
            onClick={() => check.mutate(certInputOf(values, siteId))}
          >
            {t('check')}
          </Button>
          <div aria-live="polite" className="w-full empty:hidden">
            {fresh && check.data ? <CertFactsView facts={check.data} /> : null}
            {fresh && check.isError ? (
              <InlineAlert>{`${t('checkFailed')} ${errorText(check.error)}`}</InlineAlert>
            ) : null}
          </div>
        </div>
      )}
    </fieldset>
  );
}
