import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { ArrowsClockwiseIcon } from '@phosphor-icons/react';
import { GirihPattern } from '../../components/brand/GirihPattern';
import { Mark } from '../../components/brand/Mark';
import { Section } from '../../components/layout/Section';
import { useT } from '../../i18n/locale';
import { overviewMessages } from './messages';

/** Panels found on this server. Discovery lands in phase 3; until then this is the empty state. */
export function PanelsSection({
  onScan,
  scanning,
}: {
  onScan: () => Promise<unknown>;
  scanning: boolean;
}) {
  const t = useT(overviewMessages);
  const toasts = useKumoToastManager();

  async function scan() {
    await onScan();
    toasts.add({ title: t('scanned') });
  }

  return (
    <Section title={t('panelsTitle')}>
      <div className="relative overflow-hidden px-5 py-12 sm:py-16">
        <GirihPattern className="pointer-events-none absolute inset-0 text-kumo-hairline [mask-image:radial-gradient(closest-side,black,transparent)]" />
        <div className="relative mx-auto flex max-w-md flex-col items-center gap-3 text-center">
          <Mark size={56} className="opacity-90" />
          <h3 className="mt-2 text-heading font-semibold text-kumo-strong">
            {t('panelsEmptyTitle')}
          </h3>
          <p className="text-balance text-kumo-subtle">{t('panelsEmptyBody')}</p>
          <Button
            className="mt-3 h-11 sm:h-9"
            icon={<ArrowsClockwiseIcon size={16} />}
            loading={scanning}
            onClick={() => void scan()}
          >
            {t('scanAgain')}
          </Button>
        </div>
      </div>
    </Section>
  );
}
