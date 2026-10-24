import type { WizardKey } from './messages';

/** The wizard's steps in order; every step but Connect lives under the job. */
export const wizardSteps = [
  'connect',
  'summary',
  'compare',
  'changes',
  'preview',
  'start',
] as const;

export type WizardStep = (typeof wizardSteps)[number];

export const stepWords: Record<WizardStep, WizardKey> = {
  connect: 'stepConnect',
  summary: 'stepSummary',
  compare: 'stepCompare',
  changes: 'stepChanges',
  preview: 'stepPreview',
  start: 'stepStart',
};

export const newMigrationPath = '/migrations/new';

/** The address of a step; Connect has one address, before a job exists. */
export function stepPath(step: WizardStep, jobId?: string): string {
  if (step === 'connect' || !jobId) return newMigrationPath;
  return `/migrations/${encodeURIComponent(jobId)}/${step}`;
}

/** The step a job page shows, from the last part of its path. */
export function stepOf(pathname: string): WizardStep {
  const last = pathname.replace(/\/+$/, '').split('/').at(-1) ?? '';
  return (wizardSteps as readonly string[]).includes(last) ? (last as WizardStep) : 'summary';
}
