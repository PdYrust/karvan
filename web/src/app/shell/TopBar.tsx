import { useT } from '../../i18n/locale';
import { AccountMenu } from './AccountMenu';
import { GitHubLink } from './GitHubLink';
import { MenuIcon } from './MenuIcon';
import { shellMessages } from './messages';
import { SafeModePill } from './SafeModePill';
import { useWide } from './use-wide';
/** Phones get only the menu button, GitHub and the account; the mark and search live in the sheet. */
export function TopBar({ onMenu, onShortcuts }: { onMenu: () => void; onShortcuts: () => void }) {
  const t = useT(shellMessages);
  const wide = useWide();
  return (
    <header className="sticky top-0 z-10 flex h-14 shrink-0 items-center gap-1 border-b border-kumo-hairline bg-kumo-canvas px-3 sm:px-5">
      <button
        type="button"
        aria-label={t('openMenu')}
        className="grid size-9 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint pointer-coarse:size-11 lg:hidden"
        onClick={onMenu}
      >
        <MenuIcon />
      </button>
      <div className="ms-auto flex items-center gap-2">
        {wide ? <SafeModePill /> : null}
        <GitHubLink />
        <AccountMenu onShortcuts={onShortcuts} />
      </div>
    </header>
  );
}
