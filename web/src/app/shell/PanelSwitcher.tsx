import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { ArrowsClockwiseIcon, CaretUpDownIcon, HardDrivesIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { useScanPanels } from '../../features/panels/use-scan-panels';
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
            className="flex min-h-12 w-full items-center gap-3 rounded-lg bg-kumo-base px-3 text-start ring ring-kumo-line transition-colors hover:bg-kumo-tint"
          >
            <span className="grid size-8 shrink-0 place-items-center rounded-md bg-kumo-recessed text-kumo-subtle">
              <HardDrivesIcon size={18} aria-hidden="true" />
            </span>
            <span className="flex min-w-0 flex-1 flex-col leading-tight">
              <span className="text-tiny text-kumo-subtle">{t('panelSwitcherLabel')}</span>
              <span className="truncate text-sm font-medium text-kumo-default">{t('noPanel')}</span>
            </span>
            <CaretUpDownIcon size={16} aria-hidden="true" className="text-kumo-subtle" />
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
