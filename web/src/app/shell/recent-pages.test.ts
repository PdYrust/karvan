import { describe, expect, it } from 'vitest';
import { recentPage, recentPath } from './recent-pages';

describe('recentPage', () => {
  it('names the pages the overview lists', () => {
    expect(recentPage('/analytics/server')).toEqual({ section: 'analytics' });
    expect(recentPage('/analytics/panels/2')).toEqual({ section: 'analytics', panelId: 2 });
    expect(recentPage('/panels')).toEqual({ section: 'panels' });
    expect(recentPage('/panels/12')).toEqual({ section: 'panels', panelId: 12 });
    expect(recentPage('/migrations')).toEqual({ section: 'migrations' });
    expect(recentPage('/migrations/new')).toEqual({ section: 'migrations', page: 'new' });
    expect(recentPage('/batches')).toEqual({ section: 'batches' });
    expect(recentPage('/rules')).toEqual({ section: 'rules' });
    expect(recentPage('/rules/new')).toEqual({ section: 'rules', page: 'new' });
    expect(recentPage('/backups')).toEqual({ section: 'backups' });
    expect(recentPage('/settings')).toEqual({ section: 'settings', tab: 'general' });
    expect(recentPage('/settings/safe-mode')).toEqual({ section: 'settings', tab: 'safe-mode' });
    expect(recentPage('/settings/access')).toEqual({ section: 'settings', tab: 'access' });
    expect(recentPage('/settings/audit-log')).toEqual({ section: 'settings', tab: 'audit-log' });
  });

  it('reads a trailing slash as the same page', () => {
    expect(recentPage('/panels/')).toEqual({ section: 'panels' });
    expect(recentPath('/panels/2/')).toBe('/panels/2');
    expect(recentPath('/')).toBeUndefined();
  });

  it('leaves out the overview, redirects and pages that do not exist or were removed', () => {
    for (const path of [
      '/',
      '',
      '/analytics',
      '/panels/0',
      '/panels/x',
      '/panels/2/more',
      '/settings/billing',
      '/migrations/9f1c2b3a4d5e6f7081920a1b',
      '/reports',
      '/reports/9f1c2b3a4d5e6f7081920a1b',
      '/migrations/9f1c2b3a4d5e6f7081920a1b/report',
      '/rules/3',
      '/links',
      '/links/new',
      '/links/3',
      '/batches/3',
      '/templates',
      '/nope',
    ]) {
      expect(recentPage(path)).toBeUndefined();
    }
  });
});
