import { Button } from '@cloudflare/kumo/components/button';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { Steps } from '../../components/form/Steps';
import { useT } from '../../i18n/locale';
import { certificateKeys, scanCertificates, type Certificate } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { useFirstLoad } from '../../lib/motion/use-first-load';
import { useNow } from '../overview/use-now';
import { DetailsStep, SaveError } from './DetailsStep';
import {
  bestCert,
  choiceOf,
  isPortError,
  missingOf,
  recommendedSource,
  refusedDomainField,
  serverCerts,
  stepsFor,
  wayFor,
  type DomainFormValues,
  type Source,
  type Step,
} from './domain-form';
import { domainFormMessages } from './domain-messages';
import { Mono } from './parts';
import { action, Footer, HttpsQuestion, NameStep, StepTitle, SourceQuestion } from './Questions';

/**
 * Add domain one question at a time: the name, with or without a certificate,
 * which one, then what that way needs. A press on an answer goes on by itself.
 * Change certificate (`isNew` false) starts at the second question.
 */
export function DomainForm({
  initial,
  isNew,
  certificates,
  needsAgreement,
  saving,
  error,
  onSave,
  onReset,
  onCancel,
}: {
  initial: DomainFormValues;
  isNew: boolean;
  certificates: readonly Certificate[];
  needsAgreement: boolean;
  saving: boolean;
  error: unknown;
  onSave: (name: string, choice: CertChoice) => void;
  onReset: () => void;
  onCancel: () => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const scanText = useSubscriptionErrorText();
  const now = useNow();
  const noteId = useId();
  const [v, setV] = useState(initial);
  const set = (patch: Partial<DomainFormValues>) => setV((old) => ({ ...old, ...patch }));
  const [step, setStep] = useState<Step>(isNew ? 'domain' : 'https');
  // The questions already answered keep their answer tinted after Back.
  const [answered, setAnswered] = useState<readonly Step[]>([]);
  const answer = (s: Step) => setAnswered((a) => (a.includes(s) ? a : [...a, s]));
  const scan = useQuery({
    queryKey: [...certificateKeys.all, 'scan'],
    queryFn: scanCertificates,
    staleTime: Infinity,
    retry: false,
    enabled: step !== 'domain',
  });
  const load = useFirstLoad(scan.isPending ? undefined : true);
  const certs = serverCerts(v.name, certificates, scan.data);
  const best = bestCert(certs, now);
  const steps = stepsFor(isNew, v.https);
  const at = steps.indexOf(step);
  const refused = refusedDomainField(error);
  const [shown, setShown] = useState(error);
  if (error !== shown) {
    setShown(error);
    if (refused === 'name') setStep('domain');
  }
  const reason = error ? errorText(error) : '';
  const errorOf = (f: string) => (refused === f ? reason : undefined);
  const heading = useRef<HTMLHeadingElement>(null);
  const moved = useRef(false);
  useEffect(() => {
    if (!moved.current) return;
    if (step === 'domain') document.getElementById('domain-name')?.focus();
    else heading.current?.focus();
  }, [step]);
  const go = (to: Step) => {
    moved.current = true;
    setStep(to);
  };
  const back = () => (at > 0 ? go(steps[at - 1] ?? 'domain') : onCancel());
  const backButton = (
    <Button type="button" className={action} onClick={back}>
      {at > 0 ? t('back') : t('cancel')}
    </Button>
  );
  const source: Source | undefined =
    v.way === 'acme-http' || v.way === 'acme-dns' ? v.way : v.way === 'none' ? undefined : 'own';
  const rec = recommendedSource(v.name, best);
  const missing = missingOf(v, needsAgreement);
  const label = (s: Step) => t(`step_${s}`);
  let body: ReactNode;
  if (step === 'domain') {
    body = (
      <NameStep
        name={v.name}
        error={errorOf('name')}
        onName={(name) => set({ name })}
        onContinue={() => go('https')}
      />
    );
  } else if (step === 'https') {
    body = (
      <>
        <HttpsQuestion
          chosen={answered.includes('https') ? v.https : undefined}
          onChoose={(https) => {
            answer('https');
            if (https === 'without') {
              set({ https, way: 'none', pick: null });
              go('details');
            } else {
              set({ https, ...(v.way === 'none' ? wayFor(rec, best) : {}) });
              go('source');
            }
          }}
        />
        <Footer>{backButton}</Footer>
      </>
    );
  } else if (step === 'source') {
    body = (
      <>
        <SourceQuestion
          name={v.name}
          ready={load.data !== undefined}
          found={certs.filter((c) => c.covers).length}
          recommended={rec}
          chosen={answered.includes('source') ? source : undefined}
          onChoose={(s) => {
            answer('source');
            set(s === source && s === 'own' ? {} : wayFor(s, best));
            go('details');
          }}
        />
        <Footer>{backButton}</Footer>
      </>
    );
  } else {
    body = (
      <form
        noValidate
        className="flex flex-col gap-8"
        onSubmit={(e) => {
          e.preventDefault();
          const choice = choiceOf(v, needsAgreement);
          if (choice) onSave(v.name, choice);
        }}
      >
        <div className="flex flex-col gap-4">
          <DetailsStep
            values={v}
            set={set}
            certs={certs}
            best={best}
            scan={scan.data}
            scanError={scan.isError ? `${t('scanFailed')} ${scanText(scan.error)}` : undefined}
            errorOf={errorOf}
            needsAgreement={needsAgreement}
          />
        </div>
        {error && !refused ? (
          <SaveError
            text={`${t('failed')} ${reason}`}
            onDns={
              isPortError(error)
                ? () => {
                    set({ way: 'acme-dns' });
                    onReset();
                  }
                : undefined
            }
          />
        ) : null}
        <Footer
          note={
            missing ? (
              <p id={noteId} className="text-sm text-kumo-subtle sm:me-auto">
                {t(`need_${missing}`)}
              </p>
            ) : undefined
          }
        >
          {backButton}
          <Button
            type="submit"
            variant="primary"
            loading={saving}
            disabled={missing !== undefined}
            aria-describedby={missing ? noteId : undefined}
            className={action}
          >
            {isNew ? t('addSubmit') : t('save')}
          </Button>
        </Footer>
      </form>
    );
  }
  return (
    <div className="flex max-w-[720px] flex-col gap-8">
      <Steps
        steps={steps.map((s) => ({ label: label(s) }))}
        current={at}
        label={t('stepsLabel')}
        progress={t('stepOf', { n: at + 1, total: steps.length, name: label(step) })}
      />
      <div className="flex flex-col gap-5">
        {step === 'domain' ? null : (
          <div className="flex flex-col gap-1">
            <h2
              ref={heading}
              tabIndex={-1}
              className="text-heading font-semibold text-kumo-strong outline-none"
            >
              <StepTitle step={step} way={v.way} name={v.name} />
            </h2>
            {step === 'https' || !isNew ? null : (
              <Mono text={v.name.trim()} className="text-sm text-kumo-subtle" />
            )}
          </div>
        )}
        {body}
      </div>
    </div>
  );
}
