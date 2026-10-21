import { MagnifyingGlassIcon } from '@phosphor-icons/react';
import { shellMessages } from '../../app/shell/messages';
import { useShell } from '../../app/shell/outlet';
import { paletteShortcut } from '../../app/shell/SearchButton';
import { useT } from '../../i18n/locale';

/** The wide search across the overview, like the dashboard home's: it opens the palette. */
export function HomeSearch() {
  const ts = useT(shellMessages);
  const { openSearch } = useShell();
  return (
    <button
      type="button"
      aria-haspopup="dialog"
      onClick={openSearch}
      className="flex h-11 w-full items-center gap-3 rounded-lg bg-kumo-base px-4 text-start text-kumo-subtle ring ring-kumo-line hover:ring-kumo-focus/40"
    >
      <MagnifyingGlassIcon size={18} aria-hidden="true" className="shrink-0" />
      <span className="min-w-0 flex-1 truncate">{ts('searchPlaceholder')}</span>
      <kbd
        dir="ltr"
        className="hidden shrink-0 rounded border border-kumo-hairline px-1.5 py-0.5 font-mono text-tiny sm:inline"
      >
        {paletteShortcut}
      </kbd>
    </button>
  );
}
