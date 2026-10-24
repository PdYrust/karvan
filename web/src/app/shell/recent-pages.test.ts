import { describe, expect, it } from 'vitest';
import { recentPage, recentPath } from './recent-pages';

describe('recentPage', () => {
  it('names the pages the overview lists', () => {
    expect(recentPage('/analytics/server')).toEqual({ section: 'analytics' });
    expect(recentPage('/analytics/panels/2')).toEqual({ section: 'analytics', panelId: 2 });
    expect(recentPage('/panels')).toEqual({ section: 'panels' });
    expect(recentPage('/panels/12')).toEqual({ section: 'panels', panelId: 12 });
    expect(recentPage('/migrations/new')).toEqual({ section: 'migrations', page: 'new' });
    expect(recentPage('/settings')).toEqual({ section: 'settings', tab: 'general' });
    expect(recentPage('/settings/safe-mode')).toEqual({ section: 'settings', tab: 'safe-mode' });
  });

  it('reads a trailing slash as the same page', () => {
    expect(recentPage('/panels/')).toEqual({ section: 'panels' });
    expect(recentPath('/panels/2/')).toBe('/panels/2');
    expect(recentPath('/')).toBeUndefined();
  });

  it('leaves out the overview, redirects and pages that do not exist', () => {
    for (const path of [
      '/',
      '',
      '/analytics',
      '/panels/0',
      '/panels/x',
      '/panels/2/more',
      '/settings/access',
      '/nope',
    ]) {
      expect(recentPage(path)).toBeUndefined();
    }
  });
});
