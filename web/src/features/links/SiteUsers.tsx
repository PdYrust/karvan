import { Bar } from '../../components/charts/Bar';
import { useT } from '../../i18n/locale';
import { pendingOf, type SiteStats } from '../../lib/api/links';
import { linksMessages } from './messages';

/** Proven users over moved ones, with pending and closed as quiet numbers. */
export function SiteUsers({ stats }: { stats: SiteStats | undefined }) {
  const t = useT(linksMessages);
  if (!stats) return <p className="text-sm text-kumo-subtle">{t('statsFailed')}</p>;
  if (stats.moved === 0) return <p className="text-sm text-kumo-subtle">{t('noUsers')}</p>;
  const pending = pendingOf(stats);
  return (
    <div className="flex flex-col gap-2">
      <p className="font-medium text-kumo-strong tabular">
        {t('proven', { verified: stats.verified, moved: stats.moved })}
      </p>
      <Bar grow share={stats.verified / stats.moved} />
      <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-kumo-subtle tabular">
        <span>{t('pending', { count: pending })}</span>
        {stats.closed > 0 ? <span>{t('closed', { count: stats.closed })}</span> : null}
      </p>
      {pending > 0 ? <p className="text-sm text-kumo-subtle">{t('usersHint')}</p> : null}
    </div>
  );
}
