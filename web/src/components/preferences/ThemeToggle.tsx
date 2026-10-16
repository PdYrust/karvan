import { Button } from '@cloudflare/kumo/components/button';
import { MoonIcon, SunIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { useTheme } from '../../lib/preferences/theme';

/** Flips between light and dark; the native title is its tooltip. "Match system" is in the palette. */
export function ThemeToggle() {
  const { mode, setPref } = useTheme();
  const t = useT(sharedMessages);
  const label = mode === 'dark' ? t('switchToLight') : t('switchToDark');
  return (
    <Button
      variant="ghost"
      shape="square"
      aria-label={label}
      title={label}
      className="pointer-coarse:size-11"
      icon={mode === 'dark' ? <SunIcon size={20} /> : <MoonIcon size={20} />}
      onClick={() => setPref(mode === 'dark' ? 'light' : 'dark')}
    />
  );
}
