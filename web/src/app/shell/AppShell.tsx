import { TooltipProvider } from '@cloudflare/kumo/components/tooltip';
import { cn } from '@cloudflare/kumo/utils';
import { Suspense, useCallback, useMemo, useRef, useState } from 'react';
import { Outlet, useLocation } from 'react-router';
import { PageSkeleton } from '../../components/feedback/PageSkeleton';
import { CurrentPanelProvider } from '../../features/panels/current-panel';
import { useSession } from '../../features/session/use-session';
import { useT } from '../../i18n/locale';
import { readSidebarCollapsed, saveSidebarCollapsed } from '../../lib/preferences/preferences';
import { CommandMenu } from './CommandMenu';
import { Footer } from './Footer';
import { shellMessages } from './messages';
import { MobileNav } from './MobileNav';
import type { ShellOutlet } from './outlet';
import { PageBoundary } from './PageBoundary';
import { ShortcutSheet } from './ShortcutSheet';
import { Sidebar, SidebarBody } from './Sidebar';
import { TopBar } from './TopBar';
import { useActivity } from './use-activity';
import { useModHotkey } from './use-hotkey';
import { useRecordRecents } from './use-record-recents';
import { useShortcuts } from './use-shortcuts';
import { useWide } from './use-wide';

/** Dashboard layout: sidebar (a sheet on small screens), top bar, the page and the footer. */
export function AppShell() {
  const t = useT(shellMessages);
  // Loads the CSRF token every state-changing request needs.
  useSession();
  useActivity();
  useRecordRecents();
  const { pathname } = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  // The palette builds every command and the sheet its rows on each render, so
  // neither mounts before it first opens; after that each stays for its close.
  const [searchUsed, setSearchUsed] = useState(false);
  const [helpUsed, setHelpUsed] = useState(false);
  const [collapsed, setCollapsed] = useState(readSidebarCollapsed);
  // The column is hidden under lg anyway; not building it keeps phones quick.
  const wide = useWide();
  // What had focus when the palette opened: a search field, a button or the page itself.
  const opener = useRef<HTMLElement | null>(null);
  const openSearch = useCallback(() => {
    opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setSearchUsed(true);
    // One frame later, so a palette mounting now still opens with its animation.
    requestAnimationFrame(() => setSearchOpen(true));
  }, []);
  const openHelp = useCallback(() => {
    setHelpUsed(true);
    requestAnimationFrame(() => setHelpOpen(true));
  }, []);
  const outlet = useMemo<ShellOutlet>(() => ({ openSearch }), [openSearch]);
  useModHotkey('KeyK', openSearch);
  useShortcuts(openHelp);

  function toggleSidebar() {
    saveSidebarCollapsed(!collapsed);
    setCollapsed(!collapsed);
  }

  return (
    <CurrentPanelProvider>
      <TooltipProvider>
        <div
          className={cn(
            'min-h-dvh lg:grid',
            collapsed ? 'lg:grid-cols-[4rem_minmax(0,1fr)]' : 'lg:grid-cols-[14rem_minmax(0,1fr)]',
          )}
        >
          <a
            href="#content"
            className="sr-only z-50 rounded-lg bg-kumo-base px-3 py-2 focus:not-sr-only focus:fixed focus:start-3 focus:top-3"
          >
            {t('skipToContent')}
          </a>

          {wide ? (
            <Sidebar collapsed={collapsed} onToggle={toggleSidebar} onSearch={openSearch} />
          ) : null}

          <MobileNav open={menuOpen} onClose={() => setMenuOpen(false)}>
            <SidebarBody
              onSearch={() => {
                setMenuOpen(false);
                openSearch();
              }}
              onNavigate={() => setMenuOpen(false)}
            />
          </MobileNav>

          <div className="flex min-w-0 flex-col">
            <TopBar onMenu={() => setMenuOpen(true)} onShortcuts={openHelp} />
            <main id="content" tabIndex={-1} className="flex-1 outline-none">
              <PageBoundary page={pathname}>
                <Suspense fallback={<PageSkeleton label={t('loadingPage')} />}>
                  <Outlet context={outlet} />
                </Suspense>
              </PageBoundary>
            </main>
            <Footer />
          </div>

          {searchUsed ? (
            <CommandMenu
              open={searchOpen}
              onOpenChange={setSearchOpen}
              onShortcuts={openHelp}
              opener={opener}
            />
          ) : null}
          {helpUsed ? <ShortcutSheet open={helpOpen} onOpenChange={setHelpOpen} /> : null}
        </div>
      </TooltipProvider>
    </CurrentPanelProvider>
  );
}
