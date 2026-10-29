import { Link } from 'react-router';
import { FactList } from '../../components/layout/FactList';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { pendingOf, type LinkSite } from '../../lib/api/rules';
import { panelKind, type Panel } from '../../lib/api/panels';
import { CertFactsView } from '../certificates/CertFactsView';
import { mono, Part, RuleCardFrame } from './card-parts';
import { rulesMessages } from './messages';
import { NewLinks } from './NewLinks';
import { SiteUsers } from './SiteUsers';
import { TokenUpload } from './TokenUpload';

const linkClass =
  'text-kumo-default underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** The closed card's one line: the domains, HTTPS or plain HTTP, proven and pending users. */
function Summary({ site, tls }: { site: LinkSite; tls: boolean }) {
  const t = useT(rulesMessages);
  const domains = site.domains ?? [];
  const stats = site.stats;
  return (
    <p className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
      <span className="min-w-0 truncate">
        {domains.length ? mono(domains.join(', ')) : t('everyDomain')}
      </span>
      <span>{mono(`:${String(site.port)}/${site.oldPath}/`)}</span>
      <span>{tls ? t('https') : t('plainHttp')}</span>
      {stats && stats.moved > 0 ? (
        <>
          <span className="tabular">
            {t('proven', { verified: stats.verified, moved: stats.moved })}
          </span>
          <span className="tabular">{t('pending', { count: pendingOf(stats) })}</span>
        </>
      ) : null}
    </p>
  );
}
/**
 * One panel rule, closed to its summary; its title row opens the details in place: what it
 * matches, its certificate, its users, the old links file and the new links.
 */
export function SiteCard({
  site,
  panel,
  domains,
  onDelete,
}: {
  site: LinkSite;
  panel: Panel | undefined;
  domains: readonly Domain[];
  onDelete: () => void;
}) {
  const t = useT(rulesMessages);
  const to = panel ? panelKind(panel) : undefined;
  const names = site.domains ?? [];
  const tls = site.https;
  const facts = [
    site.batches.length
      ? {
          label: t('batches'),
          value: (
            <span className="flex flex-col gap-0.5">
              {site.batches.map((b) => (
                <Link key={b.id} to={`/batches/${String(b.id)}`} className={linkClass}>
                  <bdi>{b.name}</bdi>
                </Link>
              ))}
            </span>
          ),
        }
      : { label: t('oldPanel'), value: mono(site.sourceUrl) },
    { label: t('domains'), value: names.length ? mono(names.join(', ')) : t('everyDomain') },
    { label: t('oldPath'), value: mono(`/${site.oldPath}/`) },
    { label: t('port'), value: mono(String(site.port)) },
    { label: t('answersOver'), value: tls ? t('https') : t('plainHttp') },
    { label: t('lazy'), value: site.lazy ? t('on') : t('off') },
  ];
  return (
    <RuleCardFrame
      name={site.name}
      badge={
        <>
          {to ? (
            <MoveLabel
              from={site.sourceKind}
              to={to}
              label={t('move', { from: panelNames[site.sourceKind], to: panelNames[to] })}
            />
          ) : null}
          {panel ? (
            <bdi className="min-w-0 truncate text-sm text-kumo-subtle">{panel.name}</bdi>
          ) : null}
        </>
      }
      summary={<Summary site={site} tls={tls} />}
      rule={site}
      domains={domains}
      to={`/rules/${String(site.id)}`}
      onDelete={onDelete}
    >
      <div className="grid divide-kumo-hairline border-t border-kumo-hairline lg:grid-cols-2 lg:divide-x rtl:lg:divide-x-reverse">
        <div className="flex flex-col divide-y divide-kumo-hairline">
          <div className="px-4 py-1">
            <FactList facts={facts} />
          </div>
          {site.cert ? (
            <Part title={t('certTitle')}>
              <CertFactsView facts={site.cert} />
            </Part>
          ) : null}
        </div>
        <div className="flex flex-col divide-y divide-kumo-hairline border-t border-kumo-hairline lg:border-t-0">
          <Part title={t('usersTitle')}>
            <SiteUsers stats={site.stats} />
          </Part>
          <Part title={t('uploadTitle')}>
            <TokenUpload siteId={site.id} />
          </Part>
          <Part title={t('newLinksTitle')}>
            <NewLinks site={site} />
          </Part>
        </div>
      </div>
    </RuleCardFrame>
  );
}
