import {
  ArchiveIcon,
  ArrowsLeftRightIcon,
  ChartLineIcon,
  FileTextIcon,
  HardDrivesIcon,
  LinkSimpleIcon,
  SquaresFourIcon,
  StackIcon,
  type Icon,
} from '@phosphor-icons/react';
import type { ShellKey } from './messages';

export interface NavItem {
  label: ShellKey;
  icon: Icon;
  /** Route path; absent while the page is not built yet. */
  to?: string;
  /** The `g` shortcut letter, as a physical key code. */
  key?: string;
}

export interface NavGroup {
  label?: ShellKey;
  items: NavItem[];
}

/** Sidebar structure. Settings live in the account menu, as in the dashboard. */
export const nav: NavGroup[] = [
  {
    items: [
      { label: 'navOverview', icon: SquaresFourIcon, to: '/', key: 'KeyO' },
      { label: 'navAnalytics', icon: ChartLineIcon, to: '/analytics', key: 'KeyA' },
    ],
  },
  {
    label: 'groupMove',
    items: [
      { label: 'navMigrations', icon: ArrowsLeftRightIcon, to: '/migrations', key: 'KeyM' },
      { label: 'navReports', icon: FileTextIcon, to: '/reports', key: 'KeyR' },
      { label: 'navTemplates', icon: StackIcon, to: '/templates', key: 'KeyT' },
      { label: 'navLinks', icon: LinkSimpleIcon, to: '/links', key: 'KeyL' },
    ],
  },
  {
    label: 'groupServer',
    items: [
      { label: 'navPanels', icon: HardDrivesIcon, to: '/panels', key: 'KeyP' },
      { label: 'navBackups', icon: ArchiveIcon, key: 'KeyB' },
    ],
  },
];

/** Pages that `g` and the palette can reach, besides the sidebar ones. */
export const settingsPath = '/settings';
