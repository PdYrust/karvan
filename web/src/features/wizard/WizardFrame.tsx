import type { ReactNode } from 'react';
import { Steps } from '../../components/form/Steps';
import { PageHeader } from '../../components/layout/PageHeader';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { wizardMessages } from './messages';
import { stepPath, stepWords, wizardSteps, type WizardStep } from './steps';

/** The wizard's header and step list around one step. */
export function WizardFrame({
  step,
  jobId,
  description,
  children,
}: {
  step: WizardStep;
  /** Once a job exists, the steps before this one link back. */
  jobId?: string;
  description?: ReactNode;
  children: ReactNode;
}) {
  const t = useT(wizardMessages);
  const current = wizardSteps.indexOf(step);
  const name = t(stepWords[step]);
  usePageTitle(`${name}: ${t('title')}`);
  const steps = wizardSteps.map((s, i) => ({
    label: t(stepWords[s]),
    // Connect cannot be opened again: the job holds the login it was given.
    to: jobId && i > 0 && i < current ? stepPath(s, jobId) : undefined,
  }));
  return (
    <div className="flex flex-col">
      <PageHeader eyebrow={t('eyebrow')} title={t('title')} description={description} />
      <div className="flex max-w-[1400px] flex-col gap-8 p-4 sm:p-8">
        <Steps
          steps={steps}
          current={current}
          label={t('stepsLabel')}
          progress={t('stepOf', { n: current + 1, total: steps.length, name })}
        />
        {children}
      </div>
    </div>
  );
}
