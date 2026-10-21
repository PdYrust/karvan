import { MagnifyingGlassIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

const isMac = /Mac|iPhone|iPad/.test(navigator.platform);

/** The palette's shortcut as this keyboard writes it. */
export const paletteShortcut = isMac ? '⌘K' : 'Ctrl K';

/** Looks like a search field, opens the Ctrl+K palette. */
export function SearchButton({ onClick }: { onClick: () => void }) {
  const t = useT(shellMessages);
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex h-9 w-full items-center gap-2 rounded-lg bg-kumo-base px-3 text-sm text-kumo-placeholder ring ring-kumo-line hover:text-kumo-subtle pointer-coarse:h-11"
    >
      <MagnifyingGlassIcon size={16} aria-hidden="true" className="text-kumo-subtle" />
      <span className="flex-1 text-start">{t('quickSearch')}</span>
      <kbd dir="ltr" className="font-mono text-tiny text-kumo-subtle">
        {paletteShortcut}
      </kbd>
    </button>
  );
}
