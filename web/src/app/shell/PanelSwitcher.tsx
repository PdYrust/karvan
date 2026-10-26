import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { cn } from '@cloudflare/kumo/utils';
import {
  CaretUpDownIcon,
  HardDrivesIcon,
  StackIcon,
  WarningCircleIcon,
} from '@phosphor-icons/react';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useCurrentPanel } from '../../features/panels/current-panel';
import { usePanels } from '../../features/panels/use-panels';
import { useT } from '../../i18n/locale';
import { panelKind } from '../../lib/api/panels';
import { shellMessages } from './messages';

/**
 * Chooses the panel the pages work on, or all of them, in the place of the dashboard's
 * account switcher: the wizard's default destination and a filter on lists.
 */
export function PanelSwitcher({ collapsed = false }: { collapsed?: boolean }) {
  const t = useT(shellMessages);
  const panels = usePanels();
  const { panel, select } = useCurrentPanel();
  const list = (panels.data ?? []).filter((p) => p.id > 0);
  const kind = panel ? panelKind(panel) : undefined;
  const label = panel ? panel.name : list.length > 0 ? t('allPanels') : t('noPanel');
  const loading = panels.isPending;

  const icon = loading ? (
    <span aria-hidden="true" className="size-[18px] animate-pulse rounded bg-kumo-recessed" />
  ) : kind ? (
    <PanelMark kind={kind} size={18} />
  ) : list.length > 0 ? (
    <StackIcon size={18} aria-hidden="true" className="text-kumo-subtle" />
  ) : (
    <HardDrivesIcon size={18} aria-hidden="true" className="text-kumo-subtle" />
  );

  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <button
            type="button"
            aria-label={`${t('panelSwitcherLabel')}: ${label}`}
            className={cn(
              'flex min-h-10 items-center gap-2 rounded-md text-start hover:bg-kumo-tint pointer-coarse:min-h-11',
              collapsed ? 'size-10 justify-center' : 'min-w-0 flex-1 px-2',
            )}
          >
            {icon}
            {collapsed ? null : (
              <>
                {loading ? (
                  <span
                    aria-hidden="true"
                    className="h-3.5 flex-1 animate-pulse rounded bg-kumo-recessed"
                  />
                ) : (
                  <span className="min-w-0 flex-1 truncate text-base font-medium text-kumo-default">
                    {label}
                  </span>
                )}
                <CaretUpDownIcon size={14} aria-hidden="true" className="text-kumo-subtle" />
              </>
            )}
          </button>
        }
      />
      <DropdownMenu.Content className="min-w-64">
        <DropdownMenu.Group>
          <DropdownMenu.Label>{t('panelsOnServer')}</DropdownMenu.Label>
          <DropdownMenu.RadioGroup
            value={panel ? String(panel.id) : 'all'}
            onValueChange={(v: string) => select(v === 'all' ? null : Number(v))}
          >
            {list.length > 1 || panel ? (
              <DropdownMenu.RadioItem value="all" closeOnClick>
                <span className="flex w-full items-center gap-2">
                  <StackIcon size={16} aria-hidden="true" className="text-kumo-subtle" />
                  {t('allPanels')}
                  <span className="ms-auto">
                    <DropdownMenu.RadioItemIndicator />
                  </span>
                </span>
              </DropdownMenu.RadioItem>
            ) : null}
            {list.map((p) => {
              const k = panelKind(p);
              return (
                <DropdownMenu.RadioItem key={p.id} value={String(p.id)} closeOnClick>
                  <span className="flex w-full items-center gap-2">
                    {k ? <PanelMark kind={k} size={16} /> : null}
                    <span className="truncate">{p.name}</span>
                    {p.ready ? null : (
                      <WarningCircleIcon
                        size={14}
                        className="shrink-0 text-kumo-warning"
                        aria-label={t('panelNeedsAttention')}
                      />
                    )}
                    <span className="ms-auto">
                      <DropdownMenu.RadioItemIndicator />
                    </span>
                  </span>
                </DropdownMenu.RadioItem>
              );
            })}
          </DropdownMenu.RadioGroup>
          {list.length === 0 ? (
            <DropdownMenu.Item disabled>{t('noPanel')}</DropdownMenu.Item>
          ) : null}
        </DropdownMenu.Group>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
