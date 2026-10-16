import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { ArrowsClockwiseIcon, CaretUpDownIcon, HardDrivesIcon } from '@phosphor-icons/react';
import { useScanPanels } from '../../features/panels/use-scan-panels';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

/**
 * Chooses which panel the pages work on. Discovery comes in phase 3, so the
 * list is empty for now.
 */
export function PanelSwitcher() {
  const t = useT(shellMessages);
  const { scan } = useScanPanels();
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={t('panelSwitcherLabel')}
            className="flex min-h-10 w-full items-center gap-3 rounded-md px-3 text-start transition-colors hover:bg-kumo-tint pointer-coarse:min-h-11"
          >
            <HardDrivesIcon size={18} aria-hidden="true" className="text-kumo-subtle" />
            <span className="min-w-0 flex-1 truncate text-base text-kumo-default">
              {t('noPanel')}
            </span>
            <CaretUpDownIcon size={14} aria-hidden="true" className="text-kumo-subtle" />
          </button>
        }
      />
      <DropdownMenu.Content className="min-w-60">
        <DropdownMenu.Group>
          <DropdownMenu.Label>{t('panelsOnServer')}</DropdownMenu.Label>
          <DropdownMenu.Item disabled>{t('noPanelsFound')}</DropdownMenu.Item>
        </DropdownMenu.Group>
        <DropdownMenu.Separator />
        <DropdownMenu.Item onClick={() => void scan()}>
          <span className="flex items-center gap-2">
            <ArrowsClockwiseIcon size={16} aria-hidden="true" />
            {t('scanAgain')}
          </span>
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
