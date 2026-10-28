import type { RecentPage } from '../../app/shell/recent-pages';
/** The shell's word for each section a recent page sits in. */
export const recentSections = {
  analytics: 'navAnalytics',
  panels: 'navPanels',
  migrations: 'navMigrations',
  batches: 'navBatches',
  rules: 'navRules',
  backups: 'navBackups',
  settings: 'settings',
} as const;
/** The overview's word for each settings tab. */
export const recentTab = {
  general: 'recentGeneral',
  'safe-mode': 'recentSafeMode',
  access: 'recentAccess',
  'two-step': 'recentTwoStep',
  sessions: 'recentSessions',
  'audit-log': 'recentAudit',
} as const;
/** The panel a recent page is about, if any. */
export function recentPanelOf(page: RecentPage): number | undefined {
  return page.section === 'analytics' || page.section === 'panels' ? page.panelId : undefined;
}
