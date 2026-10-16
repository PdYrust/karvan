import { Button } from '@cloudflare/kumo/components/button';
import { ListIcon, MagnifyingGlassIcon } from '@phosphor-icons/react';
import { Wordmark } from '../../components/brand/Wordmark';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { useT } from '../../i18n/locale';
import { AccountMenu } from './AccountMenu';
import { shellMessages } from './messages';

const isMac = /Mac|iPhone|iPad/.test(navigator.platform);

export function TopBar({ onMenu, onSearch }: { onMenu: () => void; onSearch: () => void }) {
  const t = useT(shellMessages);
  return (
    <header className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b border-kumo-hairline bg-kumo-base/90 px-3 backdrop-blur sm:px-5">
      <Button
        variant="ghost"
        shape="square"
        aria-label={t('openMenu')}
        className="size-11 lg:hidden"
        icon={<ListIcon size={22} />}
        onClick={onMenu}
      />
      <span className="lg:hidden">
        <Wordmark size={26} />
      </span>

      <button
        type="button"
        onClick={onSearch}
        className="ms-auto flex h-11 items-center gap-2 rounded-lg px-3 text-sm text-kumo-subtle ring ring-kumo-line transition-colors hover:bg-kumo-tint hover:text-kumo-default sm:h-9 sm:w-72 lg:ms-0"
      >
        <MagnifyingGlassIcon size={18} aria-hidden="true" />
        <span className="hidden sm:inline">{t('search')}</span>
        <kbd
          dir="ltr"
          className="ms-auto hidden rounded border border-kumo-hairline px-1.5 font-mono text-tiny sm:inline"
        >
          {isMac ? '⌘K' : 'Ctrl K'}
        </kbd>
        <span className="sr-only sm:hidden">{t('search')}</span>
      </button>

      <div className="flex items-center gap-1 lg:ms-auto">
        <ThemeToggle />
        <AccountMenu />
      </div>
    </header>
  );
}
