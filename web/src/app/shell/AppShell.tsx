import { useCallback, useState } from 'react';
import { Outlet } from 'react-router';
import { Wordmark } from '../../components/brand/Wordmark';
import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { useT } from '../../i18n/locale';
import { CommandMenu } from './CommandMenu';
import { shellMessages } from './messages';
import { MobileNav } from './MobileNav';
import { PanelSwitcher } from './PanelSwitcher';
import { SearchButton } from './SearchButton';
import { SidebarNav } from './SidebarNav';
import { TopBar } from './TopBar';
import { useModHotkey } from './use-hotkey';
import { useSession } from '../../features/session/use-session';

function SidebarBody({ onSearch, onNavigate }: { onSearch: () => void; onNavigate?: () => void }) {
  return (
    <div className="flex flex-col gap-3 p-3">
      <PanelSwitcher />
      <SearchButton onClick={onSearch} />
      <SidebarNav onNavigate={onNavigate} />
    </div>
  );
}

/** Dashboard layout: sidebar (a sheet on small screens), top bar and the page. */
export function AppShell() {
  const t = useT(shellMessages);
  // Loads the CSRF token every state-changing request needs.
  useSession();
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const openSearch = useCallback(() => setSearchOpen(true), []);
  useModHotkey('KeyK', openSearch);

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[16rem_1fr]">
      <a
        href="#content"
        className="sr-only z-50 rounded-lg bg-kumo-base px-3 py-2 focus:not-sr-only focus:fixed focus:start-3 focus:top-3"
      >
        {t('skipToContent')}
      </a>

      <aside className="sticky top-0 hidden h-dvh flex-col overflow-y-auto border-e border-kumo-hairline bg-kumo-canvas lg:flex">
        <div className="flex h-14 shrink-0 items-center border-b border-kumo-hairline px-5">
          <Wordmark />
        </div>
        <SidebarBody onSearch={openSearch} />
      </aside>

      <MobileNav open={menuOpen} onClose={() => setMenuOpen(false)}>
        <SidebarBody
          onSearch={() => {
            setMenuOpen(false);
            openSearch();
          }}
          onNavigate={() => setMenuOpen(false)}
        />
        <div className="mt-auto border-t border-kumo-hairline p-3 sm:hidden">
          <LanguageSwitch />
        </div>
      </MobileNav>

      <div className="flex min-w-0 flex-col">
        <TopBar onMenu={() => setMenuOpen(true)} onSearch={openSearch} />
        <main id="content" tabIndex={-1} className="flex-1 outline-none">
          <Outlet />
        </main>
      </div>

      <CommandMenu open={searchOpen} onOpenChange={setSearchOpen} />
    </div>
  );
}
