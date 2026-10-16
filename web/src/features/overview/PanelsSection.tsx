import { Button } from '@cloudflare/kumo/components/button';
import { ArrowsClockwiseIcon, HardDrivesIcon } from '@phosphor-icons/react';
import { Section } from '../../components/layout/Section';
import { PanelLogo } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { useScanPanels } from '../panels/use-scan-panels';
import { overviewMessages } from './messages';

/** Panels found on this server. Discovery lands in phase 3; until then this is the empty state. */
export function PanelsSection() {
  const t = useT(overviewMessages);
  const { scan, scanning } = useScanPanels();

  return (
    <Section title={t('panelsTitle')}>
      <div className="flex flex-col items-center gap-3 px-6 pt-8 pb-10 text-center">
        <HardDrivesIcon size={44} weight="light" aria-hidden="true" className="text-kumo-default" />
        <h3 className="mt-1 text-[20px] leading-tight font-semibold text-kumo-strong">
          {t('panelsEmptyTitle')}
        </h3>
        <div
          className="flex items-center gap-4 text-sm text-kumo-subtle"
          aria-label={t('panelsSupported')}
        >
          <span className="flex items-center gap-2 text-kumo-default">
            <PanelLogo kind="marzban" size={18} />
            Marzban
          </span>
          <span className="flex items-center gap-2 text-kumo-default">
            <span className="rounded-sm bg-on-brand">
              <PanelLogo kind="pasarguard" size={18} />
            </span>
            PasarGuard
          </span>
        </div>
        <Button
          className="mt-2 h-9 pointer-coarse:h-11"
          icon={<ArrowsClockwiseIcon size={16} />}
          loading={scanning}
          onClick={() => void scan()}
        >
          {t('scanAgain')}
        </Button>
      </div>
    </Section>
  );
}
