import {
  ArrowsClockwiseIcon,
  DesktopIcon,
  MoonIcon,
  SignOutIcon,
  SunIcon,
  TranslateIcon,
} from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { locales } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames } from '../../i18n/shared';
import { useScanPanels } from '../../features/panels/use-scan-panels';
import { useTheme } from '../../lib/preferences/theme';
import { shellMessages } from './messages';
import { nav } from './nav';
import { useSignOut } from '../../features/session/use-session';

export interface Command {
  id: string;
  title: string;
  icon: ReactNode;
  /** Extra words that should also find this command, such as the other language's name. */
  keywords?: string;
  run: () => void;
}

export interface CommandGroup {
  id: string;
  label: string;
  items: Command[];
}

/** Everything the Ctrl+K palette can do. Only built pages are listed. */
export function useCommands(): CommandGroup[] {
  const t = useT(shellMessages);
  const navigate = useNavigate();
  const { scan } = useScanPanels();
  const { setPref } = useTheme();
  const { locale, setLocale } = useLocale();
  const signOut = useSignOut();

  const pages: Command[] = nav.flatMap((g) =>
    g.items.flatMap((item) => {
      const to = item.to;
      if (!to) return [];
      const Icon = item.icon;
      return [
        {
          id: `page:${to}`,
          title: t(item.label),
          icon: <Icon size={18} />,
          run: () => void navigate(to),
        },
      ];
    }),
  );

  const actions: Command[] = [
    {
      id: 'scan',
      title: t('actionScan'),
      icon: <ArrowsClockwiseIcon size={18} />,
      run: () => void scan(),
    },
    { id: 'signout', title: t('actionSignOut'), icon: <SignOutIcon size={18} />, run: signOut },
  ];

  const preferences: Command[] = [
    {
      id: 'theme:light',
      title: t('actionThemeLight'),
      icon: <SunIcon size={18} />,
      run: () => setPref('light'),
    },
    {
      id: 'theme:dark',
      title: t('actionThemeDark'),
      icon: <MoonIcon size={18} />,
      run: () => setPref('dark'),
    },
    {
      id: 'theme:system',
      title: t('actionThemeSystem'),
      icon: <DesktopIcon size={18} />,
      run: () => setPref('system'),
    },
    ...locales
      .filter((l) => l !== locale)
      .map((l) => ({
        id: `lang:${l}`,
        title: t('actionLanguage', { language: languageNames[l] }),
        keywords: 'language زبان farsi persian english',
        icon: <TranslateIcon size={18} />,
        run: () => setLocale(l),
      })),
  ];

  return [
    { id: 'pages', label: t('groupPages'), items: pages },
    { id: 'actions', label: t('groupActions'), items: actions },
    { id: 'preferences', label: t('groupPreferences'), items: preferences },
  ];
}

/** Case-insensitive match on title and keywords; empty groups are dropped. */
export function filterCommands(groups: CommandGroup[], query: string): CommandGroup[] {
  const q = query.trim().toLocaleLowerCase();
  if (!q) return groups;
  return groups
    .map((g) => ({
      ...g,
      items: g.items.filter((c) =>
        `${c.title} ${c.keywords ?? ''}`.toLocaleLowerCase().includes(q),
      ),
    }))
    .filter((g) => g.items.length > 0);
}
