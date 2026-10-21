import { useId } from 'react';
import { useT } from '../../../i18n/locale';
import { overviewMessages } from '../messages';
import { LinksCard } from './LinksCard';
import { MovedCard } from './MovedCard';
import { SafeModeCard } from './SafeModeCard';

/** Three cards of different shapes: users moved, Safe Mode and old links. */
export function Summary() {
  const t = useT(overviewMessages);
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex flex-col gap-4">
      <h2 id={id} className="text-heading font-semibold text-kumo-strong">
        {t('summaryTitle')}
      </h2>
      <div className="grid gap-4 md:grid-cols-3">
        <MovedCard />
        <SafeModeCard />
        <LinksCard />
      </div>
    </section>
  );
}
