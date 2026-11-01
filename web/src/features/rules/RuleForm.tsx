import { Button } from '@cloudflare/kumo/components/button';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { Steps } from '../../components/form/Steps';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import type { BatchGroup } from '../../lib/api/batches';
import { ApiError } from '../../lib/api/client';
import type { Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { DomainPicker, RuleHttpsQuestion } from './DomainPicker';
import { siteFormMessages } from './form-messages';
import { OldPanelPicker } from './OldPanelPicker';
import { RedirectFields } from './RedirectFields';
import {
  canContinueDomain,
  canSubmit,
  groupDefaults,
  isPlainHttp,
  refusedField,
  stepForRefused,
  stepsForRule,
  unregistered,
  withDomains,
  withGroup,
  type RuleField,
  type RuleStep,
  type RuleValues,
} from './rule-form';
import { action, BatchesDetailsFields, Footer, KindQuestion, StepHeading } from './SiteFields';
import { TextField } from './TextField';

/** Add or edit a rule one step at a time: Kind, Source, HTTPS, then Details. */
export function RuleForm({
  initial,
  isNew,
  domains,
  groups,
  saving,
  error,
  onSave,
  onCancel,
}: {
  initial: RuleValues;
  isNew: boolean;
  domains: readonly Domain[];
  groups: readonly BatchGroup[];
  saving: boolean;
  error: unknown;
  onSave: (v: RuleValues, group: BatchGroup | undefined) => void;
  onCancel: () => void;
}) {
  const t = useT(siteFormMessages);
  const errorText = useSubscriptionErrorText();
  const [v, setV] = useState(() =>
    isNew && initial.groupId > 0 ? withGroup(initial, initial.groupId, groups, domains) : initial,
  );
  const set = (patch: Partial<RuleValues>) => setV((old) => ({ ...old, ...patch }));
  const steps = stepsForRule(isNew);
  const [step, setStep] = useState<RuleStep>(() =>
    isNew && initial.groupId === 0 ? 'kind' : 'domain',
  );
  const [answered, setAnswered] = useState<readonly RuleStep[]>(() =>
    isNew && initial.groupId > 0 ? ['kind'] : [],
  );
  const answer = (s: RuleStep) => setAnswered((a) => (a.includes(s) ? a : [...a, s]));
  const at = steps.indexOf(step);
  const group = groups.find((g) => g.id === v.groupId);
  const facts = groupDefaults(group, domains);
  const refused = refusedField(error);
  const [shown, setShown] = useState(error);
  if (error !== shown) {
    setShown(error);
    const target = stepForRefused(refused);
    if (target) setStep(target);
  }
  const reasoned = error instanceof ApiError && typeof error.detail.reason === 'string';
  const errorOf = (f: RuleField) =>
    refused === f ? (reasoned ? errorText(error) : t(`field_${f}`)) : undefined;
  const heading = useRef<HTMLHeadingElement>(null);
  const moved = useRef(false);
  useEffect(() => {
    if (!moved.current) return;
    heading.current?.focus();
  }, [step]);
  const go = (to: RuleStep) => {
    moved.current = true;
    setStep(to);
  };
  const back = () => (at > 0 ? go(steps[at - 1] ?? 'domain') : onCancel());
  const backButton = (
    <Button type="button" className={action} onClick={back}>
      {at > 0 ? t('back') : t('cancel')}
    </Button>
  );
  const label = (s: RuleStep) => t(`step_${s}`);
  const plainSource = !!group && isPlainHttp(group.source.origin);
  let body: ReactNode;
  if (step === 'kind') {
    body = (
      <>
        <KindQuestion
          chosen={answered.includes('kind') ? v.kind : undefined}
          onChoose={(kind) => {
            answer('kind');
            set({ kind });
            go('domain');
          }}
        />
        <Footer>{backButton}</Footer>
      </>
    );
  } else if (step === 'domain') {
    const domainReady = canContinueDomain(v);
    body = (
      <form
        noValidate
        className="flex flex-col gap-6"
        onSubmit={(e) => {
          e.preventDefault();
          if (domainReady) {
            answer('domain');
            go('https');
          }
        }}
      >
        <TextField
          id="site-name"
          label={t('nameLabel')}
          hint={t('nameHint')}
          error={errorOf('name')}
          value={v.name}
          onChange={(name) => set({ name })}
        />
        {v.kind === 'batches' ? (
          <OldPanelPicker
            groups={groups}
            picked={v.groupId}
            error={errorOf('batchIds')}
            onPick={(id) =>
              isNew ? setV((old) => withGroup(old, id, groups, domains)) : set({ groupId: id })
            }
          />
        ) : null}
        <DomainPicker
          domains={domains}
          picked={v.domains}
          missing={v.kind === 'batches' ? unregistered(facts.hosts, domains) : []}
          error={errorOf('domains')}
          onPick={(names) => setV((old) => withDomains(old, names, domains))}
        />
        <Footer>
          {backButton}
          <Button type="submit" variant="primary" disabled={!domainReady} className={action}>
            {t('continue')}
          </Button>
        </Footer>
      </form>
    );
  } else if (step === 'https') {
    body = (
      <>
        <RuleHttpsQuestion
          domains={domains}
          picked={v.domains}
          chosen={answered.includes('https') ? (v.https ? 'with' : 'without') : undefined}
          error={errorOf('https')}
          onChoose={(https) => {
            answer('https');
            set({ https });
            go('details');
          }}
        />
        <Footer>{backButton}</Footer>
      </>
    );
  } else {
    const fields = { values: v, set, errorOf };
    body = (
      <form
        noValidate
        className="flex flex-col gap-6"
        onSubmit={(e) => {
          e.preventDefault();
          onSave(v, group);
        }}
      >
        {v.kind === 'batches' ? (
          <BatchesDetailsFields {...fields} refused={refused} plainSource={plainSource} />
        ) : (
          <RedirectFields {...fields} />
        )}
        {!v.https && v.domains.length > 0 ? (
          <InlineAlert tone="warning">{t('plainWarning')}</InlineAlert>
        ) : null}
        {error && !refused ? (
          <InlineAlert>
            <CodeText text={`${t('saveFailed')} ${errorText(error)}`} />
          </InlineAlert>
        ) : null}
        <Footer>
          {backButton}
          <Button
            type="submit"
            variant="primary"
            loading={saving}
            disabled={!canSubmit(v)}
            className={action}
          >
            {isNew ? t('addSite') : t('save')}
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
        <div className="flex flex-col gap-1">
          <h2
            ref={heading}
            tabIndex={-1}
            className="text-heading font-semibold text-kumo-strong outline-none"
          >
            <StepHeading step={step} kind={v.kind} name={v.name} />
          </h2>
          {step === 'https' || step === 'details' ? (
            <p className="flex flex-wrap items-center gap-x-2 text-sm text-kumo-subtle">
              {v.domains.length > 0 ? (
                <bdi dir="ltr" className="font-mono text-[0.9em]">
                  {v.domains.join(', ')}
                </bdi>
              ) : (
                t('portCert')
              )}
            </p>
          ) : null}
        </div>
        {body}
      </div>
    </div>
  );
}
