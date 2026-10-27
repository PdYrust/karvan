export type SettingsTab =
  'general' | 'safe-mode' | 'access' | 'two-step' | 'sessions' | 'audit-log';

/** A page the overview can list under Recents; its words come from the page's language at show time. */
export type RecentPage =
  | { section: 'analytics'; panelId?: number }
  | { section: 'panels'; panelId?: number }
  | { section: 'migrations'; page?: 'new' }
  | { section: 'reports'; jobId?: string }
  | { section: 'links'; page?: 'new' }
  | { section: 'backups' }
  | { section: 'settings'; tab: SettingsTab };

function withPanel(section: 'analytics' | 'panels', id: string | undefined) {
  const panelId = Number(id);
  return panelId > 0 ? { section, panelId } : undefined;
}

// Built pages only; /analytics itself is left out, since it moves on to its server tab at once.
const routes: [RegExp, (m: RegExpExecArray) => RecentPage | undefined][] = [
  [/^\/analytics\/server$/, () => ({ section: 'analytics' })],
  [/^\/analytics\/panels\/(\d+)$/, (m) => withPanel('analytics', m[1])],
  [/^\/panels$/, () => ({ section: 'panels' })],
  [/^\/panels\/(\d+)$/, (m) => withPanel('panels', m[1])],
  [/^\/migrations$/, () => ({ section: 'migrations' })],
  [/^\/migrations\/new$/, () => ({ section: 'migrations', page: 'new' })],
  [/^\/reports$/, () => ({ section: 'reports' })],
  [/^\/reports\/([0-9a-f]{24})$/, (m) => ({ section: 'reports', jobId: m[1] })],
  [/^\/links$/, () => ({ section: 'links' })],
  [/^\/links\/new$/, () => ({ section: 'links', page: 'new' })],
  [/^\/backups$/, () => ({ section: 'backups' })],
  [/^\/settings$/, () => ({ section: 'settings', tab: 'general' })],
  [
    /^\/settings\/(safe-mode|access|two-step|sessions|audit-log)$/,
    (m) => ({ section: 'settings', tab: m[1] as SettingsTab }),
  ],
];

const trim = (path: string) => path.replace(/(.)\/+$/, '$1');

/** The page a path opens, or undefined for the overview and paths that are not pages. */
export function recentPage(path: string): RecentPage | undefined {
  const clean = trim(path);
  for (const [pattern, page] of routes) {
    const m = pattern.exec(clean);
    if (m) return page(m);
  }
  return undefined;
}

/** The path to remember for an address the operator opened, if it is such a page. */
export function recentPath(pathname: string): string | undefined {
  return recentPage(pathname) ? trim(pathname) : undefined;
}
