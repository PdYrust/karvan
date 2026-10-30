import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import {
  DesktopIcon,
  GearSixIcon,
  GlobeHemisphereEastIcon,
  KeyboardIcon,
  MoonIcon,
  SignOutIcon,
  SunIcon,
  TranslateIcon,
  UserCircleIcon,
} from '@phosphor-icons/react';
import { useMemo, type ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { useSubToggle } from '../../components/menus/use-sub-toggle';
import { useSession, useSignOut } from '../../features/session/use-session';
import { zoneItems } from '../../features/settings/zones';
import { locales, type Locale } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames } from '../../i18n/shared';
import type { ThemePref } from '../../lib/preferences/preferences';
import { useRegion } from '../../lib/preferences/region';
import { useTheme } from '../../lib/preferences/theme';
import { shellMessages } from './messages';
import { settingsPath } from './nav';

// Kumo's item icon slot and submenu caret use physical margins, so icons go
// in the children and the caret flips in RTL.
const subTrigger = 'gap-2 rtl:[&>svg:last-child]:-scale-x-100';

function Choice({ value, children }: { value: string; children: ReactNode }) {
  return (
    <DropdownMenu.RadioItem value={value} closeOnClick>
      <span className="flex w-full items-center gap-2">
        {children}
        <span className="ms-auto">
          <DropdownMenu.RadioItemIndicator />
        </span>
      </span>
    </DropdownMenu.RadioItem>
  );
}

/** A submenu whose trigger opens it and, pressed again, closes it. */
function Sub({ label, children }: { label: ReactNode; children: ReactNode }) {
  const { root, trigger } = useSubToggle();
  return (
    <DropdownMenu.Sub {...root}>
      <DropdownMenu.SubTrigger className={subTrigger} {...trigger}>
        {label}
      </DropdownMenu.SubTrigger>
      <DropdownMenu.SubContent>{children}</DropdownMenu.SubContent>
    </DropdownMenu.Sub>
  );
}

/** Everything personal, as in the dashboard: who, settings, appearance, language, time zone, shortcuts, sign out. */
export function AccountMenu({ onShortcuts }: { onShortcuts: () => void }) {
  const t = useT(shellMessages);
  const navigate = useNavigate();
  const { locale, setLocale } = useLocale();
  const { pref, setPref } = useTheme();
  const { timeZone, setTimeZone } = useRegion();
  const zones = useMemo(() => zoneItems(timeZone, locale), [timeZone, locale]);
  const signOut = useSignOut();
  const username = useSession().data?.operator?.username ?? '';
  const themes: { value: ThemePref; label: string; icon: ReactNode }[] = [
    { value: 'light', label: t('themeLight'), icon: <SunIcon size={16} aria-hidden="true" /> },
    { value: 'dark', label: t('themeDark'), icon: <MoonIcon size={16} aria-hidden="true" /> },
    {
      value: 'system',
      label: t('themeSystem'),
      icon: <DesktopIcon size={16} aria-hidden="true" />,
    },
  ];
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={t('account')}
            title={t('account')}
            className="pointer-coarse:size-9"
            icon={<UserCircleIcon size={22} />}
          />
        }
      />
      <DropdownMenu.Content className="w-48">
        <DropdownMenu.Group>
          <DropdownMenu.Label>
            <span
              title={username}
              className="block truncate font-mono text-sm font-normal text-kumo-default"
            >
              <bdi dir="ltr">{username}</bdi>
            </span>
          </DropdownMenu.Label>
        </DropdownMenu.Group>
        <DropdownMenu.Separator />
        <DropdownMenu.Item className="gap-2" onClick={() => void navigate(settingsPath)}>
          <GearSixIcon size={16} aria-hidden="true" />
          {t('settings')}
        </DropdownMenu.Item>
        <Sub
          label={
            <>
              {themes.find((x) => x.value === pref)?.icon}
              {t('appearance')}
            </>
          }
        >
          <DropdownMenu.RadioGroup value={pref} onValueChange={(v: ThemePref) => setPref(v)}>
            {themes.map((x) => (
              <Choice key={x.value} value={x.value}>
                {x.icon}
                {x.label}
              </Choice>
            ))}
          </DropdownMenu.RadioGroup>
        </Sub>
        <Sub
          label={
            <>
              <TranslateIcon size={16} aria-hidden="true" />
              {t('language')}
            </>
          }
        >
          <DropdownMenu.RadioGroup value={locale} onValueChange={(v: Locale) => setLocale(v)}>
            {locales.map((l) => (
              <Choice key={l} value={l}>
                <span lang={l}>{languageNames[l]}</span>
              </Choice>
            ))}
          </DropdownMenu.RadioGroup>
        </Sub>
        <Sub
          label={
            <>
              <GlobeHemisphereEastIcon size={16} aria-hidden="true" />
              {t('timeZone')}
            </>
          }
        >
          <DropdownMenu.RadioGroup value={timeZone} onValueChange={(v: string) => setTimeZone(v)}>
            {zones.map((z) => (
              <Choice key={z.id} value={z.id}>
                {z.label}
              </Choice>
            ))}
          </DropdownMenu.RadioGroup>
        </Sub>
        <DropdownMenu.Item className="gap-2" onClick={onShortcuts}>
          <KeyboardIcon size={16} aria-hidden="true" />
          {t('shortcuts')}
          <kbd className="ms-auto font-mono text-tiny text-kumo-subtle">?</kbd>
        </DropdownMenu.Item>
        <DropdownMenu.Separator />
        <DropdownMenu.Item className="gap-2" onClick={signOut}>
          <SignOutIcon size={16} aria-hidden="true" className="rtl:-scale-x-100" />
          {t('signOut')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
