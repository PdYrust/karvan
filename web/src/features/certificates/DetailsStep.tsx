import { Button } from '@cloudflare/kumo/components/button';
import { Radio } from '@cloudflare/kumo/components/radio';
import { useId } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { ScanResult } from '../../lib/api/certificates';
import { AcmeFields } from './AcmeFields';
import { ModeFields, type CertField } from './CertFields';
import { notScanned } from './cert-view';
import type { DomainFormValues, ServerCert } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { Mono } from './parts';
import { ServerCerts } from './ServerCerts';

function Skipped({ scan }: { scan: ScanResult }) {
  const t = useT(domainFormMessages);
  const id = useId();
  const skipped = notScanned(scan);
  if (skipped.length === 0) return null;
  return (
    <section aria-labelledby={id} className="flex flex-col gap-1.5">
      <h3 id={id} className="text-sm font-medium text-kumo-strong">
        {t('notScanned')}
      </h3>
      <ul className="flex flex-col gap-1 text-sm text-kumo-subtle">
        {skipped.map((p) => (
          <li key={p}>
            <Mono text={p} />
          </li>
        ))}
      </ul>
    </section>
  );
}

/** A certificate the operator has: one on this server, typed paths, or upload or paste. */
function OwnFields({
  values: v,
  set,
  certs,
  best,
  scan,
  scanError,
  errorOf,
}: {
  values: DomainFormValues;
  set: (patch: Partial<DomainFormValues>) => void;
  certs: readonly ServerCert[];
  best?: ServerCert;
  scan?: ScanResult;
  scanError?: string;
  errorOf: (f: CertField) => string | undefined;
}) {
  const t = useT(domainFormMessages);
  const from = v.way === 'paste' ? 'paste' : 'server';
  const ghost = 'self-start pointer-coarse:h-9';
  return (
    <>
      <Radio.Group
        legend={t('ownFrom')}
        orientation="horizontal"
        value={from}
        onValueChange={(f: string) =>
          set(
            f === 'paste' ? { way: 'paste' } : { way: 'found', pick: v.pick ?? best?.pick ?? null },
          )
        }
        className="[&>div:first-child]:sr-only [&>div:last-child]:gap-x-6"
      >
        <Radio.Item value="server" label={t('own_server')} />
        <Radio.Item value="paste" label={t('own_paste')} />
      </Radio.Group>
      {scanError ? <InlineAlert>{scanError}</InlineAlert> : null}
      {v.way === 'found' ? (
        <>
          <ServerCerts
            certs={certs}
            pick={v.pick}
            best={best?.key}
            onPick={(pick) => set({ pick })}
          />
          <Button variant="ghost" className={ghost} onClick={() => set({ way: 'file' })}>
            {t('typePaths')}
          </Button>
          {scan && !certs.some((c) => c.covers) ? <Skipped scan={scan} /> : null}
        </>
      ) : null}
      {v.way === 'file' ? (
        <>
          <ModeFields
            values={{ ...v, certMode: 'files' }}
            set={set}
            errorOf={errorOf}
            hasKey={false}
          />
          <Button variant="ghost" className={ghost} onClick={() => set({ way: 'found' })}>
            {t('pickListed')}
          </Button>
        </>
      ) : null}
      {v.way === 'paste' ? (
        <ModeFields
          values={{ ...v, certMode: 'upload' }}
          set={set}
          errorOf={errorOf}
          hasKey={false}
        />
      ) : null}
    </>
  );
}

/** The last step: what the chosen way needs before Add domain. */
export function DetailsStep({
  needsAgreement,
  ...own
}: Parameters<typeof OwnFields>[0] & { needsAgreement: boolean }) {
  const t = useT(domainFormMessages);
  const { values: v, set } = own;
  if (v.way === 'acme-http' || v.way === 'acme-dns') {
    return (
      <>
        <p>{t(`detail_${v.way}`)}</p>
        {needsAgreement ? <AcmeFields agreeTos={v.agreeTos} set={set} /> : null}
      </>
    );
  }
  if (v.way === 'none') {
    return (
      <ModeFields
        values={{ ...v, certMode: 'none' }}
        set={set}
        errorOf={own.errorOf}
        hasKey={false}
      />
    );
  }
  return <OwnFields {...own} />;
}

/** A refused save in words; a port problem offers the DNS way. */
export function SaveError({ text, onDns }: { text: string; onDns?: () => void }) {
  const t = useT(domainFormMessages);
  return (
    <InlineAlert>
      <span className="flex flex-col items-start gap-3">
        <CodeText text={text} />
        {onDns ? (
          <Button className="pointer-coarse:h-9" onClick={onDns}>
            {t('useDns')}
          </Button>
        ) : null}
      </span>
    </InlineAlert>
  );
}
