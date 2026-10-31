import { Button } from '@cloudflare/kumo/components/button';
import { Dialog } from '@cloudflare/kumo/components/dialog';
import { createContext, use, useState, type ReactNode } from 'react';
import { useT } from '../../i18n/locale';
import { wizardMessages } from './messages';
import { usePlan, type PlanApi } from './use-plan';

/** How the moved admin gets its password: the reseller's own (or one Karvan makes), or a new one. */
export interface AdminPassword {
  choice: 'keep' | 'new';
  value: string;
}

interface Wizard {
  jobId: string;
  plan: PlanApi;
  /** Lives in this state only, until the wizard closes. */
  password: AdminPassword;
  setPassword: (p: AdminPassword) => void;
}

const WizardContext = createContext<Wizard | null>(null);

/** The open job's steps share its plan, and the admin password the operator set. */
export function useWizard(): Wizard {
  const w = use(WizardContext);
  if (!w) throw new Error('useWizard outside WizardProvider');
  return w;
}

/** Asks what to do with an edit that met a second conflict. */
function ConflictDialog({ plan }: { plan: PlanApi }) {
  const t = useT(wizardMessages);
  return (
    <Dialog.Root open={plan.conflict} onOpenChange={(open) => !open && plan.dismiss()}>
      <Dialog className="flex flex-col gap-4 p-6" size="base">
        <Dialog.Title className="text-heading font-semibold">{t('conflictTitle')}</Dialog.Title>
        <Dialog.Description className="text-kumo-subtle">{t('conflictBody')}</Dialog.Description>
        <div className="flex flex-wrap justify-end gap-2">
          <Button className="pointer-coarse:h-9" onClick={plan.dismiss}>
            {t('conflictKeep')}
          </Button>
          <Button variant="primary" className="pointer-coarse:h-9" onClick={plan.replay}>
            {t('conflictReplay')}
          </Button>
        </div>
      </Dialog>
    </Dialog.Root>
  );
}

export function WizardProvider({ jobId, children }: { jobId: string; children: ReactNode }) {
  const plan = usePlan(jobId);
  const [password, setPassword] = useState<AdminPassword>({ choice: 'keep', value: '' });
  return (
    <WizardContext value={{ jobId, plan, password, setPassword }}>
      {children}
      <ConflictDialog plan={plan} />
    </WizardContext>
  );
}
