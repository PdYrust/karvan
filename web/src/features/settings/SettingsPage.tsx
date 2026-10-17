import { Outlet } from 'react-router';
import { PageHeader } from '../../components/layout/PageHeader';
import { RouteTabs } from '../../components/layout/RouteTabs';
import { usePageTitle } from '../../components/layout/use-page-title';
import { useT } from '../../i18n/locale';
import { settingsMessages } from './messages';

/** Settings and security on one page with a tab per part, opened from the account menu. */
export function SettingsPage() {
  const t = useT(settingsMessages);
  usePageTitle(t('title'));
  return (
    <div className="flex flex-col">
      <PageHeader
        title={t('title')}
        description={t('description')}
        tabs={
          <RouteTabs
            label={t('tabs')}
            tabs={[
              { to: '/settings', label: t('tabGeneral'), end: true },
              { to: '/settings/safe-mode', label: t('tabSafeMode') },
              { to: '/settings/access', label: t('tabAccess'), soon: true },
              { to: '/settings/two-step', label: t('tabTwoStep'), soon: true },
              { to: '/settings/sessions', label: t('tabSessions'), soon: true },
              { to: '/settings/audit-log', label: t('tabAudit'), soon: true },
            ]}
          />
        }
      />
      <div className="flex max-w-[1400px] flex-col gap-6 p-4 sm:p-8">
        <Outlet />
      </div>
    </div>
  );
}
