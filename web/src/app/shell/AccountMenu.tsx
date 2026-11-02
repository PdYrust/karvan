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
import { useOverview } from '../../features/overview/use-overview';
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
import { repoUrl } from './repo';

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
  const overview = useOverview().data;
  const version = overview?.version;
  const latest = overview?.latestVersion;
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
            className="relative pointer-coarse:size-9"
          >
            <UserCircleIcon size={22} aria-hidden="true" />
            {latest ? (
              <span
                aria-hidden="true"
                className="absolute top-1 end-1 size-2 rounded-full bg-kumo-success ring-2 ring-kumo-canvas"
              />
            ) : null}
          </Button>
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
        {version ? (
          latest ? (
            <a
              href={`${repoUrl}/releases/tag/v${latest.replace(/^v/, '')}`}
              target="_blank"
              rel="noopener noreferrer"
              className="mx-1 mb-1 flex flex-col gap-0.5 rounded-md bg-kumo-tint px-2.5 py-1.5 ring ring-kumo-hairline hover:bg-kumo-fill"
            >
              <span className="flex items-center justify-between gap-2 font-mono text-xs text-kumo-default tabular">
                <bdi dir="ltr">{version}</bdi>
                <span
                  aria-hidden="true"
                  className="size-1.5 shrink-0 rounded-full bg-kumo-success"
                />
              </span>
              <span className="whitespace-nowrap text-tiny font-normal text-kumo-link">
                <bdi dir="ltr" className="font-mono tabular">
                  {latest}
                </bdi>{' '}
                {t('newRelease')}
              </span>
            </a>
          ) : (
            <div className="mx-1 mb-1 flex flex-col gap-0.5 rounded-md bg-kumo-tint px-2.5 py-1.5 ring ring-kumo-hairline">
              <span className="font-mono text-xs text-kumo-default tabular">
                <bdi dir="ltr">{version}</bdi>
              </span>
              <span className="whitespace-nowrap text-tiny font-normal text-kumo-subtle">
                {t('upToDate')}
              </span>
            </div>
          )
        ) : null}
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
