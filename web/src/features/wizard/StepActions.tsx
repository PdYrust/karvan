import { Button } from '@cloudflare/kumo/components/button';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { useT } from '../../i18n/locale';
import { wizardMessages } from './messages';
import { stepPath, type WizardStep } from './steps';

/**
 * The bottom of a step: what stands in the way, if anything, then Back and
 * the way on, at the end.
 */
export function StepActions({
  jobId,
  back,
  next,
  children,
}: {
  jobId: string;
  back?: WizardStep;
  next: WizardStep;
  /** A short note before the buttons, such as how many problems still need a fix. */
  children?: ReactNode;
}) {
  const t = useT(wizardMessages);
  const navigate = useNavigate();
  return (
    <div className="flex flex-col gap-3 border-t border-kumo-hairline pt-5 sm:flex-row sm:items-center sm:justify-end">
      {children ? <div className="text-sm text-kumo-subtle sm:me-auto">{children}</div> : null}
      <div className="flex gap-2 sm:justify-end">
        {back ? (
          <Button
            className="flex-1 justify-center sm:flex-none pointer-coarse:h-9"
            onClick={() => void navigate(stepPath(back, jobId))}
          >
            {t('back')}
          </Button>
        ) : null}
        <Button
          variant="primary"
          className="flex-1 justify-center sm:flex-none pointer-coarse:h-9"
          onClick={() => void navigate(stepPath(next, jobId))}
        >
          {t('continue')}
        </Button>
      </div>
    </div>
  );
}
