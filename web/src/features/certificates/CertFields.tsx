import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { FileArrowUpIcon, SealCheckIcon } from '../../components/icons/regular';
import { useMutation } from '@tanstack/react-query';
import { useRef, useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { inspectCert } from '../../lib/api/rules';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { CertFactsView } from './CertFactsView';
import { siteFormMessages } from '../rules/form-messages';
import { canCheckCert, certInputOf, type CertMode, type SiteValues } from '../rules/site-form';
import type { FieldsProps } from '../rules/SiteFields';
import { TextField } from '../rules/TextField';

const siteModes: readonly CertMode[] = ['files', 'upload', 'paste', 'none'];

/** The certificate fields a rule and a stored certificate share. */
export type CertValues = Pick<
  SiteValues,
  'certMode' | 'certFile' | 'keyFile' | 'certPem' | 'keyPem'
>;

export type CertField = 'certFile' | 'keyFile' | 'certPem' | 'keyPem';

export interface SourceProps {
  values: CertValues;
  set: (patch: Partial<CertValues>) => void;
  /** The sentence of the field the server refused. */
  errorOf: (f: CertField) => string | undefined;
  hasKey: boolean;
}

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

function PasteFields({ values, set, errorOf, hasKey }: SourceProps) {
  const t = useT(siteFormMessages);
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
        hint={hasKey ? t('keyKeep') : t('keyHidden')}
        error={errorOf('keyPem')}
        value={values.keyPem}
        onChange={(keyPem) => set({ keyPem })}
        machine
        rows={5}
      />
    </>
  );
}

/**
 * The chosen way's fields. Upload reads the files in this browser into the
 * paste fields. A saved key never comes back, so an empty key field keeps it.
 */
export function ModeFields({ values, set, errorOf, hasKey }: SourceProps) {
  const t = useT(siteFormMessages);
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
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <PemFile label={t('chooseCert')} onText={(certPem) => set({ certPem })} />
            <PemFile label={t('chooseKey')} onText={(keyPem) => set({ keyPem })} />
          </div>
          <PasteFields values={values} set={set} errorOf={errorOf} hasKey={hasKey} />
        </>
      );
    case 'paste':
      return <PasteFields values={values} set={set} errorOf={errorOf} hasKey={hasKey} />;
    case 'none':
      return <InlineAlert tone="warning">{t('noneWarning')}</InlineAlert>;
  }
}

/** How Karvan gets a certificate, as radio cards, then the chosen way's fields. */
export function CertSource({
  legend,
  modes,
  ...props
}: SourceProps & { legend: string; modes: readonly CertMode[] }) {
  const t = useT(siteFormMessages);
  return (
    <>
      <Radio.Group
        legend={legend}
        appearance="card"
        value={props.values.certMode}
        onValueChange={(v: CertMode) => props.set({ certMode: v })}
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
    </>
  );
}

/** How the site gets its certificate, and a check of it before saving. */
export function CertFields(props: FieldsProps & { siteId?: number; hasKey: boolean }) {
  const { values, siteId } = props;
  const t = useT(siteFormMessages);
  const errorText = useSubscriptionErrorText();
  const check = useMutation({ mutationFn: inspectCert });
  // Facts only show for the certificate they were read from.
  const asked = JSON.stringify(certInputOf(values, siteId));
  const fresh = check.variables && JSON.stringify(check.variables) === asked;
  return (
    <fieldset className="flex flex-col gap-5">
      <CertSource legend={t('certTitle')} modes={siteModes} {...props} />
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
