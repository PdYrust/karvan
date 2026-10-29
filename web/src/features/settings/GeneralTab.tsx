import { Select } from '@cloudflare/kumo/components/select';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { SettingRow, SettingsCard } from '../../components/layout/SettingRow';
import { locales, type Locale } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames } from '../../i18n/shared';
import { getInstall } from '../../lib/api/install';
import type { CalendarPref, ThemePref } from '../../lib/preferences/preferences';
import { useFormat, useRegion } from '../../lib/preferences/region';
import { useTheme } from '../../lib/preferences/theme';
import { useNow } from '../overview/use-now';
import { settingsMessages } from './messages';
import { zoneItems } from './zones';

const control = 'w-full sm:w-64 pointer-coarse:h-9';

export function GeneralTab() {
  const t = useT(settingsMessages);
  const { locale, setLocale } = useLocale();
  const { pref, setPref } = useTheme();
  const { timeZone, calendarPref, setTimeZone, setCalendarPref } = useRegion();
  const f = useFormat();
  const now = useNow();
  const install = useQuery({
    queryKey: ['settings', 'install'],
    queryFn: getInstall,
    staleTime: Infinity,
  });

  const languages = Object.fromEntries(locales.map((l) => [l, languageNames[l]]));
  const themes: Record<ThemePref, string> = {
    light: t('themeLight'),
    dark: t('themeDark'),
    system: t('themeSystem'),
  };
  const calendars: Record<CalendarPref, string> = {
    auto: t('calendarAuto'),
    persian: t('calendarPersian'),
    gregory: t('calendarGregory'),
  };
  const zones = useMemo(
    () => Object.fromEntries(zoneItems(timeZone, locale).map((z) => [z.id, z.label])),
    [timeZone, locale],
  );

  return (
    <>
      <SettingsCard title={t('lookTitle')}>
        <SettingRow
          title={t('language')}
          labelId="set-language"
          description={t('languageHint')}
          control={
            <Select
              aria-labelledby="set-language"
              className={control}
              value={locale}
              onValueChange={(v) => v && setLocale(v as Locale)}
              items={languages}
            />
          }
        />
        <SettingRow
          title={t('appearance')}
          labelId="set-appearance"
          description={t('appearanceHint')}
          control={
            <Select
              aria-labelledby="set-appearance"
              className={control}
              value={pref}
              onValueChange={(v) => v && setPref(v as ThemePref)}
              items={themes}
            />
          }
        />
      </SettingsCard>
      <SettingsCard title={t('regionTitle')}>
        <SettingRow
          title={t('timeZone')}
          labelId="set-zone"
          description={t('timeZoneHint', { time: f.time(now), zone: f.zone })}
          control={
            <Select
              aria-labelledby="set-zone"
              className={control}
              value={timeZone}
              onValueChange={(v) => v && setTimeZone(v as string)}
              items={zones}
            />
          }
        />
        <SettingRow
          title={t('calendar')}
          labelId="set-calendar"
          description={t('calendarHint')}
          control={
            <Select
              aria-labelledby="set-calendar"
              className={control}
              value={calendarPref}
              onValueChange={(v) => v && setCalendarPref(v as CalendarPref)}
              items={calendars}
            />
          }
        />
      </SettingsCard>
      {install.data ? (
        <SettingsCard title={t('serverTitle')}>
          <SettingRow
            title={t('runsAs')}
            description={install.data.runsAsRoot ? t('runsAsRoot') : t('runsAsOther')}
            control={
              install.data.runsAsRoot ? (
                <bdi dir="ltr" className="font-mono text-sm text-kumo-strong">
                  root
                </bdi>
              ) : (
                <span className="text-sm text-kumo-strong">{t('otherUser')}</span>
              )
            }
          />
        </SettingsCard>
      ) : null}
    </>
  );
}
