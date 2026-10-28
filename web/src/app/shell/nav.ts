import {
  ArchiveIcon,
  ArrowsLeftRightIcon,
  ChartLineIcon,
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
  /** Route path. */
  to: string;
  /** The `g` shortcut letter, as a physical key code. */
  key: string;
  /** Sub-pages under the row: this server, then one per panel. */
  perPanel?: boolean;
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
      { label: 'navAnalytics', icon: ChartLineIcon, to: '/analytics', key: 'KeyA', perPanel: true },
    ],
  },
  {
    label: 'groupMove',
    items: [
      { label: 'navMigrations', icon: ArrowsLeftRightIcon, to: '/migrations', key: 'KeyM' },
      { label: 'navBatches', icon: StackIcon, to: '/batches', key: 'KeyH' },
      { label: 'navLinks', icon: LinkSimpleIcon, to: '/links', key: 'KeyL' },
    ],
  },
  {
    label: 'groupServer',
    items: [
      { label: 'navPanels', icon: HardDrivesIcon, to: '/panels', key: 'KeyP' },
      { label: 'navBackups', icon: ArchiveIcon, to: '/backups', key: 'KeyB' },
    ],
  },
];

/** The page that `g s` and the palette reach besides the sidebar ones. */
export const settingsPath = '/settings';
