import {
  ArchiveIcon,
  ArrowsLeftRightIcon,
  ClockCounterClockwiseIcon,
  FileTextIcon,
  GearSixIcon,
  HardDrivesIcon,
  LockKeyIcon,
  ShieldCheckIcon,
  SquaresFourIcon,
  type Icon,
} from '@phosphor-icons/react';
import type { ShellKey } from './messages';

export interface NavItem {
  label: ShellKey;
  icon: Icon;
  /** Route path; absent while the page is not built yet. */
  to?: string;
}

export interface NavGroup {
  label?: ShellKey;
  items: NavItem[];
}

/** Sidebar structure. Pages without `to` show as "Soon" until they land. */
export const nav: NavGroup[] = [
  { items: [{ label: 'navOverview', icon: SquaresFourIcon, to: '/' }] },
  {
    label: 'groupMove',
    items: [
      { label: 'navMigrations', icon: ArrowsLeftRightIcon },
      { label: 'navReports', icon: FileTextIcon },
    ],
  },
  {
    label: 'groupProtect',
    items: [
      { label: 'navBackups', icon: ArchiveIcon },
      { label: 'navSafeMode', icon: ShieldCheckIcon },
    ],
  },
  {
    label: 'groupServer',
    items: [
      { label: 'navPanels', icon: HardDrivesIcon },
      { label: 'navActivity', icon: ClockCounterClockwiseIcon },
      { label: 'navSecurity', icon: LockKeyIcon },
      { label: 'navSettings', icon: GearSixIcon },
    ],
  },
];
