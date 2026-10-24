import {
  ArrowsClockwiseIcon,
  DesktopIcon,
  GearSixIcon,
  HardDrivesIcon,
  KeyboardIcon,
  MoonIcon,
  PlusIcon,
  ShieldCheckIcon,
  SignOutIcon,
  SunIcon,
  TranslateIcon,
} from '@phosphor-icons/react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelSearchNames } from '../../components/panels/PanelLogo';
import { useCurrentPanel } from '../../features/panels/current-panel';
import { usePanels, useScanPanels } from '../../features/panels/use-panels';
import { newMigrationPath } from '../../features/wizard/steps';
import { useSignOut } from '../../features/session/use-session';
import { locales } from '../../i18n/core';
import { useLocale, useT } from '../../i18n/locale';
import { languageNames } from '../../i18n/shared';
import { panelKind, panelRelease } from '../../lib/api/panels';
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

  // Each panel opens its page, and can become the panel the other pages work on.
  // Both rows carry the panel's logo, as every panel name does.
  const panelItems: Command[] = panels
    .filter((p) => p.id > 0)
    .flatMap((p) => {
      const kind = panelKind(p);
      const mark = kind ? <PanelMark kind={kind} size={18} /> : <HardDrivesIcon className={icon} />;
      const keywords = `${kind ? panelSearchNames[kind] : ''} ${panelRelease(p.revision) ?? ''} ${p.db ?? ''}`;
      return [
        {
          id: `panel:${String(p.id)}`,
          title: p.name,
          keywords,
          icon: mark,
          run: () => void navigate(`/panels/${String(p.id)}`),
        },
        {
          id: `panel:${String(p.id)}:use`,
          title: t('actionUsePanel', { name: p.name }),
          keywords,
          icon: mark,
          run: () => select(p.id),
        },
      ];
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
      id: 'migration:new',
      title: t('actionNewMigration'),
      keywords: 'migrate move wizard انتقال',
      icon: <PlusIcon className={icon} />,
      run: () => void navigate(newMigrationPath),
    },
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
