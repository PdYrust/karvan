import { ListIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { Mark } from '../../components/brand/Mark';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { AccountMenu } from './AccountMenu';
import { shellMessages } from './messages';
import { SafeModePill } from './SafeModePill';

const iconButton =
  'grid size-9 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint pointer-coarse:size-11';

export function TopBar({
  onMenu,
  onSearch,
  onShortcuts,
}: {
  onMenu: () => void;
  onSearch: () => void;
  onShortcuts: () => void;
}) {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  return (
    <header className="sticky top-0 z-10 flex h-14 shrink-0 items-center gap-1 border-b border-kumo-hairline bg-kumo-canvas px-3 sm:px-5">
      <button
        type="button"
        aria-label={t('openMenu')}
        className={`${iconButton} lg:hidden`}
        onClick={onMenu}
      >
        <ListIcon size={20} aria-hidden="true" />
      </button>
      <span className="ms-1 lg:hidden">
        <Mark size={26} title={ts('brand')} />
      </span>

      <div className="ms-auto flex items-center gap-2">
        <button
          type="button"
          aria-label={t('search')}
          className={`${iconButton} lg:hidden`}
          onClick={onSearch}
        >
          <MagnifyingGlassIcon size={18} aria-hidden="true" />
        </button>
        <SafeModePill />
        <AccountMenu onShortcuts={onShortcuts} />
      </div>
    </header>
  );
}
