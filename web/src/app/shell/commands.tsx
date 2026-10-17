import {
  ArrowsClockwiseIcon,
  DesktopIcon,
  GearSixIcon,
  KeyboardIcon,
  MoonIcon,
  ShieldCheckIcon,
  SignOutIcon,
  SunIcon,
  TranslateIcon,
} from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { PanelMark } from '../../components/panels/PanelLabel';
import { useCurrentPanel } from '../../features/panels/current-panel';
import { usePanels, useScanPanels } from '../../features/panels/use-panels';
import { useSignOut } from '../../features/session/use-session';
import { locales } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames } from '../../i18n/shared';
import { panelKind } from '../../lib/api/panels';
import { useTheme } from '../../lib/preferences/theme';
import { shellMessages } from './messages';
import { settingsPath } from './nav';
import { builtPages } from './routes-built';

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

const icon = 'size-[18px]';

/** Everything the Ctrl+K palette can do. Only built pages are listed. */
export function useCommands(onShortcuts: () => void): CommandGroup[] {
  const t = useT(shellMessages);
  const navigate = useNavigate();
  const { scan } = useScanPanels();
  const panels = usePanels().data ?? [];
  const { select } = useCurrentPanel();
  const { setPref } = useTheme();
  const { locale, setLocale } = useLocale();
  const signOut = useSignOut();

  const pages: Command[] = builtPages().map((item) => {
    const Icon = item.icon;
    const to = item.to ?? '/';
    return {
      id: `page:${to}`,
      title: t(item.label),
      icon: <Icon className={icon} />,
      run: () => void navigate(to),
    };
  });

  const panelItems: Command[] = panels
    .filter((p) => p.id > 0)
    .map((p) => {
      const kind = panelKind(p);
      return {
        id: `panel:${p.id}`,
        title: t('actionUsePanel', { name: p.name }),
        keywords: `${kind ?? ''} ${p.db ?? ''}`,
        icon: kind ? <PanelMark kind={kind} size={18} /> : null,
        run: () => select(p.id),
      };
    });

  const settings: Command[] = [
    {
      id: 'settings',
      title: t('settings'),
      icon: <GearSixIcon className={icon} />,
      run: () => void navigate(settingsPath),
    },
    {
      id: 'settings:safe-mode',
      title: t('safeModeSettings'),
      keywords: 'safe mode backup rollback حالت امن',
      icon: <ShieldCheckIcon className={icon} />,
      run: () => void navigate(`${settingsPath}/safe-mode`),
    },
  ];

  const actions: Command[] = [
    {
      id: 'scan',
      title: t('actionScan'),
      icon: <ArrowsClockwiseIcon className={icon} />,
      run: scan,
    },
    {
      id: 'shortcuts',
      title: t('actionShortcuts'),
      icon: <KeyboardIcon className={icon} />,
      run: onShortcuts,
    },
    {
      id: 'signout',
      title: t('actionSignOut'),
      icon: <SignOutIcon className={icon} />,
      run: signOut,
    },
  ];

  const preferences: Command[] = [
    {
      id: 'theme:light',
      title: t('actionThemeLight'),
      icon: <SunIcon className={icon} />,
      run: () => setPref('light'),
    },
    {
      id: 'theme:dark',
      title: t('actionThemeDark'),
      icon: <MoonIcon className={icon} />,
      run: () => setPref('dark'),
    },
    {
      id: 'theme:system',
      title: t('actionThemeSystem'),
      icon: <DesktopIcon className={icon} />,
      run: () => setPref('system'),
    },
    ...locales
      .filter((l) => l !== locale)
      .map((l) => ({
        id: `lang:${l}`,
        title: t('actionLanguage', { language: languageNames[l] }),
        keywords: 'language زبان farsi persian english',
        icon: <TranslateIcon className={icon} />,
        run: () => setLocale(l),
      })),
  ];

  return [
    { id: 'pages', label: t('groupPages'), items: pages },
    { id: 'panels', label: t('groupPanels'), items: panelItems },
    { id: 'settings', label: t('groupSettings'), items: settings },
    { id: 'actions', label: t('groupActions'), items: actions },
    { id: 'preferences', label: t('groupPreferences'), items: preferences },
  ].filter((g) => g.items.length > 0);
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
