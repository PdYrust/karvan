import { Button } from '@cloudflare/kumo/components/button';
import { WarningIcon } from '@phosphor-icons/react';
import { Fragment, useId } from 'react';
import { FactList } from '../../../components/layout/FactList';
import { useT } from '../../../i18n/locale';
import { useFormat } from '../../../lib/preferences/region';
import { fingerprintPairs, type CertFacts } from './form';
import { connectMessages } from './messages';

/**
 * A certificate Karvan does not trust, with what the operator needs to
 * compare it against the old server. Trusting it pins exactly this one.
 */
export function CertificateCard({
  cert,
  onTrust,
  pending,
}: {
  cert: CertFacts;
  onTrust: () => void;
  pending?: boolean;
}) {
  const t = useT(connectMessages);
  const fmt = useFormat();
  const id = useId();
  const expires = cert.notAfter ? new Date(cert.notAfter) : null;
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col gap-4 rounded-lg bg-kumo-base p-4 ring ring-kumo-warning/40 sm:p-5"
    >
      <div className="flex items-start gap-3">
        <WarningIcon
          size={20}
          weight="bold"
          aria-hidden="true"
          className="mt-0.5 shrink-0 text-kumo-warning"
        />
        <div className="flex flex-col gap-1">
          <h3 id={id} className="font-semibold text-kumo-strong">
            {t('certTitle')}
          </h3>
          <p className="text-sm text-kumo-subtle">{t('certHint')}</p>
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="text-sm text-kumo-subtle">{t('certFingerprint')}</span>
        {/* Start aligned in full rows, so it reads like the browser's own view of it. */}
        <code
          dir="ltr"
          className="rounded-md bg-kumo-tint px-3 py-2 text-start font-mono text-[0.9em] text-kumo-default"
        >
          {/* Lines break only between pairs. */}
          {fingerprintPairs(cert.sha256)
            .split(':')
            .map((pair, i) => (
              <Fragment key={i}>
                {i > 0 ? ':' : null}
                {i > 0 ? <wbr /> : null}
                {pair}
              </Fragment>
            ))}
        </code>
      </div>
      <FactList
        facts={[
          ...(cert.subject
            ? [{ label: t('certSubject'), value: <bdi dir="ltr">{cert.subject}</bdi> }]
            : []),
          ...(expires && !Number.isNaN(expires.getTime())
            ? [{ label: t('certExpires'), value: fmt.dateTime(expires) }]
            : []),
        ]}
      />
      <div className="flex justify-end">
        <Button loading={pending} onClick={onTrust} className="pointer-coarse:h-9">
          {t('certTrust')}
        </Button>
      </div>
    </section>
  );
}
