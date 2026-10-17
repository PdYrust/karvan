import { Tooltip } from '@cloudflare/kumo/components/tooltip';
import { cn } from '@cloudflare/kumo/utils';
import { MagnifyingGlassIcon, SidebarSimpleIcon } from '@phosphor-icons/react';
import { Mark } from '../../components/brand/Mark';
import { useLocale, useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';
import { PanelSwitcher } from './PanelSwitcher';
import { SearchButton } from './SearchButton';
import { SidebarNav } from './SidebarNav';

const iconButton =
  'grid size-10 place-items-center rounded-md text-kumo-subtle hover:bg-kumo-tint hover:text-kumo-default pointer-coarse:size-11';

/** The sidebar body: switcher row, search and navigation. Shared by the desktop column and the phone sheet. */
export function SidebarBody({
  onSearch,
  onNavigate,
}: {
  onSearch: () => void;
  onNavigate?: () => void;
}) {
  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex">
        <PanelSwitcher />
      </div>
      <SearchButton onClick={onSearch} />
      <SidebarNav onNavigate={onNavigate} />
    </div>
  );
}

/** The desktop sidebar, as in the dashboard: mark and switcher, search, navigation, collapse. */
export function Sidebar({
  collapsed,
  onToggle,
  onSearch,
}: {
  collapsed: boolean;
  onToggle: () => void;
  onSearch: () => void;
}) {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  const { dir } = useLocale();
  const side = dir === 'rtl' ? 'left' : 'right';
  const toggleLabel = collapsed ? t('expandSidebar') : t('collapseSidebar');
  return (
    <aside className="sticky top-0 hidden h-dvh flex-col border-e border-kumo-hairline bg-kumo-canvas lg:flex">
      <div
        className={cn(
          'flex h-14 shrink-0 items-center gap-2 border-b border-kumo-hairline',
          collapsed ? 'justify-center px-2' : 'px-3',
        )}
      >
        <Mark size={28} title={ts('brand')} />
        {collapsed ? null : <PanelSwitcher />}
      </div>
      <div
        className={cn(
          'flex flex-1 flex-col gap-3 overflow-y-auto p-3',
          collapsed && 'items-center px-2',
        )}
      >
        {collapsed ? (
          <>
            <PanelSwitcher collapsed />
            <Tooltip content={t('search')} side={side} asChild>
              <button
                type="button"
                aria-label={t('search')}
                className={iconButton}
                onClick={onSearch}
              >
                <MagnifyingGlassIcon size={18} aria-hidden="true" />
              </button>
            </Tooltip>
          </>
        ) : (
          <SearchButton onClick={onSearch} />
        )}
        <SidebarNav collapsed={collapsed} />
      </div>
      <div
        className={cn(
          'flex shrink-0 border-t border-kumo-hairline p-2',
          collapsed ? 'justify-center' : 'justify-start',
        )}
      >
        <Tooltip content={toggleLabel} side={side} asChild>
          <button
            type="button"
            aria-label={toggleLabel}
            aria-pressed={collapsed}
            className={iconButton}
            onClick={onToggle}
          >
            <SidebarSimpleIcon size={18} aria-hidden="true" className="rtl:-scale-x-100" />
          </button>
        </Tooltip>
      </div>
    </aside>
  );
}
