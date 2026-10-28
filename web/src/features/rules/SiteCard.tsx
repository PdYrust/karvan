import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import {
  CaretDownIcon,
  CaretRightIcon,
  DotsThreeIcon,
  PencilSimpleIcon,
  TrashIcon,
} from '@phosphor-icons/react';
import { useId, useState, type ReactNode } from 'react';
import { Collapse } from '../../components/layout/Collapse';
import { useNavigate } from 'react-router';
import { FactList } from '../../components/layout/FactList';
import { MoveLabel } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import { pendingOf, type LinkSite } from '../../lib/api/rules';
import { panelKind, type Panel } from '../../lib/api/panels';
import { CertFactsView } from '../certificates/CertFactsView';
import { rulesMessages } from './messages';
import { NewLinks } from './NewLinks';
import { SiteUsers } from './SiteUsers';
import { TokenUpload } from './TokenUpload';

function SiteMenu({ site, onDelete }: { site: LinkSite; onDelete: () => void }) {
  const t = useT(rulesMessages);
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
          onClick={() => void navigate(`/rules/${String(site.id)}`)}
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
 * One site, closed to its summary; its title row opens the details in place: where it
 * answers, its certificate, its users, the old links file and the new links.
 */
export function SiteCard({
  site,
  panel,
  onDelete,
}: {
  site: LinkSite;
  panel: Panel | undefined;
  onDelete: () => void;
}) {
  const t = useT(rulesMessages);
  const id = useId();
  const [open, setOpen] = useState(false);
  const Caret = open ? CaretDownIcon : CaretRightIcon;
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
      {/* The name's button covers the whole row, so a tap anywhere on it opens or closes. */}
      <header className="relative flex items-start gap-3 py-2 ps-4 pe-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1 py-1.5">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <h2 id={id} className="min-w-0 font-semibold text-kumo-strong">
              <button
                type="button"
                aria-expanded={open}
                aria-controls={`${id}-details`}
                onClick={() => setOpen((o) => !o)}
                className="text-start outline-none after:absolute after:inset-0 after:rounded-t-lg after:content-[''] focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-kumo-focus"
              >
                <bdi>{site.name}</bdi>
              </button>
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
          <Summary site={site} tls={tls} />
        </div>
        <Caret
          size={16}
          aria-hidden="true"
          className="mt-2.5 shrink-0 text-kumo-subtle rtl:-scale-x-100"
        />
        <div className="relative">
          <SiteMenu site={site} onDelete={onDelete} />
        </div>
      </header>
      <Collapse id={`${id}-details`} open={open}>
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
      </Collapse>
    </article>
  );
}
