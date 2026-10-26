import { cn } from '@cloudflare/kumo/utils';
import { MoonIcon, SunIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { useTheme } from '../../lib/preferences/theme';

/** Flips between light and dark; the native title is its tooltip. "Match system" is in the palette. */
export function ThemeToggle({ className }: { className?: string }) {
  const { mode, setPref } = useTheme();
  const t = useT(sharedMessages);
  const label = mode === 'dark' ? t('switchToLight') : t('switchToDark');
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={() => setPref(mode === 'dark' ? 'light' : 'dark')}
      className={cn(
        'grid size-8 place-items-center rounded-md transition-colors pointer-coarse:size-11',
        className ?? 'text-kumo-default hover:bg-kumo-tint',
      )}
    >
      {mode === 'dark' ? <SunIcon size={18} /> : <MoonIcon size={18} />}
    </button>
  );
}
