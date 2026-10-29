import { Button } from '@cloudflare/kumo/components/button';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Steps } from '../../components/form/Steps';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { certificateKeys, scanCertificates, type Certificate } from '../../lib/api/certificates';
import type { CertChoice } from '../../lib/api/domains';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { useNow } from '../overview/use-now';
import { TextField } from '../rules/TextField';
import { CertStep, type Looked } from './CertStep';
import {
  bestChoice,
  choiceOf,
  domainName,
  foundChoices,
  isPortError,
  missingOf,
  recommended,
  refusedDomainField,
  stepAfter,
  type DomainFormValues,
  type Step,
} from './domain-form';
import { domainFormMessages } from './domain-messages';

const actions = 'flex-1 justify-center sm:flex-none pointer-coarse:h-9';

/**
 * Add domain as two steps, the name then its certificate, or Change certificate
 * (`isNew` false) as the certificate step alone. `except` is the domain's own
 * certificate, which the change page does not offer again.
 */
export function DomainForm({
  initial,
  isNew,
  certificates,
  except,
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
  except?: number;
  needsAgreement: boolean;
  saving: boolean;
  error: unknown;
  onSave: (name: string, choice: CertChoice) => void;
  onReset: () => void;
  onCancel: () => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const now = useNow();
  const missingId = useId();
  const [v, setV] = useState(initial);
  const set = (patch: Partial<DomainFormValues>) => setV((old) => ({ ...old, ...patch }));
  const [step, setStep] = useState<Step>(isNew ? 'domain' : 'certificate');
  const [looking, setLooking] = useState(!isNew);
  const scan = useQuery({
    queryKey: [...certificateKeys.all, 'scan'],
    queryFn: scanCertificates,
    staleTime: Infinity,
    retry: false,
    enabled: looking,
  });
  const looked: Looked | undefined =
    looking && !scan.isPending ? { result: scan.data, error: scan.error ?? undefined } : undefined;
  const choices = foundChoices(v.name, certificates, scan.data, except);
  const best = bestChoice(choices, now);
  const rec = recommended(v.name, choices, now);
  // The answer is chosen once the scan answered, again only for another name.
  const [chosenFor, setChosenFor] = useState<string>();
  if (looked && chosenFor !== domainName(v.name)) {
    setChosenFor(domainName(v.name));
    setV((old) => ({ ...old, ...rec }));
  }
  const refused = refusedDomainField(error);
  const [shownError, setShownError] = useState(error);
  if (error !== shownError) {
    setShownError(error);
    if (refused === 'name') setStep('domain');
  }
  const reason = error ? errorText(error) : '';
  const errorOf = (f: string) => (refused === f ? reason : undefined);
  const heading = useRef<HTMLHeadingElement>(null);
  const focusNext = useRef<'heading' | 'field' | undefined>(undefined);
  useEffect(() => {
    if (focusNext.current === 'heading') heading.current?.focus();
    if (focusNext.current === 'field') document.getElementById('domain-name')?.focus();
    focusNext.current = undefined;
  }, [step]);
  useEffect(() => {
    if (refused === 'name') document.getElementById('domain-name')?.focus();
  }, [refused, error]);
  const go = (to: Step) => {
    focusNext.current = to === 'certificate' ? 'heading' : 'field';
    setStep(to);
  };
  const missing = missingOf(v, needsAgreement);
  const labels = [t('step_domain'), t('step_certificate')] as const;
  const current = step === 'domain' ? 0 : 1;
  const steps = labels.map((label) => ({ label }));
  return (
    <div className="flex max-w-[720px] flex-col gap-8">
      {isNew ? (
        <Steps
          steps={steps}
          current={current}
          label={t('stepsLabel')}
          progress={t('stepOf', { n: current + 1, total: 2, name: labels[current] })}
        />
      ) : null}
      {step === 'domain' ? (
        <form
          noValidate
          className="flex flex-col gap-8"
          onSubmit={(e) => {
            e.preventDefault();
            if (stepAfter(v.name) === 'domain') return;
            setLooking(true);
            go('certificate');
          }}
        >
          <TextField
            id="domain-name"
            label={t('nameLabel')}
            hint={<CodeText text={t('nameHint')} />}
            error={errorOf('name')}
            value={v.name}
            onChange={(name) => set({ name })}
            placeholder="sub.example.com"
            machine
          />
          <div className="flex justify-end border-t border-kumo-hairline pt-5">
            <Button
              type="submit"
              variant="primary"
              disabled={stepAfter(v.name) === 'domain'}
              className="w-full justify-center sm:w-auto pointer-coarse:h-9"
            >
              {t('continue')}
            </Button>
          </div>
        </form>
      ) : (
        <form
          noValidate
          className="flex flex-col gap-8"
          onSubmit={(e) => {
            e.preventDefault();
            const choice = choiceOf(v, needsAgreement);
            if (choice) onSave(v.name, choice);
          }}
        >
          <CertStep
            headingRef={isNew ? heading : undefined}
            looked={looked}
            values={v}
            set={set}
            choices={choices}
            best={best?.key}
            needsAgreement={needsAgreement}
            errorOf={errorOf}
          />
          {error && !refused ? (
            <InlineAlert>
              <span className="flex flex-col items-start gap-3">
                <CodeText text={`${t('failed')} ${reason}`} />
                {isPortError(error) ? (
                  <Button
                    className="pointer-coarse:h-9"
                    onClick={() => {
                      set({ way: 'acme-dns' });
                      onReset();
                    }}
                  >
                    {t('useDns')}
                  </Button>
                ) : null}
              </span>
            </InlineAlert>
          ) : null}
          <div className="flex flex-col gap-3 border-t border-kumo-hairline pt-5 sm:flex-row sm:items-center sm:justify-end">
            {looked && missing ? (
              <p id={missingId} className="text-sm text-kumo-subtle sm:me-auto">
                {t(`need_${missing}`)}
              </p>
            ) : null}
            <div className="flex gap-2 sm:justify-end">
              <Button
                type="button"
                className={actions}
                onClick={isNew ? () => go('domain') : onCancel}
              >
                {isNew ? t('back') : t('cancel')}
              </Button>
              <Button
                type="submit"
                variant="primary"
                loading={saving}
                disabled={!looked || missing !== undefined}
                aria-describedby={looked && missing ? missingId : undefined}
                className={actions}
              >
                {isNew ? t('addSubmit') : t('save')}
              </Button>
            </div>
          </div>
        </form>
      )}
    </div>
  );
}
