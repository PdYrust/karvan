import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { DotsThreeIcon, PencilSimpleIcon, TrashIcon } from '@phosphor-icons/react';
import { useId, type ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { FactList } from '../../components/layout/FactList';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import type { LinkSite } from '../../lib/api/links';
import { panelKind, type Panel } from '../../lib/api/panels';
import { CertFactsView } from './CertFactsView';
import { linksMessages } from './messages';
import { NewLinks } from './NewLinks';
import { SiteUsers } from './SiteUsers';
import { TokenUpload } from './TokenUpload';

function SiteMenu({ site, onDelete }: { site: LinkSite; onDelete: () => void }) {
  const t = useT(linksMessages);
  const navigate = useNavigate();
  const label = t('moreFor', { name: site.name });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} weight="bold" />}
            className="shrink-0 pointer-coarse:size-9"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-44">
        <DropdownMenu.Item
          className="gap-2"
          onClick={() => void navigate(`/links/${String(site.id)}`)}
        >
          <PencilSimpleIcon size={16} aria-hidden="true" />
          {t('edit')}
        </DropdownMenu.Item>
        <DropdownMenu.Item className="gap-2" variant="danger" onClick={onDelete}>
          <TrashIcon size={16} aria-hidden="true" />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

/** A titled part of the card, split from the next by a hairline. */
function Part({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2 px-4 py-3">
      <h3 id={id} className="text-sm font-medium text-kumo-strong">
        {title}
      </h3>
      {children}
    </section>
  );
}

const mono = (text: string) => (
  <bdi dir="ltr" className="font-mono text-[0.9em] break-all">
    {text}
  </bdi>
);

/** One site: where it answers, its certificate, its users, the old links file and the new links. */
export function SiteCard({
  site,
  panel,
  onDelete,
}: {
  site: LinkSite;
  panel: Panel | undefined;
  onDelete: () => void;
}) {
  const t = useT(linksMessages);
  const id = useId();
  const to = panel ? panelKind(panel) : undefined;
  const domains = site.domains ?? [];
  const tls = Boolean(site.certFile || site.certPem);
  const facts = [
    { label: t('oldPanel'), value: mono(site.sourceUrl) },
    { label: t('domains'), value: domains.length ? mono(domains.join(', ')) : t('everyDomain') },
    { label: t('oldPath'), value: mono(`/${site.oldPath}/`) },
    { label: t('port'), value: mono(String(site.port)) },
    { label: t('answersOver'), value: tls ? t('https') : t('plainHttp') },
    { label: t('lazy'), value: site.lazy ? t('on') : t('off') },
  ];
  return (
    <article aria-labelledby={id} className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
      <header className="flex items-start gap-3 border-b border-kumo-hairline py-2 ps-4 pe-2">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1 py-1.5">
          <h2 id={id} className="min-w-0 font-semibold text-kumo-strong">
            <bdi>{site.name}</bdi>
          </h2>
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
        </div>
        <SiteMenu site={site} onDelete={onDelete} />
      </header>
      <div className="grid divide-kumo-hairline lg:grid-cols-2 lg:divide-x rtl:lg:divide-x-reverse">
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
    </article>
  );
}
