import { useT } from '../../i18n/locale';
import { wizardMessages } from './messages';

/** Holds the place of a step that is not built yet. */
export function StepSoon() {
  const t = useT(wizardMessages);
  return (
    <p className="max-w-[720px] rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">
      {t('stepSoon')}
    </p>
  );
}
