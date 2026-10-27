import { cn } from '@cloudflare/kumo/utils';
import { useEffect, useRef } from 'react';
import { useT } from '../../i18n/locale';
import { AccountMenu } from './AccountMenu';
import { GitHubLink } from './GitHubLink';
import { MenuIcon } from './MenuIcon';
import { shellMessages } from './messages';
import { SafeModePill } from './SafeModePill';
import { useScrollHide } from './scroll-hide';
import { useWide } from './use-wide';
/** Phones get only the menu button, GitHub and the account; the mark and search live in the sheet. */
export function TopBar({
  onMenu,
  onShortcuts,
  hold,
}: {
  onMenu: () => void;
  onShortcuts: () => void;
  /** The sheet or the palette is open: the bar stays. */
  hold: boolean;
}) {
  const t = useT(shellMessages);
  const wide = useWide();
  // Under lg the bar slides away while the page scrolls down (56 px, its height).
  const { hidden, show } = useScrollHide({ enabled: !wide, hold, after: 56 });
  const bar = useRef<HTMLElement>(null);
  useEffect(() => {
    // Focus moving into the bar (Tab from the page's top) brings it back.
    const el = bar.current;
    el?.addEventListener('focusin', show);
    return () => el?.removeEventListener('focusin', show);
  }, [show]);
  useEffect(() => {
    // Sticky rows under the bar (the analytics toolbar) move up with it.
    document.documentElement.toggleAttribute('data-bar-hidden', hidden);
  }, [hidden]);
  return (
    <header
      ref={bar}
      className={cn(
        'sticky top-0 z-10 flex h-14 shrink-0 items-center gap-1 border-b border-kumo-hairline bg-kumo-canvas px-3 transition-transform duration-200 ease-out sm:px-5',
        hidden && '-translate-y-full',
      )}
    >
      <button
        type="button"
        aria-label={t('openMenu')}
        className="grid size-8 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint pointer-coarse:size-9 lg:hidden"
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
