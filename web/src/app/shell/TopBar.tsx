import { ListIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { Wordmark } from '../../components/brand/Wordmark';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { AccountMenu } from './AccountMenu';
import { shellMessages } from './messages';

const iconButton =
  'grid size-9 place-items-center rounded-md text-kumo-default transition-colors hover:bg-kumo-tint pointer-coarse:size-11';

export function TopBar({ onMenu, onSearch }: { onMenu: () => void; onSearch: () => void }) {
  const t = useT(shellMessages);
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
        <Wordmark size={24} />
      </span>

      <div className="ms-auto flex items-center gap-1">
        <button
          type="button"
          aria-label={t('search')}
          className={`${iconButton} lg:hidden`}
          onClick={onSearch}
        >
          <MagnifyingGlassIcon size={18} aria-hidden="true" />
        </button>
        <LanguageSwitch className="hidden text-kumo-default hover:bg-kumo-tint sm:inline-flex" />
        <ThemeToggle />
        <AccountMenu />
      </div>
    </header>
  );
}
