import { useId, type Ref } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Bone } from '../../components/feedback/PageSkeleton';
import { Reveal } from '../../components/feedback/Reveal';
import { useT } from '../../i18n/locale';
import type { ScanResult } from '../../lib/api/certificates';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { AcmeFields } from './AcmeFields';
import { CertChoices } from './CertChoices';
import { ModeFields, type CertField, type CertMode } from './CertFields';
import { notScanned } from './cert-view';
import { otherWays, type DomainFormValues, type FoundChoice, type Way } from './domain-form';
import { domainFormMessages } from './domain-messages';
import { Mono } from './parts';
import { OtherWays } from './OtherWays';

const modeOf: Partial<Record<Way, CertMode>> = {
  upload: 'upload',
  paste: 'paste',
  none: 'none',
  file: 'files',
};

/** What the scan answered, once it answered; `result` is missing when it failed. */
export interface Looked {
  result?: ScanResult;
  error?: unknown;
}

function Skeleton() {
  const t = useT(domainFormMessages);
  return (
    <div role="status" aria-label={t('scanning')} className="flex flex-col gap-4">
      <Bone className="h-5 w-72 max-w-full" />
      <Bone className="h-16 w-full" />
      <Bone className="h-8 w-28" />
    </div>
  );
}

/** What Karvan found for the name, in one quiet line. */
function FoundLine({ choices, best }: { choices: readonly FoundChoice[]; best?: string }) {
  const t = useT(domainFormMessages);
  let text = t('foundNone');
  if (choices.length > 0) text = best ? t('foundSome', { count: choices.length }) : t('noneServes');
  return (
    <p role="status" className="text-sm text-kumo-subtle">
      {text}
    </p>
  );
}

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

/**
 * The certificate step: the heading on Add domain, then once the scan answered
 * the chosen answer, Other ways and the chosen way's fields.
 */
export function CertStep({
  headingRef,
  looked,
  values: v,
  set,
  choices,
  best,
  needsAgreement,
  errorOf,
}: {
  /** Add domain only: the heading focus moves to after Continue. */
  headingRef?: Ref<HTMLHeadingElement>;
  looked: Looked | undefined;
  values: DomainFormValues;
  set: (patch: Partial<DomainFormValues>) => void;
  choices: readonly FoundChoice[];
  best?: string;
  needsAgreement: boolean;
  errorOf: (f: CertField) => string | undefined;
}) {
  const t = useT(domainFormMessages);
  const errorText = useSubscriptionErrorText();
  const load = useFirstLoad(looked);
  const mode = modeOf[v.way];
  const pickWay = (way: Way) => {
    const first = choices.find((c) => c.key === best) ?? choices[0];
    set(way === 'found' ? { way, pick: first?.pick ?? null } : { way });
  };
  return (
    <div className="flex flex-col gap-5">
      {headingRef || load.data ? (
        <div className="flex flex-col gap-1">
          {headingRef ? (
            <h2
              ref={headingRef}
              tabIndex={-1}
              className="text-heading font-semibold text-kumo-strong outline-none"
            >
              {t('secureTitle', { name: '' })}
              <bdi dir="ltr">{v.name.trim()}</bdi>
            </h2>
          ) : null}
          {load.data ? (
            <Reveal on={load.reveal}>
              <FoundLine choices={choices} best={best} />
            </Reveal>
          ) : null}
        </div>
      ) : null}
      {load.data ? (
        <Reveal on={load.reveal}>
          <div className="flex flex-col gap-4">
            {load.data.error ? (
              <InlineAlert>{`${t('scanFailed')} ${errorText(load.data.error)}`}</InlineAlert>
            ) : null}
            <CertChoices
              way={v.way}
              choices={choices}
              pick={v.pick}
              best={best}
              onPick={(pick) => set({ pick })}
            />
            {v.way.startsWith('acme') && needsAgreement ? (
              <AcmeFields agreeTos={v.agreeTos} set={set} />
            ) : null}
            {mode ? (
              <ModeFields
                values={{ ...v, certMode: mode }}
                set={set}
                errorOf={errorOf}
                hasKey={false}
              />
            ) : null}
            <OtherWays ways={otherWays(v.way, v.name, choices.length > 0)} onWay={pickWay} />
            {choices.length === 0 && load.data.result ? <Skipped scan={load.data.result} /> : null}
          </div>
        </Reveal>
      ) : (
        <Skeleton />
      )}
    </div>
  );
}
