import { useCallback, useState } from 'react';
import { Outlet } from 'react-router';
import { Wordmark } from '../../components/brand/Wordmark';
import { useT } from '../../i18n/locale';
import { CommandMenu } from './CommandMenu';
import { shellMessages } from './messages';
import { MobileNav } from './MobileNav';
import { PanelSwitcher } from './PanelSwitcher';
import { SidebarNav } from './SidebarNav';
import { TopBar } from './TopBar';
import { useModHotkey } from './use-hotkey';

function SidebarBody({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <>
      <PanelSwitcher />
      <SidebarNav onNavigate={onNavigate} />
    </>
  );
}

/** Sidebar (a sheet on small screens), top bar and the page. */
export function AppShell() {
  const t = useT(shellMessages);
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

      <aside className="sticky top-0 hidden h-dvh flex-col gap-5 overflow-y-auto border-e border-kumo-hairline bg-kumo-base px-3 py-4 lg:flex">
        <div className="px-2 pt-1">
          <Wordmark size={28} />
        </div>
        <SidebarBody />
      </aside>

      <MobileNav open={menuOpen} onClose={() => setMenuOpen(false)}>
        <SidebarBody onNavigate={() => setMenuOpen(false)} />
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
