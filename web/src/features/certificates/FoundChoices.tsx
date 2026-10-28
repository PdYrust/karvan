import { Radio } from '@cloudflare/kumo/components/radio';
import { useQuery } from '@tanstack/react-query';
import { useId } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { useT } from '../../i18n/locale';
import { certificateKeys, scanCertificates, type Certificate } from '../../lib/api/certificates';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useFormat } from '../../lib/preferences/region';
import { domainWarningTone, notScanned, sortedWarnings } from './cert-view';
import { domainName, foundChoices, type FoundChoice, type FoundPick } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { certFactMessages } from './fact-messages';
import { Mono } from './parts';

function ChoiceLabel({ choice }: { choice: FoundChoice }) {
  const t = useT(domainFormMessages);
  const tc = useT(certFactMessages);
  const fmt = useFormat();
  const { facts } = choice;
  const stored = choice.pick.kind === 'stored';
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      {stored ? (
        <bdi className="font-medium text-kumo-strong">{choice.title}</bdi>
      ) : (
        <Mono text={choice.title} className="font-medium text-kumo-strong" />
      )}
      <span className="flex flex-wrap gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
        {stored ? <span>{t('storedLabel')}</span> : null}
        {facts.domains.length > 0 ? <Mono text={facts.domains.join(', ')} /> : null}
        {facts.notAfter ? (
          <span>{t('endsOn', { date: fmt.date(new Date(facts.notAfter)) })}</span>
        ) : null}
      </span>
      {sortedWarnings(facts.warnings).map((w) => (
        <span key={w} className="text-sm">
          <StatusLabel tone={domainWarningTone(w)}>{tc(`warn_${w}`)}</StatusLabel>
        </span>
      ))}
    </span>
  );
}

/**
 * The certificates that cover the name: stored ones, then those the read-only
 * scan found (D51). The scan runs once when this way is picked; the places it
 * skipped are listed as not scanned, with no reason (D59).
 */
export function FoundChoices({
  name,
  certificates,
  pick,
  onPick,
}: {
  name: string;
  certificates: readonly Certificate[];
  pick: FoundPick | null;
  onPick: (p: FoundPick) => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useSubscriptionErrorText();
  const skippedId = useId();
  const scan = useQuery({
    queryKey: [...certificateKeys.all, 'scan'],
    queryFn: scanCertificates,
    staleTime: Infinity,
    retry: false,
  });
  if (!domainName(name)) return <p className="text-sm text-kumo-subtle">{t('noName')}</p>;
  const choices = foundChoices(name, certificates, scan.data);
  const value = choices.find((c) => JSON.stringify(c.pick) === JSON.stringify(pick))?.key ?? '';
  const skipped = scan.data ? notScanned(scan.data) : [];
  return (
    <div className="flex flex-col gap-4" aria-live="polite">
      {choices.length > 0 ? (
        <Radio.Group
          legend={t('way_found')}
          value={value}
          onValueChange={(key: string) => {
            const c = choices.find((x) => x.key === key);
            if (c) onPick(c.pick);
          }}
          className="[&_legend]:sr-only"
        >
          {choices.map((c) => (
            <Radio.Item key={c.key} value={c.key} label={<ChoiceLabel choice={c} />} />
          ))}
        </Radio.Group>
      ) : null}
      {scan.isPending ? (
        <div role="status" aria-label={t('scanning')} className="flex flex-col gap-3">
          <Bone className="h-12 w-full" />
          <Bone className="h-12 w-full" />
        </div>
      ) : null}
      {scan.isError ? (
        <InlineAlert>{`${t('scanFailed')} ${errorText(scan.error)}`}</InlineAlert>
      ) : null}
      {scan.isSuccess && choices.length === 0 ? (
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-sm text-kumo-subtle">
          {t('foundNone')}
        </p>
      ) : null}
      {skipped.length > 0 ? (
        <section aria-labelledby={skippedId} className="flex flex-col gap-1.5">
          <h3 id={skippedId} className="text-sm font-medium text-kumo-strong">
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
      ) : null}
    </div>
  );
}
